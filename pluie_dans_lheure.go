package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"
)

const STATUS_LEN int = 9

func get_bearer() (string, error) {
	var url string = "https://meteofrance.com/previsions-meteo-france/"

	clt := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	req, errHttp := http.NewRequest(http.MethodHead, url, nil)
	if errHttp != nil {
		return "", errHttp
	}
	req.Header.Set("User-Agent", "PluieDansLheureFori3status")

	res, errGet := clt.Do(req)
	if errGet != nil {
		return "", errGet
	}

	setcookie := res.Header.Get("Set-Cookie")
	if len(setcookie) < 20 {
		return "", errors.New("set cookie string too short")
	}
	if !strings.HasPrefix(setcookie, "mfsession=") {
		return "", errors.New("set cookie string does not start with mfsession")
	}
	setcookie = strings.TrimPrefix(setcookie, "mfsession=")
	idx := strings.IndexByte(setcookie, ';')
	if idx < 0 {
		return "", errors.New("set cookie string does not have a ';'")
	}
	buf := bytes.NewBufferString("Bearer ")
	/* Yes, that's ROT13 … o\ */
	for i := 0; i < idx; i++ {
		c := setcookie[i]
		if c >= 'a' && c <= 'm' || c >= 'A' && c <= 'M' {
			c += 13
		} else if c >= 'n' && c <= 'z' || c >= 'N' && c <= 'Z' {
			c -= 13
		}
		buf.WriteByte(c)
	}
	return buf.String(), nil
}

func get_status_from_http(location *string) (string, error) {
	var url string = "https://rpcache-aa.meteofrance.com/internet2018client/2.0/nowcast/rain?" + *location

	bearer, errBearer := get_bearer()
	if errBearer != nil {
		return "", errBearer
	}

	clt := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	req, errHttp := http.NewRequest(http.MethodGet, url, nil)
	if errHttp != nil {
		return "", errHttp
	}

	req.Header.Set("User-Agent", "PluieDansLheureFori3status")
	req.Header.Set("Authorization", bearer)

	res, errGet := clt.Do(req)
	if errGet != nil {
		return "", errGet
	}

	body, errRead := ioutil.ReadAll(res.Body)
	if errRead != nil {
		return "", errRead
	}

	var data map[string]interface{}
	errJson := json.Unmarshal(body, &data)
	if errJson != nil {
		return "", errJson
	}
	var dataProperties, okP = data["properties"].(map[string]interface{})
	if !okP {
		return "", errors.New("can not find 'properties' in JSON")
	}
	var dataForecast, okF = dataProperties["forecast"].([]interface{})
	if !okF {
		return "", errors.New("can not find 'forecast' in 'properties' in JSON")
	}
	var b strings.Builder
	b.Grow(STATUS_LEN * 4 /* to account for unicode character */)
	for _, v := range dataForecast {
		var rain, ok = v.(map[string]interface{})
		if !ok {
			return "", errors.New("invalid element in 'properties.forecast' in JSON")
		}
		var intensity, ok2 = rain["rain_intensity"].(float64)
		if !ok2 {
			return "", errors.New("'rain_intensity' not found in sub element in JSON")
		}
		switch {
		case intensity <= 1:
			b.WriteRune('_')
		case intensity <= 2:
			b.WriteRune('░')
		case intensity <= 3:
			b.WriteRune('▒')
		case intensity <= 4:
			b.WriteRune('▓')
		default:
			b.WriteRune('█')
		}
	}

	return b.String(), nil
}

func read_status_from_file_no_lock(f *os.File, file_length int64) (string, error) {
	var buf = make([]byte, file_length) // to account for unicode characters
	var _, err = f.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func write_status_to_file_no_lock(f *os.File, status string) error {
	var truncateErr = f.Truncate(0)
	if truncateErr != nil {
		return truncateErr
	}
	var _, writeErr = f.WriteString(status)
	if writeErr != nil {
		return writeErr
	}
	var syncErr = f.Sync()
	return syncErr
}

func need_new_status(f *os.File, location *string) (string, error) {
	var status, err = get_status_from_http(location)
	if err != nil {
		var file_path string = "/tmp/pluie_dans_lheure.err"

		var f, openErr = os.OpenFile(file_path, os.O_RDWR|os.O_CREATE, 0644)
		if openErr != nil {
			return "", nil
		}
		defer f.Close()
		f.WriteString(err.Error())
		return "", err
	}
	var writeErr = write_status_to_file_no_lock(f, status)
	if writeErr != nil {
		return "", writeErr
	}
	return status, nil
}

func GetRain(location *string) (string, error) {
	var file_path string = "/tmp/pluie_dans_lheure"

	var f, openErr = os.OpenFile(file_path, os.O_RDWR|os.O_CREATE, 0644)
	if openErr != nil {
		return "", nil
	}
	defer f.Close()
	var fd = f.Fd()

	/* Flock */
	var flockErr = syscall.Flock(int(fd), syscall.LOCK_EX)
	if flockErr != nil {
		return "", nil
	}
	defer syscall.Flock(int(fd), syscall.LOCK_UN)

	var st, statErr = f.Stat()
	if statErr != nil {
		return "", nil
	}
	var t_hour, t_min, _ int = st.ModTime().Clock()
	var now_hour, now_min, _ int = time.Now().Clock()
	if st.Size() < int64(STATUS_LEN) || t_hour != now_hour || t_min/5 != now_min/5 {
		return need_new_status(f, location)
	}
	return read_status_from_file_no_lock(f, st.Size())
}

func GetRainI3barFormat(location *string, rain_color *string) (block I3ProtocolBlock, err error) {
	status, err := GetRain(location)
	if err != nil {
		return
	}
	block.FullText = status
	if status != "____________" {
		block.Color = *rain_color
	}
	return
}
