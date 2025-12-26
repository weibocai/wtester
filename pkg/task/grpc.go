package task

import (
	"fmt"
	"time"

	"github.com/wtester/pkg/request"
)

type GrpcRequest struct {
	HttpRequest `yaml:",inline"`
	Service     string `yaml:"service"`
	Timeout     string `yaml:"timeout"`
	MaxIdleConn int    `yaml:"maxIdleConn"`
}

func (r *GrpcRequest) AddClientPool() error {
	timeout, err := time.ParseDuration(r.Timeout)
	if err != nil {
		return err
	}
	if _, err := request.RegisterGrpcClientPool(r.Url, r.MaxIdleConn, timeout); err != nil {
		return err
	}
	return nil
}

func (r *GrpcRequest) Runner(index int) (bool, int64, error) {
	now := time.Now()
	isEqual := false
	method := fmt.Sprintf("%s.%s", r.Service, r.Method)
	if para, err := r.getParam(index); err != nil {
		return false, 0, err
	} else {
		if res, err := request.GrpcRequest(r.Url, method, para); err == nil {
			isEqual = r.compare(index, res)
		} else {
			return false, 0, err
		}
	}

	duration := time.Since(now).Milliseconds()
	return isEqual, duration, nil
}
