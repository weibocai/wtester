package result

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wtester/pkg/logger"
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

func (Error) TableName() string {
	return config.WTesterConfig.Result.DbConfig.GetErrorPath()
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
	return config.WTesterConfig.Result.DbConfig.GetResultPath()
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
	return config.WTesterConfig.Result.DbConfig.GetStatisticPath()
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
	// 如果未声明日志保存方式，默认为写入本地文件；或是只是指定了存储到文件，但是没有具体写文件的路径，这里配置默认的文件路径
	if config.WTesterConfig.Result == nil || (config.WTesterConfig.Result.StorageType == constants.FileStorageType && config.WTesterConfig.Result.FileConfig == nil) {
		pwd, _ := os.Getwd()
		config.WTesterConfig.Result = &config.ResultConfig{
			StorageType: constants.FileStorageType,
			FileConfig: &config.FileConfig{
				Path:    filepath.Join(pwd, constants.ResultFilePath),
				BufSize: constants.ResultFileBufSize,
			},
		}
	}

	// 日志保存配置初始化
	switch config.WTesterConfig.Result.StorageType {
	case constants.MysqlStorageType:
		// mysql 保存到mysql数据库
		if config.WTesterConfig.Result.DbConfig == nil || config.WTesterConfig.Result.DbConfig.GetPassword() == "" || config.WTesterConfig.Result.DbConfig.GetDatabase() == "" {
			panic(fmt.Errorf("数据库参数配置异常：password=%s; Database=%s", config.WTesterConfig.Result.DbConfig.GetPassword(), config.WTesterConfig.Result.DbConfig.GetDatabase()))
		}
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
		logger.Logger.Info(fmt.Sprintf("记录存储：recode=%s, Statistic=%s, error=%s",
			config.WTesterConfig.Result.DbConfig.GetResultPath(), config.WTesterConfig.Result.DbConfig.GetStatisticPath(), config.WTesterConfig.Result.DbConfig.GetErrorPath()),
		)
	case constants.FileStorageType:
		path := config.WTesterConfig.Result.FileConfig.Path
		if err := library.CreateDirectoryIfNotExists(path); err != nil {
			panic(err)
		}
		// 结果
		if f, err := os.Create(config.WTesterConfig.Result.FileConfig.GetResultPath()); err == nil {
			_ = f.Close()
		} else {
			panic(err)
		}
		// 异常结果
		if f, err := os.Create(config.WTesterConfig.Result.FileConfig.GetErrorPath()); err == nil {
			_ = f.Close()
		} else {
			panic(err)
		}
		// 统计结果
		if f, err := os.Create(config.WTesterConfig.Result.FileConfig.GetStatisticPath()); err == nil {
			_ = f.Close()
		} else {
			panic(err)
		}
		logger.Logger.Info(fmt.Sprintf("记录存储路径：recode=%s, Statistic=%s, error=%s",
			config.WTesterConfig.Result.FileConfig.GetResultPath(), config.WTesterConfig.Result.FileConfig.GetStatisticPath(), config.WTesterConfig.Result.FileConfig.GetErrorPath()),
		)
	default:
		panic(fmt.Errorf("指定的存储类型不支持: %s, 目前仅支持：%s", config.WTesterConfig.Result.StorageType, constants.AllStorageType))
	}
}
