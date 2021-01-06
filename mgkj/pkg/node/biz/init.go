package node

import (
	"mgkj/pkg/mq"
)

var (
	amqp *mq.Amqp
)

// Init 初始化服务
func Init(mqURL string, rpc, event string) {
	amqp = mq.New(rpc, event, mqURL)
	handleRPCMsgs()
	handleBroadCastMsgs()
}

// Close 关闭连接
func Close() {
	if amqp != nil {
		amqp.Close()
	}
}
