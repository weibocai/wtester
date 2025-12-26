package task

import (
	"time"

	"github.com/wtester/pkg/request"
)

type GrpcReflectionRequest struct {
	HttpRequest `yaml:",inline"`
	Service     string `yaml:"service"`
}

func (r *GrpcReflectionRequest) AddClientPool() error {
	if _, err := request.RegisterGrpcReflectionClient(r.Url, r.Service, r.Method); err != nil {
		return err
	}
	return nil
}

func (r *GrpcReflectionRequest) Runner(index int) (bool, int64, error) {
	now := time.Now()
	isEqual := false
	if para, err := r.getParam(index); err != nil {
		return false, 0, err
	} else {
		if res, err := request.GrpcReflectionRequest(r.Url, r.Service, r.Method, para); err != nil {
			return false, 0, err
		} else {
			isEqual = r.compare(index, res)
		}
	}

	duration := time.Since(now).Milliseconds()
	return isEqual, duration, nil
}
