package node

import (
	"time"

	"mgkj/pkg/db"
	"mgkj/pkg/log"
	"mgkj/pkg/mq"
	"mgkj/pkg/server"
)

const (
	redisShort  = 60 * time.Second
	redisKeyTTL = 24 * time.Hour
)

var (
	amqp  *mq.Amqp
	redis *db.Redis
	node  *server.ServiceNode
	watch *server.ServiceWatcher
)

// Init 初始化服务
func Init(serviceNode *server.ServiceNode, ServiceWatcher *server.ServiceWatcher, mqURL string, config db.Config) {
	// 赋值
	node = serviceNode
	watch = ServiceWatcher
	amqp = mq.New(node.GetRPCChannel(), node.GetEventChannel(), mqURL)
	redis = db.NewRedis(config)
	// 启动
	handleRPCMsgs()
}

// Close 关闭连接
func Close() {
	if amqp != nil {
		amqp.Close()
	}
}

// WatchServiceNodes 查看所有的Node节点
func WatchServiceNodes(state server.NodeStateType, node server.Node) {
	if state == server.ServerUp {
		log.Infof("WatchServiceNodes node up %v", node)
	} else if state == server.ServerDown {
		log.Infof("WatchServiceNodes node down %v", node)
	}
}
