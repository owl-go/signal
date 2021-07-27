package node

import (
	logger2 "signal/infra/logger"
	"time"

	dis "signal/infra/discovery"
	"signal/infra/monitor"
	db "signal/infra/redis"
	"signal/util"

	nprotoo "github.com/gearghost/nats-protoo"
)

const (
	redisShort  = 60 * time.Second
	redisKeyTTL = 24 * time.Hour
)

var (
	logger             *logger2.Logger
	nats               *nprotoo.NatsProtoo
	broadcaster        *nprotoo.Broadcaster
	redis              *db.Redis
	redis1             *db.Redis
	node               *dis.ServiceNode
	watch              *dis.ServiceWatcher
	rpcCounter         = monitor.NewMonitorCounter("islb_rpc_counter", "islb rpc request counter", []string{"method"})
	rpcProcessingGauge = monitor.NewMonitorGauge("islb_rpc_processing_time", "islb rpc request processing time", []string{"method"})
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
	go watch.WatchServiceNode("", WatchServiceCallBack)
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

// WatchServiceCallBack 查看所有的Node节点
func WatchServiceCallBack(state dis.NodeStateType, node dis.Node) {
	if state == dis.ServerUp {
		// 处理sfu和mcu发送的广播
		if node.Name == "sfu" || node.Name == "mcu" {
			eventID := dis.GetEventChannel(node)
			nats.OnBroadcastWithGroup(eventID, "islb", handleBroadcast)
		}
	}
}
