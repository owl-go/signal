package util

import (
	"testing"
)

func TestGenerateNatsUrlString(t *testing.T) {
	str := GenerateNatsUrlString("192.168.1.1,192.168.1.2,192.168.1.3")
	t.Log(str)
	str1 := GenerateNatsUrlString("127.0.0.1")
	t.Log(str1)
}
