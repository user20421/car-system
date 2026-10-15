package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *zap.Logger

// Init 初始化全局日志器。mode 为 release 时使用 JSON 格式，debug 使用彩色控制台格式。
func Init(mode string) {
	var cfg zap.Config
	if mode == "release" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	var err error
	L, err = cfg.Build()
	if err != nil {
		panic("初始化日志器失败: " + err.Error())
	}
}

// Sync 程序退出前刷盘
func Sync() {
	if L != nil {
		_ = L.Sync()
	}
}
