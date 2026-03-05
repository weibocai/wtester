package swarm

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/wtester/pkg/config"
	"go.uber.org/zap"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Actor struct {
	Name               string           // 节点名称
	Status             ActorStatus      // 节点状态
	StageIndex         uint32           // 执行到第几个阶段
	StageIndexIsFinish bool             // 当前阶段是否执行结束
	Host               string           // 客户端连接地址
	Client             *grpc.ClientConn // 客户端连接器
	WClient            WTesterClient    // 客户端连接器
	CoroutinesNumber   chan int         // 执行端待启动线程队列
	EndSignal          chan bool        // 中止信号

	lock sync.Mutex // 锁，确保协程安全
}

// GetClient 获取grpc连接客户端
func (a *Actor) GetClient() WTesterClient {
	return a.WClient
}

// ResetClient 重新构建客户端
func (a *Actor) ResetClient() (WTesterClient, error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.Close()
	conn, err := grpc.NewClient(a.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	a.WClient = NewWTesterClient(conn)
	return a.WClient, nil
}

// Close 关闭信息接收
func (a *Actor) Close() {
	if a.Client != nil {
		_ = a.Client.Close()
	}
	if a.EndSignal != nil {
		close(a.EndSignal)
	}
}

// IsActiveNode 检查节点状态是否是活跃
func (a *Actor) IsActiveNode() bool {
	return a.Status == ActorStatus_Online
}

// SetActiveNode 设置节点状态
func (a *Actor) SetActiveNode(status ActorStatus) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.Status = status
}

// GetStageIndex 执行到第几个阶段
func (a *Actor) GetStageIndex() uint32 {
	return a.StageIndex
}

// UpdateStage 更新执行阶段
func (a *Actor) UpdateStage(index uint32, isFinish bool) {
	a.StageIndex = index
	a.StageIndexIsFinish = isFinish
}

func (a *Actor) NotifyRegister(ctx context.Context, kind ActorStatus, host string, errChan chan error) bool {
	m := RegisterRequest{Kind: kind, Pod: a.Name, Host: &host}
	ctxA, cancelA := context.WithTimeout(ctx, 10*time.Second)
	defer cancelA()
	if res, err := a.WClient.ActorRegister(ctxA, &m); err != nil || res.GetStatus() == Status_Fail {
		cancelA()
		errChan <- fmt.Errorf("执行端 %s 注册/下线：%v", a.Name, kind)
		return false
	}
	a.Status = kind
	return true
}

func (a *Actor) NotifyCoroutinesNumber(ctx context.Context, num uint32, errChan chan error) bool {
	ctxA, cancel := context.WithTimeout(ctx, 1*time.Second)
	m := StageRequest{Kind: StatementKind_FbCoroutinesNumber, CoroutinesNumber: &num, Pod: a.Name}
	defer cancel()
	if res, err := a.WClient.DirectorStage(ctxA, &m); err != nil || res.GetStatus() == Status_Fail {
		errChan <- fmt.Errorf("%s 上传执行端测试进程失败", a.Name)
		return false
	}
	return true
}

func (a *Actor) NotifyUpdateStage(ctx context.Context, errChan chan error) bool {
	ctxA, cancel := context.WithTimeout(ctx, 1*time.Second)
	sk := StageKind_FinishStage
	m := StageRequest{Kind: StatementKind_UpdateState, Pod: a.Name, StageKind: &sk, StageIndex: a.StageIndex}
	if a.CoroutinesNumber != nil {
		close(a.CoroutinesNumber)
	}
	defer cancel()
	if res, err := a.WClient.DirectorStage(ctxA, &m); err != nil || res.GetStatus() == Status_Fail {
		errChan <- fmt.Errorf("%s 反馈阶段执行结束失败", a.Name)
		return false
	}
	return true
}

// WaitingUpdateStage 等待阶段变更
func (a *Actor) WaitingUpdateStage(ctx context.Context, oldStage uint32, errChan chan error) bool {
	ctxT, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for {
		select {
		case <-ctxT.Done():
			if a.StageIndex == oldStage {
				errChan <- fmt.Errorf("%s 阶段获取/更新失败", a.Name)
				return false
			}
			return true
		default:
			if a.StageIndex != oldStage {
				return true
			}
		}
	}
}

// WaitingEndSignal 等待接收服务端终止信号
func (a *Actor) WaitingEndSignal(ctx context.Context, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.EndSignal:
			cancel()
			return
		}
	}
}

// GetActiveActor 检查活跃节点的个数
func GetActiveActor(actors map[string]*Actor) int {
	a := 0
	for _, v := range actors {
		if v.IsActiveNode() {
			a += 1
		}
	}
	return a
}

// IsStateFinish 检查执行端本阶段是否执行结束
func IsStateFinish(actors map[string]*Actor, index int) string {
	i := uint32(index)
	a := strings.Builder{}
	for _, v := range actors {
		// 死掉的节点，认为已经结束
		if !v.IsActiveNode() {
			continue
		}
		if v.StageIndex != i {
			a.WriteString(v.Name)
			continue
		}
		if !v.StageIndexIsFinish {
			a.WriteString(v.Name)
			continue
		}
	}
	return a.String()
}

// BroadcastTerminate 广播，向所有的客户端发送命令
func BroadcastTerminate(actors map[string]*Actor) bool {
	for _, a := range actors {
		ctxA, cancelA := context.WithTimeout(context.Background(), time.Second)
		_, _ = a.WClient.ActorTerminate(ctxA, &TerminateRequest{})
		cancelA()
	}
	return true
}

func NotifyUpdateStage(ctx context.Context, actors map[string]*Actor, stageIndex uint32, errChan chan error) bool {
	k := StageKind_StartStage
	msg := &StageRequest{Kind: StatementKind_UpdateState, StageIndex: stageIndex, StageKind: &k}
	for _, a := range actors {
		a.UpdateStage(stageIndex, false)
		ctxA, cancelA := context.WithTimeout(ctx, time.Second)
		res, err := a.WClient.ActorStage(ctxA, msg)
		if err != nil || res.GetStatus() == Status_Fail {
			cancelA()
			errChan <- fmt.Errorf("%s 设置执行端执行阶段失败: %v", a.Name, err)
			return false
		}
		cancelA()
	}
	return true
}

func InitActor(ctx context.Context, host, name string, initStageIndex uint32) *Actor {
	ctxA, cancelA := context.WithTimeout(ctx, 6*time.Second)
	defer cancelA()
	// 尝试构建客户端连接，如果超时连接失败，则终止测试
	config.Logger.Info("等待连接到服务端", zap.String("name", name), zap.String("host", host))
	for {
		select {
		case <-ctxA.Done():
			config.Logger.Error("服务端连接超时", zap.String("name", name), zap.String("host", host))
			return nil
		default:
			if conn, err := grpc.NewClient(host, grpc.WithTransportCredentials(insecure.NewCredentials())); err == nil {
				var actor = &Actor{Name: name, Status: ActorStatus_WaitingOnline, Client: conn, WClient: NewWTesterClient(conn), StageIndex: initStageIndex, EndSignal: make(chan bool, 10)}
				config.Logger.Info("连接到服务端成功", zap.String("name", name), zap.String("host", host))
				return actor
			}
			time.Sleep(1 * time.Second)
		}
	}
}

type ActorService struct {
	UnimplementedWTesterServer
	ActorDeal *Actor
}

func (a *ActorService) ActorStage(_ context.Context, r *StageRequest) (*CommonResponse, error) {
	switch r.Kind {
	case StatementKind_UpdateState:
		a.ActorDeal.UpdateStage(r.GetStageIndex(), false)
		a.ActorDeal.CoroutinesNumber = make(chan int, 10)
	case StatementKind_StartCoroutine:
		if a.ActorDeal.GetStageIndex() != r.GetStageIndex() {
			return &CommonResponse{Status: Status_Fail}, nil
		}
		a.ActorDeal.CoroutinesNumber <- int(r.GetCoroutinesNumber())
	}
	return &CommonResponse{Status: Status_Success}, nil
}

func (a *ActorService) ActorTerminate(context.Context, *TerminateRequest) (*CommonResponse, error) {
	config.Logger.Info("收到结束信号，中止测试", zap.String("name", a.ActorDeal.Name))
	a.ActorDeal.EndSignal <- true
	return &CommonResponse{Status: Status_Success}, nil
}

// actorService 启动执行端
func actorService(ctx context.Context, actor *Actor, host string, port int, errChan chan error) {
	service := &ActorService{ActorDeal: actor}
	grpcServer := grpc.NewServer()
	RegisterWTesterServer(grpcServer, service)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
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
	// 优雅关闭：等待客户端处理完当前流之后关闭服务
	defer grpcServer.GracefulStop()
	defer func(lis net.Listener) {
		_ = lis.Close()
	}(lis)
	// 节点注册
	if !actor.NotifyRegister(ctx, ActorStatus_Online, fmt.Sprintf("%s:%d", host, port), errChan) {
		write2recodes("actor", fmt.Sprintf("%s：节点注册失败", actor.Name), nil)
		return
	}
	// 守护进程，直到服务结束
	<-ctx.Done()
}
