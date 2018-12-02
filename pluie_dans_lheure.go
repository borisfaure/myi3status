package main
import (
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "strings"
    "time"
)

func get_status(code string) (string, error) {
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

func main() {
    status, err := get_status("920440")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("->'%v'\n", status)
}
