package task

import (
	"fmt"
	"time"

	"github.com/wtester/pkg/request"
)

type GrpcRequest struct {
	HttpRequest
	Service string `yaml:"service"`
	Method  string `yaml:"method"`
}

func (r *GrpcRequest) GetClientPoolKey() string {
	return fmt.Sprintf("%s/%s/%s", r.Url, r.Service, r.Name)
}

func (r *GrpcRequest) AddClientPool() error {
	if _, err := request.RegisterGrpcClient(r.Url, r.Service, r.Method); err != nil {
		return err
	}
	return nil
}

func (r *GrpcRequest) Runner(index int) (bool, int64, error) {
	now := time.Now()
	isEqual := false
	if para, err := r.getParam(index); err != nil {
		return false, 0, err
	} else {
		if res, err := request.GrpcRequest(r.Url, r.Service, r.Method, para); err != nil {
			return false, 0, err
		} else {
			isEqual = r.compare(index, res)
		}
	}

	duration := time.Since(now).Milliseconds()
	return isEqual, duration, nil
}
