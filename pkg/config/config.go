package config

import "github.com/wtester/pkg/constants"

// LogConfig 日志配置
type LogConfig struct {
	Path string `json:"path" yaml:"path" mapstructure:"path"` // 日志存储的路径
}

// DbConfig 数据库配置
type DbConfig struct {
	Host     string `json:"host" yaml:"host" mapstructure:"host"`
	Port     int    `json:"port" yaml:"port" mapstructure:"port"`
	Username string `json:"username" yaml:"username" mapstructure:"username"`
	Password string `json:"password" yaml:"password" mapstructure:"password"`
	Database string `json:"database" yaml:"database" mapstructure:"database"`
}

// FileConfig 文件配置
type FileConfig struct {
	Path    string `json:"path" yaml:"path" mapstructure:"path"`
	BufSize int    `json:"bufSize" yaml:"bufSize" mapstructure:"bufSize"`
}

// ResultConfig 结果存储配置
type ResultConfig struct {
	StorageType constants.StorageType `json:"storageType" yaml:"storageType" mapstructure:"storageType"` // 存储介质
	FileConfig  *FileConfig           `json:"fileConfig" yaml:"fileConfig" mapstructure:"fileConfig"`    // 文本存储
	DbConfig    *DbConfig             `json:"dbConfig" yaml:"dbConfig" mapstructure:"dbConfig"`          // 数据库存储
}

// Config 全局配置
type Config struct {
	Log    *LogConfig    `json:"log" yaml:"log" mapstructure:"log"`
	Result *ResultConfig `json:"result" yaml:"result" mapstructure:"result"`
}

// WTesterConfig 全局配置
var WTesterConfig *Config
