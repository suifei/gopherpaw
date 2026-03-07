// Package logger provides structured logging based on zap.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger abstracts logging operations.
type Logger interface {
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Sync() error
}

// Config holds logger initialization options.
type Config struct {
	Level  string // debug, info, warn, error
	Format string // json, console
}

// global is the default logger instance.
var global *zap.Logger

// Init initializes the global logger from config.
func Init(cfg Config) error {
	var zapCfg zap.Config
	switch cfg.Format {
	case "console":
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.Encoding = "console"
	default:
		zapCfg = zap.NewProductionConfig()
		zapCfg.Encoding = "json"
	}

	level := parseLevel(cfg.Level)
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	logger, err := zapCfg.Build()
	if err != nil {
		return err
	}

	if global != nil {
		_ = global.Sync()
	}
	global = logger
	return nil
}

func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// L returns the global logger. Panics if Init was not called.
func L() *zap.Logger {
	if global == nil {
		global, _ = zap.NewProduction()
	}
	return global
}

// Sugared returns a sugared logger for key-value logging.
func Sugared() *zap.SugaredLogger {
	return L().Sugar()
}

// zapLogger adapts *zap.SugaredLogger to Logger interface.
type zapLogger struct {
	s *zap.SugaredLogger
}

func (z *zapLogger) Debug(args ...any) { z.s.Debug(args...) }
func (z *zapLogger) Info(args ...any)  { z.s.Info(args...) }
func (z *zapLogger) Warn(args ...any)  { z.s.Warn(args...) }
func (z *zapLogger) Error(args ...any) { z.s.Error(args...) }
func (z *zapLogger) Sync() error       { return z.s.Sync() }

// New creates a Logger from Config.
func New(cfg Config) (Logger, error) {
	if err := Init(cfg); err != nil {
		return nil, err
	}
	return &zapLogger{s: L().Sugar()}, nil
}
