package node

import (
	"time"

	dis "mgkj/infra/discovery"
	"mgkj/pkg/db"
	lgr "mgkj/pkg/logger"
	"mgkj/util"

	nprotoo "github.com/gearghost/nats-protoo"
)

const (
	redisShort  = 60 * time.Second
	redisKeyTTL = 24 * time.Hour
)

var (
	logger      *lgr.Logger
	protoo      *nprotoo.NatsProtoo
	broadcaster *nprotoo.Broadcaster
	redis       *db.Redis
	redis1      *db.Redis
	node        *dis.ServiceNode
	watch       *dis.ServiceWatcher
)

// Init 初始化服务
func Init(serviceNode *dis.ServiceNode, ServiceWatcher *dis.ServiceWatcher, natsURL string, config, config1 db.Config,
	l *lgr.Logger) {
	// 赋值
	node = serviceNode
	watch = ServiceWatcher
	protoo = nprotoo.NewNatsProtoo(util.GenerateNatsUrlString(natsURL))
	broadcaster = protoo.NewBroadcaster(node.GetEventChannel())
	redis = db.NewRedis(config)
	redis1 = db.NewRedis(config1)
	logger = l
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
