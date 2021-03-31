package node

import (
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"mgkj/pkg/db"
	"mgkj/pkg/server"
)

const (
	redisShort  = 60 * time.Second
	redisKeyTTL = 24 * time.Hour
)

var (
	protoo      *nprotoo.NatsProtoo
	broadcaster *nprotoo.Broadcaster
	redis       *db.Redis
	redis1      *db.Redis
	node        *server.ServiceNode
	watch       *server.ServiceWatcher
)

// Init 初始化服务
func Init(serviceNode *server.ServiceNode, ServiceWatcher *server.ServiceWatcher, natsURL string, config, config1 db.Config) {
	// 赋值
	node = serviceNode
	watch = ServiceWatcher
	protoo = nprotoo.NewNatsProtoo(natsURL)
	broadcaster = protoo.NewBroadcaster(node.GetEventChannel())
	redis = db.NewRedis(config)
	redis1 = db.NewRedis(config1)
	// 启动
	handleRPCRequest(node.GetRPCChannel())
}

// Close 关闭连接
func Close() {
	if protoo != nil {
		protoo.Close()
	}
	if node != nil {
		node.Close()
	}
	if watch != nil {
		watch.Close()
	}
}
