package swarm

import (
	"context"
	"net"
	"time"

	"github.com/wtester/pkg/config"
	"go.uber.org/zap"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DirectorDealMessage struct {
	Pod              string        `json:"pod,omitempty"`
	Kind             StatementKind `json:"kind,omitempty"`
	CoroutinesNumber uint32        `json:"coroutinesNumber,omitempty"` // 启动线程的个数
	Loop             uint32        `json:"loop,omitempty"`             // 执行的个数
	Message          string        `json:"message,omitempty"`          // 客户端消息
	StageIndex       uint32        ` json:"StageIndex,omitempty"`      // 当前执行阶段
}

func (d DirectorDealMessage) GetCoroutinesNumber() uint32 {
	return d.CoroutinesNumber
}

func (d DirectorDealMessage) GetStageIndex() uint32 {
	return d.StageIndex
}

type DirectorService struct {
	UnimplementedWTesterServer
	DirectorDeal chan *DirectorDealMessage
	ActorDeal    map[string]*Actor
}

func actorRegister(ctx context.Context, r *RegisterRequest) *Actor {
	ctxA, cancelA := context.WithTimeout(ctx, 6*time.Second)
	defer cancelA()
	// 尝试构建客户端连接，如果超时连接失败，则终止测试
	config.Logger.Info("等待连接到执行端", zap.String("name", r.GetPod()), zap.String("host", r.GetHost()))
	for {
		select {
		case <-ctxA.Done():
			config.Logger.Error("执行端连接超时", zap.String("name", r.GetPod()), zap.String("host", r.GetHost()))
			return nil
		default:
			if conn, err := grpc.NewClient(r.GetHost(), grpc.WithTransportCredentials(insecure.NewCredentials())); err == nil {
				actor := &Actor{Name: r.GetPod(), Status: ActorStatus_Online, StageIndex: 0, StageIndexIsFinish: false, Client: conn, WClient: NewWTesterClient(conn)}
				config.Logger.Info("连接到执行端成功", zap.String("name", r.GetPod()), zap.String("host", r.GetHost()))
				return actor
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func (s *DirectorService) ActorRegister(ctx context.Context, r *RegisterRequest) (*CommonResponse, error) {
	if r.GetKind() == ActorStatus_Online {
		a := actorRegister(ctx, r)
		if a == nil {
			return &CommonResponse{Status: Status_Fail}, nil
		}
		// 注册节点
		s.ActorDeal[r.GetPod()] = a
		write2recodes(r.Pod, "node_register", nil)
	} else {
		a, ok := s.ActorDeal[r.GetPod()]
		if !ok {
			return &CommonResponse{Status: Status_Fail}, nil
		}
		a.SetActiveNode(r.GetKind())
	}
	return &CommonResponse{Status: Status_Success}, nil
}

func (s *DirectorService) DirectorStage(_ context.Context, r *StageRequest) (*CommonResponse, error) {
	// 消息接收
	a, ok := s.ActorDeal[r.GetPod()]
	if !ok {
		return &CommonResponse{Status: Status_Fail}, nil
	}
	res := &CommonResponse{Status: Status_Success}
	switch r.Kind {
	case StatementKind_UpdateState:
		// 进入到下一阶段
		if r.GetStageKind() == StageKind_FinishStage {
			a.UpdateStage(r.StageIndex, true)
		}
	case StatementKind_FbCoroutinesNumber:
		// 执行段反馈启动的协程数
		s.DirectorDeal <- &DirectorDealMessage{Pod: r.GetPod(), Kind: r.GetKind(), CoroutinesNumber: r.GetCoroutinesNumber(), Loop: r.GetLoop()}
	}
	return res, nil
}

// directorService 启动服务端
func directorService(ctx context.Context, errChan chan error, directorDeal chan *DirectorDealMessage, actorDeal map[string]*Actor) {
	service := &DirectorService{DirectorDeal: directorDeal, ActorDeal: actorDeal}
	grpcServer := grpc.NewServer()
	RegisterWTesterServer(grpcServer, service)
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		errChan <- err
		return
	}
	// 启动服务端
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			errChan <- err
			return
		}
	}()

	// 守护进程，直到服务结束
	<-ctx.Done()
	// 优雅关闭：等待客户端处理完当前流之后关闭服务
	grpcServer.GracefulStop()
	_ = lis.Close()
	return
}
