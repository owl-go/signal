package node

import (
	"time"

	"mgkj/pkg/db"
	"mgkj/pkg/mq"
)

const (
	redisKeyTTL = 24 * time.Hour
)

var (
	amqp  *mq.Amqp
	redis *db.Redis
)

// Init 初始化服务
func Init(mqURL string, config db.Config, rpc, event string) {
	amqp = mq.New(rpc, event, mqURL)
	redis = db.NewRedis(config)
	handleRPCMsgs()
}

// Close 关闭连接
func Close() {
	if amqp != nil {
		amqp.Close()
	}
}
