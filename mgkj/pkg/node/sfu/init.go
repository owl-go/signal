package sfu

import (
	"mgkj/pkg/proto"
	"mgkj/pkg/rtc"
	"mgkj/pkg/server"
	"mgkj/pkg/util"
	"sync"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
)

const (
	statCycle = time.Second * 10
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
	protoo = nprotoo.NewNatsProtoo("nats://" + natsURL)
	broadcaster = protoo.NewBroadcaster(node.GetEventChannel())
	// 启动
	rtc.InitSfu()
	handleRPCRequest(node.GetRPCChannel())
	go checkRTC()
	go updatePayload()
}

// Close 关闭连接
func Close() {
	rtc.FreeSfu()
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

// checkRTC 通知信令流被移除
func checkRTC() {
	for mid := range rtc.CleanPub {
		broadcaster.Say(proto.SfuToBizOnStreamRemove, util.Map("mid", mid))
	}
}

// updatePayload 更新sfu服务器负载
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
