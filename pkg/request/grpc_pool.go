package request

import (
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GrpcClientConn 定义grpc连接池对应的连接器，方便后续添加操作
type GrpcClientConn struct {
	UsedNum    int              // 调用次数
	CreateTime time.Time        // 创建时间
	Conn       *grpc.ClientConn // 连接器
}

// GrpcClientConnPool grpc 连接池
type GrpcClientConnPool struct {
	Pool         chan *GrpcClientConn // 连接池
	Timeout      time.Duration        // 超时时间
	MaxIdleConns int                  // 最大空闲连接数
	MinIdleConns int                  // 最小空闲连接数
	Addr         string               // 连接地址

	lck   *sync.Mutex // 锁
	conns int         // 当前连接数
}

// Set 构建新的连接
func (p *GrpcClientConnPool) Set() error {
	p.lck.Lock()
	defer p.lck.Unlock()
	if p.conns >= p.MaxIdleConns {
		return fmt.Errorf("max idle connections reached")
	}
	conn, err := grpc.NewClient(p.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	c := &GrpcClientConn{UsedNum: 1, CreateTime: time.Now(), Conn: conn}
	p.Pool <- c
	p.conns += 1
	return nil
}

// Revert 连接使用完成之后，归还到执行队列
func (p *GrpcClientConnPool) Revert(conn *GrpcClientConn) {
	p.Pool <- conn
}

// Get 获取新的连接
func (p *GrpcClientConnPool) Get() (*GrpcClientConn, error) {
	for i := 0; i < 3; i++ {
		if len(p.Pool) > 0 || p.conns >= p.MaxIdleConns {
			break
		}
		// 如果队列为空，且仍有空闲，则创建新的连接
		if err := p.Set(); err == nil {
			break
		}
	}

	select {
	case c := <-p.Pool:
		c.UsedNum += 1
		return c, nil
	case <-time.After(p.Timeout):
		return nil, fmt.Errorf("max idle connections reached")
	}
}

// Close 关闭连接
func (p *GrpcClientConnPool) Close() {
	close(p.Pool)
	for conn := range p.Pool {
		_ = conn.Conn.Close()
	}
}

func NewClientConnPool(addr string, maxIdleConn int, timeout time.Duration) *GrpcClientConnPool {
	return &GrpcClientConnPool{
		Addr: addr, Pool: make(chan *GrpcClientConn, maxIdleConn), Timeout: timeout, MaxIdleConns: maxIdleConn, conns: 0, lck: &sync.Mutex{},
	}
}

var grpcClientPool map[string]*GrpcClientConnPool

// RegisterGrpcClientPool 注册 grpc 客户端，用于复用，在编辑task的时候，就开始注册
func RegisterGrpcClientPool(addr string, maxIdleConn int, timeout time.Duration) (*GrpcClientConnPool, error) {
	if c, ok := grpcClientPool[addr]; ok {
		return c, nil
	}
	c := NewClientConnPool(addr, maxIdleConn, timeout)
	grpcClientPool[addr] = c
	return c, nil
}

func CloseGrpcClient() {
	for _, c := range grpcClientPool {
		c.Close()
	}
}

func init() {
	grpcClientPool = make(map[string]*GrpcClientConnPool)
}
