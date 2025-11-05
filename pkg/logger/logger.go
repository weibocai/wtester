package logger

import (
	"os"
	"time"

	"github.com/wtester/pkg/config"
	"github.com/wtester/pkg/library"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger *zap.Logger // 日志

// InitLogger 初始化全局日志系统
func InitLogger() {
	if err := library.CreateFileIfNotExists(config.WTesterConfig.Log.Path); err != nil {
		panic(err)
	}
	// 配置 zap 编码器
	encoderFile := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 带有颜色，用于console输出日志
	encoderConsoleColor := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 配置日志级别
	var level = zapcore.InfoLevel

	stdout := zapcore.AddSync(os.Stdout)
	syncWriter := &zapcore.BufferedWriteSyncer{
		WS: zapcore.AddSync(&lumberjack.Logger{
			Filename:  config.WTesterConfig.Log.Path, // ⽇志⽂件路径
			MaxSize:   100,                           // 单位为MB,默认为512MB
			MaxAge:    5,                             // 文件最多保存多少天
			LocalTime: true,                          // 采用本地时间
			Compress:  false,                         // 是否压缩日志
		}),
		Size:          4096,
		FlushInterval: time.Second, // 每分钟刷新一次
	}
	// 创建 Core
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewConsoleEncoder(encoderFile), syncWriter, level),
		zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConsoleColor), stdout, level),
	)

	// 创建 Logger
	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	defer func(Log *zap.Logger) {
		err := Log.Sync()
		if err != nil {

		}
	}(Logger)
}
