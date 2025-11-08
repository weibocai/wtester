package request

import (
	"fmt"
	"sync"
	"testing"
)

func TestGrpcClientPool(t *testing.T) {
	url, serviceName, methodName := "localhost:50051", "Greeter", "SayHello"
	requestData := map[string]any{"name": "John Doe"}
	if _, err := RegisterGrpcClient(url, serviceName, methodName); err != nil {
		t.Fatalf("register grpc client failed: %v", err)
	}
	var group sync.WaitGroup
	for i := 0; i <= 10; i++ {
		group.Go(func() {
			for j := 0; j <= 10; j++ {
				if res, err := GrpcRequest(url, serviceName, methodName, requestData); err != nil {
					t.Errorf("GrpcRequest failed: %v", err)
				} else {
					fmt.Println(res)
				}
			}
		})
	}
	group.Wait()
	CloseGrpcClient()
}
