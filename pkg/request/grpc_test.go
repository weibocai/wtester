package request

import (
	"sync"
	"testing"
)

func TestGrpcClientPool(t *testing.T) {
	url, serviceName, methodName := "127.0.0.1:50051", "Greeter", "SayHello"
	requestData := map[string]any{"name": "John Doe"}
	RegisterGrpcClient(url, serviceName, methodName)
	var group sync.WaitGroup
	for i := 0; i <= 10; i++ {
		group.Go(func() {
			for j := 0; j <= 10; j++ {
				GrpcRequest(url, serviceName, methodName, requestData)
			}
		})
	}
	group.Wait()
	CloseGrpcClient()
}
