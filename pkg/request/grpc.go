 package request

import (
	"fmt"

	"github.com/wtester/proto"
)

// GrpcRequest 获取grpc请求客户端
func GrpcRequest(url string, service string, requestData map[string]any) (string, error) {
	c, ok := grpcClientPool[url]
	if !ok {
		return "", fmt.Errorf("找不到对应的客户端：%s", url)
	}
	f := proto.GetGrpcServiceFactories(service)
	if f == nil {
		return "", fmt.Errorf("未定义 grpc 相关请求客户端函数：%s", service)
	}
	conn, err := c.Get()
	if err != nil {
		return "", err
	}
	defer c.Revert(conn)
	return f(conn.Conn, requestData)
}