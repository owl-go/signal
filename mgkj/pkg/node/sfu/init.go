package sfu

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/rtc"
	"mgkj/pkg/server"
	"mgkj/pkg/util"
)

var (
	protoo      *nprotoo.NatsProtoo
	broadcaster *nprotoo.Broadcaster
	node        *server.ServiceNode
	watch       *server.ServiceWatcher
)

// Init 初始化服务
func Init(serviceNode *server.ServiceNode, ServiceWatcher *server.ServiceWatcher, natsURL string) {
	// 赋值
	node = serviceNode
	watch = ServiceWatcher
	protoo = nprotoo.NewNatsProtoo(natsURL)
	broadcaster = protoo.NewBroadcaster(node.GetEventChannel())
	// 启动
	handleRPCRequest(node.GetRPCChannel())
	checkRTC()
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

// checkRTC send `stream-remove` msg to islb when some pub has been cleaned
func checkRTC() {
	log.Infof("SFU.checkRTC start")
	go func() {
		for mid := range rtc.CleanChannel {
			broadcaster.Say(proto.SfuToBizOnStreamRemove, util.Map("mid", mid))
		}
	}()
}
