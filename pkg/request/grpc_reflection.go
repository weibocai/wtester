package request

import (
	"context"
	"fmt"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	refv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// DynamicGrpcClient grpc 请求客户端缓存
type DynamicGrpcClient struct {
	conn      *grpc.ClientConn               // 连接
	method    protoreflect.MessageDescriptor // 方法
	stub      *grpcdynamic.Stub
	refClient *grpcreflect.Client // 反射客户端
	methodC   *desc.MethodDescriptor
}

// CallMethod 访问grpc服务
func (c *DynamicGrpcClient) CallMethod(dynamicMsg *dynamicpb.Message) (string, error) {
	responseV1Msg, err := c.stub.InvokeRpc(context.Background(), c.methodC, dynamicMsg)
	if err != nil {
		return "", err
	}
	dm, err := dynamic.AsDynamicMessage(responseV1Msg)
	if err != nil {
		return "", err
	}
	// 处理结果
	js, err := dm.MarshalJSON()
	if err != nil {
		return "", err
	}
	return string(js), nil
}

var grpcReflectionClientPool = make(map[string]*DynamicGrpcClient)

// RegisterGrpcReflectionClient 注册grpc客户端
func RegisterGrpcReflectionClient(url, serviceName, methodName string) (*DynamicGrpcClient, error) {
	if c, ok := grpcReflectionClientPool[fmt.Sprintf("%s/%s/%s", url, serviceName, methodName)]; ok {
		return c, nil
	}
	// 构建连接
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	// 构建反射连接
	rConn := refv1.NewServerReflectionClient(conn)
	refClient := grpcreflect.NewClientV1(context.Background(), rConn)
	stub := grpcdynamic.NewStub(conn)

	// 查找服务描述符
	serviceDesc, err := refClient.ResolveService(serviceName)
	if err != nil {
		return nil, err
	}

	// 查找方法描述符
	methodC := serviceDesc.FindMethodByName(methodName)
	if methodC == nil {
		return nil, fmt.Errorf("method %s not found", methodName)
	}
	// 获取消息类型
	method := methodC.GetInputType().UnwrapMessage()

	// 创建请求客户端
	c := &DynamicGrpcClient{conn: conn, method: method, methodC: methodC, stub: &stub, refClient: refClient}
	grpcReflectionClientPool[fmt.Sprintf("%s/%s/%s", url, serviceName, methodName)] = c
	return c, nil
}

// GetGrpcReflectionClient 获取grpc客户端
func GetGrpcReflectionClient(url string, serviceName, methodName string) (*DynamicGrpcClient, error) {
	if c, ok := grpcReflectionClientPool[fmt.Sprintf("%s/%s/%s", url, serviceName, methodName)]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("找不到对应的客户端：%s/%s/%s", url, serviceName, methodName)
}

// CloseGrpcReflectionClient 结束时关闭连接
func CloseGrpcReflectionClient() {
	for _, c := range grpcReflectionClientPool {
		_ = c.conn.Close()
	}
}

// GrpcReflectionRequest grpc 请求
func GrpcReflectionRequest(url, serviceName, methodName string, requestData map[string]any) (string, error) {
	c, err := GetGrpcReflectionClient(url, serviceName, methodName)
	if c == nil {
		return "", err
	}
	// 创建动态消息
	inputType := c.method
	dynamicMsg := dynamicpb.NewMessage(inputType)

	// 构建请求参数
	for field, value := range requestData {
		dynamicMsg.Set(inputType.Fields().ByJSONName(field), protoreflect.ValueOf(value))
	}
	return c.CallMethod(dynamicMsg)
}
