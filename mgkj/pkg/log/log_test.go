package log

import (
	"testing"
)

func TestDebug(t *testing.T) {
	logger := NewLogger("shenzhen", "dist", "shenzhen_dist_1", "192.168.1.72", "debug", true)
	msg := logger.Debug("test", "nodename", "123435")
	info := logger.Info("test", "nodename", "123435")
	//fmt.Println(msg)
	//fmt.Println(info)
	logger.SetOutPut("./test.log", 1, 1, 1)
	logger.Print(msg)
	logger.Print(info)
}
