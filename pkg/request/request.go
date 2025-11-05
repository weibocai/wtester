package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	netUrl "net/url"
	"strings"
)

func makeGet(url string, params map[string]interface{}) (string, error) {
	Url, err := netUrl.Parse(url)
	if err != nil {
		return "", err
	}

	paramsUrl := netUrl.Values{}
	for k, v := range params {
		paramsUrl.Set(k, fmt.Sprintf("%v", v))
	}
	Url.RawQuery = paramsUrl.Encode()
	return Url.String(), nil
}

// Requests http 请求工具
func Requests(method string, url string, params map[string]interface{}) (string, error) {
	var paramsByte []byte
	var err error
	method = strings.ToUpper(method)
	if method == "GET" {
		url, err = makeGet(url, params)
		if err != nil {
			return "", err
		}
	} else {
		paramsByte, err = json.Marshal(params)
		if err != nil {
			return "", err
		}
	}

	request, err1 := http.NewRequest(method, url, bytes.NewBuffer(paramsByte))
	if err1 != nil {
		return "", err1
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err2 := client.Do(request)
	if err2 != nil {
		return "", err2
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode != 200 {
		return "", fmt.Errorf("error getting url: %s", response.Status)
	}
	if body, err4 := io.ReadAll(response.Body); err4 != nil {
		return "", err4
	} else {
		return string(body), nil
	}
}
