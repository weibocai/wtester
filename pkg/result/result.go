package result

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/constants"
	"github.com/wtester/pkg/library"
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

func (r *Result) ToError() *Error {
	return &Error{
		Stage:         r.Stage,
		Task:          r.Task,
		Exception:     r.Exception,
		ExecutionTime: r.ExecutionTime,
		Datetime:      r.Datetime,
	}
}

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

// InitResult 初始化结果记录
func InitResult() {
	switch config.WTesterConfig.Result.StorageType {
	case constants.MYSQLStorageType:
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			config.WTesterConfig.Result.DbConfig.Username, config.WTesterConfig.Result.DbConfig.Password, config.WTesterConfig.Result.DbConfig.Host, config.WTesterConfig.Result.DbConfig.Port, config.WTesterConfig.Result.DbConfig.Database)
		var err error
		if GormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{}); err != nil {
			panic(err)
		}
		sqlDB, err := GormDB.DB()
		if err != nil {
			panic(err)
		}
		// SetMaxIdleConns 用于设置连接池中空闲连接的最大数量。
		sqlDB.SetMaxIdleConns(10)
		// SetMaxOpenConns 设置打开数据库连接的最大数量。
		sqlDB.SetMaxOpenConns(100)
		// SetConnMaxLifetime 设置了连接可复用的最大时间。
		sqlDB.SetConnMaxLifetime(time.Hour)
		// 生成数据库
		if err = GormDB.AutoMigrate(&Result{}, &Statistic{}, &Error{}); err != nil {
			panic(err)
		}
	default:
		// 写入文件
		if config.WTesterConfig.Result == nil {
			pwd, _ := os.Getwd()
			config.WTesterConfig.Result = &config.ResultConfig{
				StorageType: constants.FileStorageType,
				FileConfig: &config.FileConfig{
					Path:    filepath.Join(pwd, constants.ResultFilePath),
					BufSize: constants.ResultFileBufSize,
				},
			}
		} else if config.WTesterConfig.Result.FileConfig == nil {
			pwd, _ := os.Getwd()
			config.WTesterConfig.Result.StorageType = constants.FileStorageType
			config.WTesterConfig.Result.FileConfig = &config.FileConfig{
				Path:    filepath.Join(pwd, constants.ResultFilePath),
				BufSize: constants.ResultFileBufSize,
			}
		}
		path := config.WTesterConfig.Result.FileConfig.Path
		if err := library.CreateDirectoryIfNotExists(path); err != nil {
			panic(err)
		}
		// 结果
		if f, err := os.Create(filepath.Join(config.WTesterConfig.Result.FileConfig.Path, "result.txt")); err != nil {
			panic(err)
		} else {
			_ = f.Close()
		}
		// 统计结果
		if f, err := os.Create(filepath.Join(config.WTesterConfig.Result.FileConfig.Path, "statistic.txt")); err != nil {
			panic(err)
		} else {
			_ = f.Close()
		}
	}
}
