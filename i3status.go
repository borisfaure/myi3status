package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os/exec"
)

func readSome(scanner *bufio.Scanner) error {
	if ok := scanner.Scan(); !ok {
		if err := scanner.Err(); err != nil {
			return err
		}
		return errors.New("scanner failed")
	}
	return nil
}

func main_loop(location *string, rain_color *string) error {
	path, lookupErr := exec.LookPath("i3status")
	if lookupErr != nil {
		return lookupErr
	}
	cmd := exec.Command(path)
	stdout, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		return pipeErr
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	if err := readSome(scanner); err != nil {
		return err
	}

	header := I3ProtocolHeader{}
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil || header.Version != 1 {
		return errors.New("invalid header " + scanner.Text())
	}
	fmt.Println(scanner.Text())

	if err := readSome(scanner); err != nil {
		return err
	}
	if scanner.Text() != "[" {
		return errors.New("Invalid second line")
	}
	fmt.Println("[")

	first := true
	var chWeather = make(chan *I3ProtocolBlock)
	var chSpotify = make(chan *I3ProtocolBlock)
	for {
		if err := readSome(scanner); err != nil {
			return err
		}
		var err error
		blocks := make([]I3ProtocolBlock, 0)
		output := make([]byte, 0)
		if first {
			err = json.Unmarshal(scanner.Bytes(), &blocks)
			first = false
		} else {
			err = json.Unmarshal(scanner.Bytes()[1:], &blocks)
			output = append(output, byte(','))
		}
		if err != nil {
			return errors.New("invalid blocks")
		}

		go func() {
			weather, err := GetRainI3barFormat(location, rain_color)
			if err != nil {
				chWeather <- nil
			} else {
				chWeather <- &weather
			}
		}()

		go func() {
			playing, err := SpotifyGetCurrentPlaying()
			if err != nil {
				chSpotify <- nil
			} else {
				chSpotify <- &playing
			}
		}()

		var (
			found   = 0
			weather *I3ProtocolBlock
			playing *I3ProtocolBlock
		)
		for found < 2 {
			select {
			case weather = <-chWeather:
				found = found + 1
			case playing = <-chSpotify:
				found = found + 1
			}
		}
		if weather != nil {
			blocks = append([]I3ProtocolBlock{*weather}, blocks...)
		}
		if playing != nil {
			blocks = append([]I3ProtocolBlock{*playing}, blocks...)
		}

		data, err := json.Marshal(blocks)
		if err != nil {
			return err
		}
		fmt.Printf("%s%s\n", output, data)
	}

	return nil
}

func main() {
	location := flag.String("location", "",
		"a location for the Pluie dans l'heure API, in the form 'lat=48.859333&lon=2.340591'")
	rainColor := flag.String("rain_color", "#268bd2",
		"Color to display text when it's raining")
	flag.Parse()
	if *location == "" {
		location = nil
	}
	err := main_loop(location, rainColor)
	if err != nil {
		log.Fatal(err)
	}
}
