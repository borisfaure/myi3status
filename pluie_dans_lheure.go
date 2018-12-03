package main
import (
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "strings"
    "time"
)

func get_status_from_http(code string) (string, error) {
    var url string = "http://www.meteofrance.com/mf3-rpc-portlet/rest/pluie/" + code

    clt := http.Client{
        Timeout: time.Second * 2, // Maximum of 2 secs
    }

    req, errHttp := http.NewRequest(http.MethodGet, url, nil)
    if errHttp != nil {
        return "", errHttp
    }

    req.Header.Set("User-Agent", "PluieDansLheureFori3status")

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
    var dataCadran, ok = data["dataCadran"].([]interface{})
    if !ok {
        return "", errors.New("can not find 'dataCadran' in JSON")
    }
    var b strings.Builder
    b.Grow(12)
    for _, v := range dataCadran {
        var pluie, ok = v.(map[string]interface{})
        if !ok {
            return "", errors.New("invalid element in 'dataCadran' in JSON")
        }
        var niveau, ok2 = pluie["niveauPluie"].(float64)
        if !ok2 {
            return "", errors.New("'niveauPluie' not found in sub element in JSON")
        }
        switch {
        case niveau <= 1:
            b.WriteRune('_')
        case niveau <= 2:
            b.WriteRune('░')
        case niveau <= 3:
            b.WriteRune('▒')
        case niveau <= 4:
            b.WriteRune('▓')
        default:
            b.WriteRune('█')
        }
    }

    return b.String(), nil
}

func read_status_from_file(file_path string) (string,  error) {
    var content, err = ioutil.ReadFile(file_path)
    if err != nil {
        return "", err
    }
    return string(content), nil
}

func write_status_to_file_no_lock(file_path string, status string) (error) {
    return ioutil.WriteFile(file_path, []byte(status), 0644)
}

func need_new_status(file_path string, code string) (string , error) {
    var status, err = get_status_from_http(code)
    if err != nil {
        return "", err
    }
    var writeErr = write_status_to_file_no_lock(file_path, status)
    if writeErr != nil {
        return "", writeErr
    }
    return status, nil
}


func get_status(code string) (string, error) {
    var file_path string = "/tmp/pluie_dans_lheure." + code
    var st, err = os.Stat(file_path)
    if os.IsNotExist(err) {
        return need_new_status(file_path, code)
    }
    var t_hour, t_min, _ int = st.ModTime().Clock()
    var now_hour, now_min, _ int = time.Now().Clock()
    if t_hour != now_hour || t_min/5 != now_min/5 {
        return need_new_status(file_path, code)
    }
    return read_status_from_file(file_path)
}

func main() {
    status, err := get_status("920440")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("->'%v'\n", status)
}
