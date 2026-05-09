package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/wtester/pkg/constants"
)

// LogConfig 日志配置
type LogConfig struct {
	Path string `json:"path" yaml:"path" mapstructure:"path"` // 日志存储的路径
}

// DbConfig 数据库配置
type DbConfig struct {
	StorageType constants.StorageType `json:"storageType" yaml:"storageType" mapstructure:"storageType"` // 存储介质

	// 文件存储路径
	Path string `json:"path" yaml:"path" mapstructure:"path"`

	// 数据库配置
	Host      string `json:"host" yaml:"host" mapstructure:"host"`
	Port      int    `json:"port" yaml:"port" mapstructure:"port"`
	Username  string `json:"username" yaml:"username" mapstructure:"username"`
	Password  string `json:"password" yaml:"password" mapstructure:"password"`
	Database  string `json:"database" yaml:"database" mapstructure:"database"`
	BatchSize int    `json:"batch_size" yaml:"batch_size" mapstructure:"batch_size"`
}

// GetHost 获取连接地址
func (f *DbConfig) GetHost() string {
	if f.Host == "" {
		f.Host = "127.0.0.1"
	}
	return f.Host
}

// GetPort 获取连接端口
func (f *DbConfig) GetPort() int {
	if f.Port == 0 {
		f.Port = 3306
	}
	return f.Port
}
func (f *DbConfig) GetUsername() string {
	if f.Username == "" {
		f.Username = "root"
	}
	return f.Username
}
func (f *DbConfig) GetPassword() string {
	return f.Password
}

// GetDatabase 存储数据库
func (f *DbConfig) GetDatabase() string {
	return f.Database
}

// GetDsn 连接串
func (f *DbConfig) GetDsn(kind string) (string, error) {
	switch kind {
	case "pg":
		return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d", f.GetHost(), f.GetUsername(), f.GetPassword(), f.GetDatabase(), f.GetPort()), nil
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", f.GetUsername(), f.GetPassword(), f.GetHost(), f.GetPort(), f.GetDatabase()), nil
	default:
		return "", fmt.Errorf("")

	}
}

// GetResultPath 全量日志存储路径
func (f *DbConfig) GetResultPath() string {
	if f.StorageType == constants.FileStorageType {
		return filepath.Join(f.Path, fmt.Sprintf("result_%s.txt", WTesterConfig.GetVersion()))
	}
	return fmt.Sprintf("result_%s", WTesterConfig.GetVersion())
}

// GetErrorPath 失败文件路径
func (f *DbConfig) GetErrorPath() string {
	if f.StorageType == constants.FileStorageType {
		return filepath.Join(f.Path, fmt.Sprintf("error_%s.txt", WTesterConfig.GetVersion()))
	}
	return fmt.Sprintf("error_%s", WTesterConfig.GetVersion())
}

// GetStatisticPath 统计文件路径
func (f *DbConfig) GetStatisticPath() string {
	if f.StorageType == constants.FileStorageType {
		return filepath.Join(f.Path, fmt.Sprintf("statistic_%s.txt", WTesterConfig.GetVersion()))

	}
	return fmt.Sprintf("statistic_%s", WTesterConfig.GetVersion())
}

// GetBatchSize 批量插入数据大小
func (f *DbConfig) GetBatchSize() int {
	if f.BatchSize == 0 {
		f.BatchSize = 100
	}
	return f.BatchSize
}

// Config 全局配置
type Config struct {
	Log     *LogConfig `json:"log" yaml:"log" mapstructure:"log"`             // 日志相关配置
	Db      *DbConfig  `json:"db" yaml:"db" mapstructure:"db"`                // 数据库 or 文件相关配置
	Version string     `json:"version" yaml:"version" mapstructure:"version"` // 记录测试版本，便于数据区分
}

func (c *Config) SetVersion(version string) {
	if version == "" {
		version = strings.Replace(time.Now().Format("20060102150405.000"), ".", "", 1)
	}
	c.Version = version
}
func (c *Config) GetVersion() string {
	return c.Version
}

// WTesterConfig 全局配置
var WTesterConfig *Config
