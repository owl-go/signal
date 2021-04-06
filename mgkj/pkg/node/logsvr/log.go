package logsvr

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"reflect"
	"sort"
)

type LoggerCfg struct {
	Filename   string
	MaxSize    int
	MaxAge     int
	MaxBackups int
	LocalTime  bool
	Compress   bool
	Encoder    string
	LogLevel   string
}
type Logger struct {
	lg *zap.Logger
}

func NewLogger(cfg LoggerCfg) *Logger {
	var encoder zapcore.Encoder
	var writeSync zapcore.WriteSyncer
	var debugLevel zapcore.Level
	logger := new(Logger)

	lumberJackLogger := &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
		MaxBackups: cfg.MaxBackups,
		LocalTime:  cfg.LocalTime,
		Compress:   cfg.Compress,
	}



	writeSync = zapcore.AddSync(lumberJackLogger)

	encodeConfig := zap.NewProductionEncoderConfig()
	encodeConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encodeConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	if cfg.Encoder == "json" {
		encoder = zapcore.NewJSONEncoder(encodeConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encodeConfig)
	}
	switch cfg.LogLevel {
	case "info":
		debugLevel = zapcore.InfoLevel
	case "debug":
		debugLevel = zapcore.DebugLevel
	case "error":
		debugLevel = zapcore.ErrorLevel
	case "warn":
		debugLevel = zapcore.WarnLevel
	case "panic":
		debugLevel = zapcore.PanicLevel
	default:
		debugLevel = zapcore.InfoLevel
	}
	core := zapcore.NewCore(encoder, writeSync, debugLevel)

	logger.lg = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return logger
}
func (l *Logger) With(args ...interface{}) []zap.Field {
	fs := make([]zap.Field, 0)
	fieds := make(map[string]interface{})
	if len(args)%2 == 0 && len(args) >= 2 {
		for i := 0; i < len(args); i += 2 {
			k, ok := args[i].(string)
			if ok {
				fieds[k] = args[i+1]
			}
		}
	} else if len(args) == 1 {
		t := reflect.TypeOf(args[0]).Kind()
		if t == reflect.Map {
			fieds = args[0].(map[string]interface{})
		}
	}
	if len(fieds) > 0 {
		keys := make([]string, 0, len(fieds))
		for k := range fieds {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fs = append(fs, zap.Any(k, fieds[k]))
		}
	}
	return fs
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	fields := l.With(args...)
	l.lg.With(fields...).Debug(msg)
}

func (l *Logger) Info(msg string, args ...interface{}) {
	fields := l.With(args...)
	l.lg.With(fields...).Info(msg)
}
func (l *Logger) Warn(msg string, args ...interface{}) {
	fields := l.With(args...)
	l.lg.With(fields...).Info(msg)
}
func (l *Logger) Error(msg string, args ...interface{}) {
	fields := l.With(args...)
	l.lg.With(fields...).Error(msg)
}
func (l *Logger) Panic(msg string, args ...interface{}) {
	fields := l.With(args...)
	l.lg.With(fields...).Panic(msg)

}
func (l *Logger) Sync() {
	l.lg.Sync()
}
