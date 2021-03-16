package sfu

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"mgkj/pkg/log"
	mp "mgkj/pkg/mediasoup"
	"mgkj/pkg/proto"
	"mgkj/pkg/rtc"
	"mgkj/pkg/server"
	"mgkj/pkg/util"
	"sync"
	"time"
)

const (
	statCycle = time.Second * 5
)

var (
	protoo      *nprotoo.NatsProtoo
	broadcaster *nprotoo.Broadcaster
	node        *server.ServiceNode
	watch       *server.ServiceWatcher
	routersLock sync.RWMutex
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
	go updatePayload()
	go mp.InitWork()
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

// update node's payload
func updatePayload() {
	t := time.NewTicker(statCycle)
	defer t.Stop()
	for range t.C {
		var streamcnt int = 0
		routersLock.RLock()
		pubs := rtc.GetRouters()
		for _, pub := range pubs {
			streamcnt++
			subs := pub.GetSubs()
			streamcnt += len(subs)
		}
		routersLock.RUnlock()
		node.UpdateNodePayload(streamcnt)
	}
}
