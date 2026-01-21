package result

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/influxdata/tdigest"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/constants"
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

// InitResult 初始化结果记录
func InitResult() error {
	// 如果未声明日志保存方式，默认为写入本地文件；或是只是指定了存储到文件，但是没有具体写文件的路径，这里配置默认的文件路径
	if config.WTesterConfig.Db == nil {
		pwd, _ := os.Getwd()
		config.WTesterConfig.Db = &config.DbConfig{
			StorageType: constants.FileStorageType,
			Path:        filepath.Join(pwd, constants.ResultFilePath),
			BatchSize:   constants.ResultFileBufSize,
		}
	}

	// 日志保存配置初始化
	var dsn string
	var err error
	switch config.WTesterConfig.Db.StorageType {
	case constants.MysqlStorageType:
		if config.WTesterConfig.Db == nil || config.WTesterConfig.Db.GetPassword() == "" || config.WTesterConfig.Db.GetDatabase() == "" {
			return fmt.Errorf("数据库参数配置异常：password=%s; Database=%s", config.WTesterConfig.Db.GetPassword(), config.WTesterConfig.Db.GetDatabase())
		}
		if dsn, err = config.WTesterConfig.Db.GetDsn("pg"); err != nil {
			return err
		}
		if GormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{}); err != nil {
			return err
		}
	case constants.PostgreSQLStorageType:
		if config.WTesterConfig.Db == nil || config.WTesterConfig.Db.GetPassword() == "" || config.WTesterConfig.Db.GetDatabase() == "" {
			return fmt.Errorf("数据库参数配置异常：password=%s; Database=%s", config.WTesterConfig.Db.GetPassword(), config.WTesterConfig.Db.GetDatabase())
		}
		if dsn, err = config.WTesterConfig.Db.GetDsn("pg"); err != nil {
			return err
		}
		if GormDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{}); err != nil {
			return err
		}

	case constants.FileStorageType:
		config.Logger.Info(fmt.Sprintf("记录存储路径：recode=%s, Statistic=%s, error=%s",
			config.WTesterConfig.Db.GetResultPath(), config.WTesterConfig.Db.GetStatisticPath(), config.WTesterConfig.Db.GetErrorPath()),
		)
	default:
		panic(fmt.Errorf("指定的存储类型不支持: %s, 目前仅支持：%s", config.WTesterConfig.Db.StorageType, constants.AllStorageType))
	}

	if config.WTesterConfig.Db.StorageType != constants.FileStorageType {
		sqlDB, err := GormDB.DB()
		if err != nil {
			return err
		}
		// SetMaxIdleConns 用于设置连接池中空闲连接的最大数量。
		sqlDB.SetMaxIdleConns(10)
		// SetMaxOpenConns 设置打开数据库连接的最大数量。
		sqlDB.SetMaxOpenConns(100)
		// SetConnMaxLifetime 设置了连接可复用的最大时间。
		sqlDB.SetConnMaxLifetime(time.Hour)
		// 生成数据库
		if err = GormDB.AutoMigrate(&Result{}, &Statistic{}, &Error{}); err != nil {
			return err
		}
		config.Logger.Info(fmt.Sprintf("记录存储：recode=%s, Statistic=%s, error=%s",
			config.WTesterConfig.Db.GetResultPath(), config.WTesterConfig.Db.GetStatisticPath(), config.WTesterConfig.Db.GetErrorPath()),
		)
	}
	return nil
}
