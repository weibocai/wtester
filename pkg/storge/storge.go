package storge

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/constants"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var GormDB *gorm.DB

// InitStorge 初始化结果记录
func InitStorge() error {
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
