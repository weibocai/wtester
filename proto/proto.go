package proto

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// GrpcServerCallFunc grpc 客户端调用函数定义
type GrpcServerCallFunc func(conn *grpc.ClientConn, requestData map[string]any) (string, error)

var grpcServiceFactories map[string]GrpcServerCallFunc

// RegisterGrpcServiceFactories 客户端注册
func RegisterGrpcServiceFactories(service string, f GrpcServerCallFunc) {
	if _, ok := grpcServiceFactories[service]; ok {
		return
	}
	grpcServiceFactories[service] = f
}

// GetGrpcServiceFactories 获取请求客户端
func GetGrpcServiceFactories(service string) GrpcServerCallFunc {
	if f, ok := grpcServiceFactories[service]; ok {
		return f
	}
	return nil
}
func init() {
	grpcServiceFactories = make(map[string]GrpcServerCallFunc)
	RegisterGrpcServiceFactories("Greeter.SayHello", GreeterSayHello)
}

// SetPara 参数构建
func (x *HelloRequest) SetPara(para map[string]any) {
	x.A = para["a"].(string)
	x.B = para["b"].(int32)
	x.C = para["c"].(int64)
	x.D = para["d"].(uint32)
	x.E = para["e"].(uint64)
	x.F = para["f"].(int32)
	x.G = para["g"].(int64)
	x.H = para["h"].(uint32)
	x.I = para["i"].(uint64)
	x.J = para["j"].(int32)
	x.K = para["k"].(int64)
	x.L = para["l"].(float32)
	x.M = para["m"].(float64)
	x.N = para["n"].(bool)
	x.O = para["o"].([]byte)
}
func (x *HelloReply) SetPara(para map[string]any) {
	x.Message = para["message"].(string)
}

func GreeterSayHello(conn *grpc.ClientConn, requestData map[string]any) (string, error) {
	c := NewGreeterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	para := HelloRequest{}
	para.SetPara(requestData)
	r, err := c.SayHello(ctx, &para)
	if err != nil {
		return "", err
	}
	return r.String(), nil
}
