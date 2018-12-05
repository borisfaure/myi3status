package main
import (
    "encoding/json"
    "errors"
    "io/ioutil"
    "net/http"
    "os"
    "strings"
    "syscall"
    "time"
)
const STATUS_LEN int = 12

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
    b.Grow(STATUS_LEN * 4 /* to account for unicode character */)
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

func read_status_from_file_no_lock(f *os.File) (string,  error) {
    var buf = make([]byte, STATUS_LEN*4) // to account for unicode characters
    var _, err = f.Read(buf)
    if err != nil {
        return "", err
    }
    return string(buf), nil
}

func write_status_to_file_no_lock(f *os.File, status string) (error) {
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

func need_new_status(f *os.File, code string) (string , error) {
    var status, err = get_status_from_http(code)
    if err != nil {
        return "", err
    }
    var writeErr = write_status_to_file_no_lock(f, status)
    if writeErr != nil {
        return "", writeErr
    }
    return status, nil
}


func GetRain(code string) (string, error) {
    var file_path string = "/tmp/pluie_dans_lheure." + code

    var f, openErr = os.OpenFile(file_path, os.O_RDWR|os.O_CREATE, 0644)
    if openErr != nil {
        return "", nil
    }
    defer f.Close()
    var fd = f.Fd()

    /* Flock */
    var flockErr = syscall.Flock(int(fd), syscall.LOCK_EX)
    if flockErr != nil {
        return "",  nil
    }
    defer syscall.Flock(int(fd), syscall.LOCK_UN)

    var st, statErr = f.Stat()
    if statErr != nil {
        return "", nil
    }
    var t_hour, t_min, _ int = st.ModTime().Clock()
    var now_hour, now_min, _ int = time.Now().Clock()
    if st.Size() < int64(STATUS_LEN) || t_hour != now_hour || t_min/5 != now_min/5 {
        return need_new_status(f, code)
    }
    return read_status_from_file_no_lock(f)
}

func GetRainI3barFormat(code string, rain_color string) (string, error) {
    status, rainErr := GetRain(code)
    if rainErr != nil {
        return "", rainErr
    }
    /* Poor man's json encoder */
    var b strings.Builder
    b.WriteString("{\"full_text\":\"")
    b.WriteString(status)
    b.WriteString("\"")
    if status != "____________" {
        b.WriteString(",\"color\":\"")
        b.WriteString(rain_color)
        b.WriteString("\"")
    }
    b.WriteString("}")
    return b.String(), nil
}
