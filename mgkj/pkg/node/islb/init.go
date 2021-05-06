package node

import (
	logger2 "mgkj/infra/logger"
	"time"

	dis "mgkj/infra/discovery"
	db "mgkj/infra/redis"
	"mgkj/util"

	nprotoo "github.com/gearghost/nats-protoo"
)

const (
	redisShort  = 60 * time.Second
	redisKeyTTL = 24 * time.Hour
)

var (
	logger      *logger2.Logger
	nats        *nprotoo.NatsProtoo
	broadcaster *nprotoo.Broadcaster
	redis       *db.Redis
	redis1      *db.Redis
	node        *dis.ServiceNode
	watch       *dis.ServiceWatcher
)

// Init 初始化服务
func Init(serviceNode *dis.ServiceNode, ServiceWatcher *dis.ServiceWatcher, natsURL string, config, config1 db.Config, log *logger2.Logger) {
	// 赋值
	node = serviceNode
	watch = ServiceWatcher
	nats = nprotoo.NewNatsProtoo(util.GenerateNatsUrlString(natsURL))
	broadcaster = nats.NewBroadcaster(node.GetEventChannel())
	redis = db.NewRedis(config)
	redis1 = db.NewRedis(config1)
	logger = log
	// 启动
	handleRPCRequest(node.GetRPCChannel())
}

// Close 关闭连接
func Close() {
	if nats != nil {
		nats.Close()
	}
	if node != nil {
		node.Close()
	}
	if watch != nil {
		watch.Close()
	}
}
