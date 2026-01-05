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

// GetDns 连接串
func (f *DbConfig) GetDns() string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", f.GetUsername(), f.GetPassword(), f.GetHost(), f.GetPort(), f.GetDatabase())
	return dsn
}

// GetResultPath 全量日志存储路径
func (f *DbConfig) GetResultPath() string {
	return fmt.Sprintf("result_%s", WTesterConfig.GetVersion())
}

// GetErrorPath 失败文件路径
func (f *DbConfig) GetErrorPath() string {
	return fmt.Sprintf("error_%s", WTesterConfig.GetVersion())
}

// GetStatisticPath 统计文件路径
func (f *DbConfig) GetStatisticPath() string {
	return fmt.Sprintf("statistic_%s", WTesterConfig.GetVersion())
}

// GetBatchSize 批量插入数据大小
func (f *DbConfig) GetBatchSize() int {
	if f.BatchSize == 0 {
		f.BatchSize = 100
	}
	return f.BatchSize
}

// FileConfig 文件配置
type FileConfig struct {
	Path    string `json:"path" yaml:"path" mapstructure:"path"`
	BufSize int    `json:"bufSize" yaml:"bufSize" mapstructure:"bufSize"`
}

// GetResultPath 全量日志存储路径
func (f *FileConfig) GetResultPath() string {
	return filepath.Join(f.Path, fmt.Sprintf("result_%s.txt", WTesterConfig.GetVersion()))
}

// GetErrorPath 失败文件路径
func (f *FileConfig) GetErrorPath() string {
	return filepath.Join(f.Path, fmt.Sprintf("error_%s.txt", WTesterConfig.GetVersion()))
}

// GetStatisticPath 统计文件路径
func (f *FileConfig) GetStatisticPath() string {
	return filepath.Join(f.Path, fmt.Sprintf("statistic_%s.txt", WTesterConfig.GetVersion()))
}

// GetResultBufSize 批次缓存大小
func (f *FileConfig) GetResultBufSize() int {
	return f.BufSize
}

// ResultConfig 结果存储配置
type ResultConfig struct {
	StorageType constants.StorageType `json:"storageType" yaml:"storageType" mapstructure:"storageType"` // 存储介质
	FileConfig  *FileConfig           `json:"fileConfig" yaml:"fileConfig" mapstructure:"fileConfig"`    // 文本存储
	DbConfig    *DbConfig             `json:"dbConfig" yaml:"dbConfig" mapstructure:"dbConfig"`          // 数据库存储
}

// Config 全局配置
type Config struct {
	Log     *LogConfig    `json:"log" yaml:"log" mapstructure:"log"`
	Result  *ResultConfig `json:"result" yaml:"result" mapstructure:"result"`
	Version string        `json:"version" yaml:"version" mapstructure:"version"`
}

func (c *Config) SetVersion() {
	c.Version = strings.Replace(time.Now().Format("20060102150405.000"), ".", "", 1)
	fmt.Println(c.Version)
}
func (c *Config) GetVersion() string {
	return c.Version
}

// WTesterConfig 全局配置
var WTesterConfig *Config
