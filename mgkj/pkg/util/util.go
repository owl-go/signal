package util

import (
	"encoding/json"
	"math/rand"
	"runtime"
	"runtime/debug"
	"strings"

	"mgkj/pkg/log"

	"github.com/pion/webrtc/v2"
)

// MarshalStr 将map转换成string
func MarshalStr(m map[string]string) string {
	byt, err := json.Marshal(m)
	if err != nil {
		log.Errorf(err.Error())
		return ""
	}
	return string(byt)
}

// Marshal 将map转换成string
func Marshal(m map[string]interface{}) string {
	byt, err := json.Marshal(m)
	if err != nil {
		log.Errorf(err.Error())
		return ""
	}
	return string(byt)
}

// Val 从map结构中获取值
func Val(msg map[string]interface{}, key string) string {
	if msg == nil {
		return ""
	}
	val := msg[key]
	if val == nil {
		return ""
	}
	switch val.(type) {
	case string:
		return val.(string)
	case map[string]interface{}:
		return Marshal(val.(map[string]interface{}))
	default:
		log.Errorf("util.Val val=%v", val)
		return ""
	}
}

// Unmarshal 将string转换成map
func Unmarshal(str string) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		log.Errorf(err.Error())
		return data
	}
	return data
}

// Map 将数据组装成map对象
func Map(args ...interface{}) map[string]interface{} {
	if len(args)%2 != 0 {
		return nil
	}
	msg := make(map[string]interface{})
	for i := 0; i < len(args)/2; i++ {
		msg[args[2*i].(string)] = args[2*i+1]
	}
	return msg
}

// Map2 将数据组装成map对象
func Map2(args ...interface{}) map[string]string {
	if len(args)%2 != 0 {
		return nil
	}
	msg := make(map[string]string)
	for i := 0; i < len(args)/2; i++ {
		msg[args[2*i].(string)] = args[2*i+1].(string)
	}
	return msg
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

// RandStr 随机数
func RandStr(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(65, 90))
	}
	return string(bytes)
}

// Recover 抓panic
func Recover(flag string) {
	_, _, l, _ := runtime.Caller(1)
	if err := recover(); err != nil {
		log.Errorf("[%s] Recover panic line => %v", flag, l)
		log.Errorf("[%s] Recover err => %v", flag, err)
		debug.PrintStack()
	}
}

// IsVideo 判断pt是否是视频
func IsVideo(pt uint8) bool {
	if pt == webrtc.DefaultPayloadTypeVP8 ||
		pt == webrtc.DefaultPayloadTypeVP9 ||
		pt == webrtc.DefaultPayloadTypeH264 {
		return true
	}
	return false
}

func ProcessUrlString(url string) []string {
	urls := strings.Split(url, ",")
	for i, s := range urls {
		urls[i] = strings.TrimSpace(s)
	}
	return urls
}

func GenerateNatsUrlString(url string) string {
	var result string
	urls := strings.Split(url, ",")
	length := len(urls)
	for i, s := range urls {
		result += "nats://" + strings.TrimSpace(s)
		if length-1 != i {
			result += ","
		}
	}
	return result

}
