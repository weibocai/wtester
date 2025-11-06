package request

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// DynamicGrpcClient grpc 请求客户端缓存
type DynamicGrpcClient struct {
	conn       *grpc.ClientConn              // 连接
	method     protoreflect.MethodDescriptor // 方法
	fullMethod string                        // 请求方法
}

// CallMethod 访问grpc服务
func (c *DynamicGrpcClient) CallMethod(dynamicMsg *dynamicpb.Message) (string, error) {
	result := dynamicpb.NewMessage(c.method.Output())
	if err := c.conn.Invoke(context.Background(), c.fullMethod, dynamicMsg, result); err != nil {
		return "", err
	}

	// 处理结果
	if resultJSON, err := protojson.Marshal(result); err != nil {
		return "", err
	} else {
		return string(resultJSON), nil
	}
}

var grpcClientPool = make(map[string]*DynamicGrpcClient)

// RegisterGrpcClient 注册grpc客户端
func RegisterGrpcClient(url, serviceName, methodName string) (*DynamicGrpcClient, error) {
	if c, ok := grpcClientPool[url]; ok {
		return c, nil
	}
	// 查找服务描述符
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, err
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("not a service descriptor")
	}

	// 查找方法描述符
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, fmt.Errorf("method not found")
	}

	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		return nil, err
	}
	// 创建请求客户端
	c := &DynamicGrpcClient{conn: conn, method: method, fullMethod: fmt.Sprintf("/%s/%s", serviceName, methodName)}
	grpcClientPool[fmt.Sprintf("%s/%s/%s", url, serviceName, methodName)] = c
	return c, nil
}

// GetGrpcClient 获取grpc客户端
func GetGrpcClient(url string, serviceName, methodName string) (*DynamicGrpcClient, error) {
	if c, ok := grpcClientPool[fmt.Sprintf("%s/%s/%s", url, serviceName, methodName)]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("找不到对应的客户端：%s/%s/%s", url, serviceName, methodName)
}

// CloseGrpcClient 结束时关闭连接
func CloseGrpcClient() {
	for _, c := range grpcClientPool {
		c.conn.Close()
	}
}

// GrpcRequest grpc 请求
func GrpcRequest(url string, serviceName, methodName string, requestData map[string]any) (string, error) {
	c, err := GetGrpcClient(url, serviceName, methodName)
	if c == nil {
		return "", err
	}
	// 创建动态消息
	inputType := c.method.Input()
	dynamicMsg := dynamicpb.NewMessage(inputType)

	// 构建请求参数
	for field, value := range requestData {
		dynamicMsg.Set(inputType.Fields().ByJSONName(field), protoreflect.ValueOf(value))
	}
	return c.CallMethod(dynamicMsg)
}
