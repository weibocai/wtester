package request

import (
	"fmt"
	"net/http"
	"net/http/httptrace"
	netUrl "net/url"
	"sync"
	"testing"
	"time"
)

type ConnectionTracer struct {
	start time.Time
}

func (t *ConnectionTracer) trace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			fmt.Printf("Getting connection: %s\n", hostPort)
			t.start = time.Now()
		},
		GotConn: func(info httptrace.GotConnInfo) {
			fmt.Printf("Got connection: reused=%v, idle=%v, time=%v\n",
				info.Reused, info.WasIdle, time.Since(t.start))
		},
		ConnectStart: func(network, addr string) {
			fmt.Printf("Dial start: %s %s\n", network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			fmt.Printf("Dial complete: %s %s, err=%v\n", network, addr, err)
		},
	}
}

func TestHttpClientPool(t *testing.T) {
	url := "http://127.0.0.1:9090/temp"
	_, _ = RegisterHttpClient(url)
	tracer := &ConnectionTracer{}
	Url, _ := netUrl.Parse(url)
	client := GetHtpClient(fmt.Sprintf("%s//:%s", Url.Scheme, Url.Host))
	var group sync.WaitGroup
	for i := 0; i <= 100; i++ {
		group.Go(func() {
			for j := 0; j <= 10; j++ {
				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					t.Errorf("连接失败：%s", err)
				}

				req = req.WithContext(httptrace.WithClientTrace(req.Context(), tracer.trace()))
				resp, err := client.Do(req)
				if err != nil {
					t.Errorf("请求失败：%s", err)
				}
				_ = resp.Body.Close()
			}
		})
	}
	group.Wait()
	CloseHtpClient()
}
