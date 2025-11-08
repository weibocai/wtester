package request

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	netUrl "net/url"
	"strings"
	"time"
)

var httpClientPool = make(map[string]*http.Client)

// RegisterHttpClient 注册http客户端，用于复用，在编辑task的时候，就开始注册
func RegisterHttpClient(url string) (*http.Client, error) {
	u, err := netUrl.Parse(url)
	if err != nil {
		return nil, err
	}
	host := fmt.Sprintf("%s//:%s", u.Scheme, u.Host)
	if c, ok := httpClientPool[host]; ok {
		return c, nil
	}
	transport := &http.Transport{
		// 连接池相关配置
		MaxIdleConns:        100,              // 最大空闲连接数
		MaxIdleConnsPerHost: 100,              // 每个host的最大空闲连接数
		MaxConnsPerHost:     100,              // 每个host的最大连接数
		IdleConnTimeout:     10 * time.Second, // 空闲连接超时时间

		// TLS配置
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},

		// 拨号相关配置
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // 连接超时时间
			KeepAlive: 10 * time.Second, // 保持连接存活时间
		}).DialContext,

		// 其他配置
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	c := &http.Client{
		Transport: transport,
	}
	httpClientPool[host] = c
	return c, nil
}

// GetHtpClient 获取客户端，支持url or host 获取客户端
func GetHtpClient(url string) *http.Client {
	if c, ok := httpClientPool[url]; ok {
		return c
	}
	c, _ := RegisterHttpClient(url)
	return c
}

// CloseHtpClient 关闭客户端连接
func CloseHtpClient() {
	for _, c := range httpClientPool {
		c.CloseIdleConnections()
	}
}

// 组建get请求
func makeGet(url netUrl.URL, params map[string]any) (string, error) {
	paramsUrl := netUrl.Values{}
	for k, v := range params {
		paramsUrl.Set(k, fmt.Sprintf("%v", v))
	}
	url.RawQuery = paramsUrl.Encode()
	return url.String(), nil
}

// HttpRequest http 请求工具
func HttpRequest(method string, url string, params map[string]any) (string, error) {
	Url, err := netUrl.Parse(url)
	if err != nil {
		return "", err
	}
	client := GetHtpClient(fmt.Sprintf("%s//:%s", Url.Scheme, Url.Host))
	if client == nil {
		return "", fmt.Errorf("未找到可以执行的客户端：%s", url)
	}

	var paramsByte []byte
	method = strings.ToUpper(method)
	if method == "GET" {
		url, err = makeGet(*Url, params)
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
