package node

import (
	"mgkj/pkg/mq"
)

var (
	amqp  *mq.Amqp
	bizID string
)

// Init 初始化服务
func Init(id, mqURL string) {
	bizID = id
	amqp = mq.New(id, mqURL)
	handleRPCMsgs()
	handleBroadCastMsgs()
}

// Close 关闭连接
func Close() {
	if amqp != nil {
		amqp.Close()
	}
}
