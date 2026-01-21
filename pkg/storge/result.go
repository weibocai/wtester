package storge

import (
	"math"
	"time"

	"github.com/influxdata/tdigest"

	"github.com/wtester/pkg/config"
)

// Error 请求结果
type Error struct {
	ID            uint      `gorm:"primaryKey;<-:false"`
	Stage         string    `json:"stage"`          // 阶段名称
	Task          string    `json:"task"`           // 任务名称
	Exception     string    `json:"exception"`      // 异常信息
	ExecutionTime int64     `json:"execution_time"` // 执行时间 单位为毫秒
	Datetime      time.Time `json:"datetime"`       // 统计时间
}

func (Error) TableName() string {
	return config.WTesterConfig.Db.GetErrorPath()
}

// Result 请求结果
type Result struct {
	ID            uint      `gorm:"primaryKey;<-:false"`
	Stage         string    `json:"stage"`                    // 阶段名称
	Task          string    `json:"task"`                     // 任务名称
	CC            int       `json:"cc"`                       // 当前并发的个数
	IsEqual       bool      `json:"is_equal"`                 // 是否符合预期
	IsSuccess     bool      `json:"is_success"  gorm:"-:all"` // 是否执行成功
	ExecutionTime int64     `json:"execution_time"`           // 执行时间 单位为毫秒
	Datetime      time.Time `json:"datetime"`                 // 统计时间
	Exception     string    `json:"exception"   gorm:"-:all"` // 异常信息
}

func (Result) TableName() string {
	return config.WTesterConfig.Db.GetResultPath()
}

func (r Result) ToError() *Error {
	return &Error{
		Stage:         r.Stage,
		Task:          r.Task,
		Exception:     r.Exception,
		ExecutionTime: r.ExecutionTime,
		Datetime:      r.Datetime,
	}
}

// Statistic 统计结果
type Statistic struct {
	ID           uint      `gorm:"primaryKey;<-:false"`
	Stage        string    `json:"stage"`              // 阶段名称
	Task         string    `json:"task"`               // 任务名称
	SuccessCount int       `json:"success_count"`      // 执行成功的个数
	FailureCount int       `json:"failure_count"`      // 执行失败的个数
	CC           int       `json:"cc"`                 // 平均并发的个数
	Count        int       `json:"count" gorm:"-:all"` // 本阶段收集的结果的个数
	Avg          int       `json:"avg"`                // 平均执行时间，单位为毫秒
	Sum          int64     `json:"-"`                  // 平均执行时间，单位为毫秒
	P99          float64   `json:"p99"`                // p99
	P95          float64   `json:"p95"`                // p95
	P50          float64   `json:"p50"`                // p50
	Datetime     time.Time `json:"datetime"`           // 统计时间
}

func (*Statistic) TableName() string {
	return config.WTesterConfig.Db.GetStatisticPath()
}

// Reset 重置统计结果
func (s *Statistic) Reset() {
	s.SuccessCount = 0
	s.FailureCount = 0
	s.Avg = 0
	s.Sum = 0
	s.P99 = 0
	s.P95 = 0
	s.P50 = 0
	s.Datetime = time.Now()
}

// UpdateStatistic 更新汇总结果
func (s *Statistic) UpdateStatistic(ttd *tdigest.TDigest) {
	P99 := ttd.Quantile(0.99)
	if P99 < 0 || math.IsNaN(P99) {
		P99 = 0
	}
	P95 := ttd.Quantile(0.95)
	if P95 < 0 || math.IsNaN(P95) {
		P95 = 0
	}
	P50 := ttd.Quantile(0.50)
	if P50 < 0 || math.IsNaN(P50) {
		P50 = 0
	}
	s.P99 = P99
	s.P95 = P95
	s.P50 = P50
	s.CC = s.CC / s.Count
	ttd.Reset()
	s.Avg = int(s.Sum) / (s.SuccessCount + s.FailureCount)
	s.Datetime = time.Now()
}
