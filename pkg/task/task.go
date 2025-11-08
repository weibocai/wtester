package task

import "fmt"

type Task interface {
	Init() error                           // 初始化，并校验参数
	Doc() string                           // 任务说明
	Runner(index int) (bool, int64, error) // 任务执行器
	AddClientPool() error                  // 注册请求客户端
	SetParam() error                       // 设置参数
	SetResponse() error                    // 设置返回信息
	GetName() string
	GetWeight() int
	GetParamsLength() int   // 任务参数的长度
	GetDescription() string // 任务的描述信息
}

type Factory func() Task

var taskFactories = map[string]Factory{}

// RegisterTask 注册任务
func RegisterTask(name string, factory Factory) bool {
	if _, ok := taskFactories[name]; ok {
		return false
	}
	taskFactories[name] = factory
	return true
}

// GetTask 获取任务
func GetTask(name string) (Task, error) {
	if factory, exists := taskFactories[name]; exists {
		return factory(), nil
	}
	return nil, fmt.Errorf("此任务类型为定义 %s", name)
}

func init() {
	RegisterTask("http", func() Task { return &HttpRequest{} })
	RegisterTask("grpc", func() Task { return &GrpcRequest{} })
}
