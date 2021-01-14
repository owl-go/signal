package mq

import (
	"fmt"
	"testing"
	"time"
)

func TestFunction(t *testing.T) {
	//amqp := New("module1", "amqp://guest:guest@localhost:5672/")
	amqp := New("rpc1", "board", "amqp://zhouwq:123456@8.129.55.41:5672/admin")

	rpcMsgs, err := amqp.ConsumeRPC()
	if err != nil {
		fmt.Println(err.Error())
	}

	broadCastMsgs, err := amqp.ConsumeBroadcast()
	if err != nil {
		fmt.Println(err.Error())
	}

	go func() {
		for msg := range rpcMsgs {
			fmt.Println(string(msg.Body))
		}
	}()

	go func() {
		for msg := range broadCastMsgs {
			fmt.Println(string(msg.Body))
		}
	}()

	for i := 0; i < 3; i++ {
		m := make(map[string]interface{})
		m["key"] = fmt.Sprintf("rpc %d", i)
		amqp.RPCCall("rpc1", m, "")
		amqp.RPCCall("rpc-shenzhen_islb_1", m, "")
		m["key"] = fmt.Sprintf("broadcast %d", i)
		amqp.BroadCast(m)
		time.Sleep(time.Second)
	}
}
