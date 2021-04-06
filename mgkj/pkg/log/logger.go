package log

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap/buffer"
)

var (
	_pool = buffer.NewPool()
	// Get retrieves a buffer from the pool, creating one if necessary.
	Get = _pool.Get
)

// A Level is a logging priority. Higher levels are more important.
type Level int8

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel Level = iota - 1
	// InfoLevel is the default logging priority.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel
	// DPanicLevel logs are particularly important errors. In development the
	// logger panics after writing the message.
	DPanicLevel
	// PanicLevel logs a message, then panics.
	PanicLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel

	_minLevel = DebugLevel
	_maxLevel = FatalLevel
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case DPanicLevel:
		return "dpanic"
	case PanicLevel:
		return "panic"
	case FatalLevel:
		return "fatal"
	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}

// CapitalString returns an all-caps ASCII representation of the log level.
func (l Level) CapitalString() string {
	// Printing levels in all-caps is common enough that we should export this
	// functionality.
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case DPanicLevel:
		return "DPANIC"
	case PanicLevel:
		return "PANIC"
	case FatalLevel:
		return "FATAL"
	default:
		return fmt.Sprintf("LEVEL(%d)", l)
	}
}

type Logger struct {
	dc         string //分区名称
	name       string //节点名称
	nip        string //节点ip
	nid        string //节点id
	level      string //日志等级
	addCaller  bool
	callerSkip int
	Level      Level
	en         Entry
	log        log.Logger
}
type Entry struct {
	Level   Level
	Time    time.Time
	Message string
	Caller  EntryCaller
}

// EntryCaller represents the caller of a logging function.
type EntryCaller struct {
	Defined bool
	PC      uintptr
	File    string
	Line    int
}

func NewEntryCaller(pc uintptr, file string, line int, ok bool) EntryCaller {
	if !ok {
		return EntryCaller{}
	}
	return EntryCaller{
		PC:      pc,
		File:    file,
		Line:    line,
		Defined: true,
	}
}

// String returns the full path and line number of the caller.
func (ec EntryCaller) String() string {
	return ec.FullPath()
}
func (ec EntryCaller) FullPath() string {
	if !ec.Defined {
		return "undefined"
	}
	buf := Get()
	buf.AppendString(ec.File)
	buf.AppendByte(':')
	buf.AppendInt(int64(ec.Line))
	caller := buf.String()
	buf.Free()
	return caller
}

func (ec EntryCaller) TrimmedPath() string {
	if !ec.Defined {
		return "undefined"
	}
	idx := strings.LastIndexByte(ec.File, '/')
	if idx == -1 {
		return ec.FullPath()
	}
	// Find the penultimate separator.
	idx = strings.LastIndexByte(ec.File[:idx], '/')
	if idx == -1 {
		return ec.FullPath()
	}
	buf := Get()
	// Keep everything after the penultimate separator.
	buf.AppendString(ec.File[idx+1:])
	buf.AppendByte(':')
	buf.AppendInt(int64(ec.Line))
	caller := buf.String()
	buf.Free()
	return caller
}

func NewLogger(dc, name, nid, nip, level string, addCaller bool) *Logger {
	logger := &Logger{
		dc:        dc,
		name:      name,
		nid:       nid,
		nip:       nip,
		addCaller: addCaller,
	}
	switch level {
	case "info":
		logger.Level = InfoLevel
	case "debug":
		logger.Level = DebugLevel
	case "error":
		logger.Level = ErrorLevel
	case "warn":
		logger.Level = WarnLevel
	case "panic":
		logger.Level = PanicLevel
	default:
		logger.Level = InfoLevel
	}

	return logger
}
func (log *Logger) clone() *Logger {
	copy := *log
	return &copy
}

//设置输出
func (l *Logger) SetOutPut(filename string, maxsize int, maxage int, maxBackup int) {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxsize,
		MaxAge:     maxage,
		MaxBackups: maxBackup,
		LocalTime:  true,
		Compress:   true,
	}
	l.log.SetOutput(lumberJackLogger)
	l.log.SetFlags(0)
}
func (l *Logger) Print(msg string) {
	if l.log.Writer() == nil {
		return
	}
	l.log.Print(msg + "\n")
}
func (l *Logger) check(level Level, msg string) *Logger {
	const callerSkipOffset = 2
	clone := l.clone()
	en := Entry{
		Level:   level,
		Time:    time.Now(),
		Message: msg,
	}
	if level < clone.Level {
		return nil
	}
	if clone.addCaller {
		en.Caller = NewEntryCaller(runtime.Caller(l.callerSkip + callerSkipOffset))
	}
	clone.en = en
	return clone
}
func (l *Logger) Write(args ...interface{}) string {
	options := make(map[string]interface{})
	options["dc"] = l.dc
	options["name"] = l.name
	options["nip"] = l.nip
	options["nid"] = l.nid
	options["lv"] = l.en.Level.CapitalString()
	options["ts"] = l.en.Time
	options["msg"] = l.en.Message
	if l.addCaller {
		options["caller"] = l.en.Caller.TrimmedPath()
	}
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
			options[k] = fieds[k]
		}
	}
	data, _ := json.Marshal(options)

	return string(data)
}

func (l *Logger) Debug(msg string, args ...interface{}) string {
	if ce := l.check(DebugLevel, msg); ce != nil {
		return ce.Write(args...)
	}
	return ""
}

func (l *Logger) Info(msg string, args ...interface{}) string {
	if ce := l.check(InfoLevel, msg); ce != nil {
		return ce.Write(args...)
	}
	return ""
}

func (l *Logger) Warn(msg string, args ...interface{}) string {
	if ce := l.check(WarnLevel, msg); ce != nil {
		return ce.Write(args...)
	}
	return ""
}

func (l *Logger) Error(msg string, args ...interface{}) string {
	if ce := l.check(ErrorLevel, msg); ce != nil {
		return ce.Write(args...)
	}
	return ""
}

func (l *Logger) Panic(msg string, args ...interface{}) string {
	if ce := l.check(PanicLevel, msg); ce != nil {
		return ce.Write(args...)
	}
	return ""
}
