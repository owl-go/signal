package logsvr

import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"log"
	"reflect"
	"runtime"
	"testing"
)

func TestLog(t *testing.T) {
	lcfg := LoggerCfg{
		Filename:   "./test.log",
		MaxSize:    1,
		MaxAge:     5,
		MaxBackups: 30,
		LocalTime:  true,
		Compress:   true,
		Encoder:    "json",
		LogLevel:   "debug",
	}
	logger := NewLogger(lcfg)
	for i := 0; i < 10; i++ {
		logger.Debug("test", "key1", "value2", "key2", "value2")
		logger.Info("test", "key1", "value2", "key2", "value2")
	}
	logger.Sync()
	data := make(map[string]interface{})
	data["id"] = 12
	checkType := func(agrs ...interface{}) {
		t := reflect.TypeOf(agrs[0]).Kind()
		if t == reflect.Map {
			fields := agrs[0].(map[string]interface{})
			fmt.Println(fields)
		}
	}

	checkType(data)
	testCaller()
}
func testCaller() {
	pc, file, line, ok := runtime.Caller(1)
	fmt.Println(pc, file, line, ok)
	f := runtime.FuncForPC(pc)
	fmt.Println(f.Name())
}
func TestSpiteLog(t *testing.T) {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "./test.log",
		MaxSize:    1,
		MaxAge:     2,
		MaxBackups: 20,
		LocalTime:  true,
		Compress:   true,
	}
	log.SetOutput(lumberJackLogger)
	log.SetFlags(0)
	for i := 0; i < 10000000; i++ {
		log.Print("test msg test msg test msg \n")
	}
}
