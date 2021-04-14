package sfu

import (
	dis "mgkj/infra/discovery"
	logger2 "mgkj/infra/logger"
	"mgkj/pkg/proto"
	"mgkj/pkg/rtc"
	"mgkj/util"
	"sync"
	"time"

	nprotoo "github.com/gearghost/nats-protoo"
)

const (
	statCycle = time.Second * 10
)

var (
	logger      *logger2.Logger
	protoo      *nprotoo.NatsProtoo
	broadcaster *nprotoo.Broadcaster
	node        *dis.ServiceNode
	watch       *dis.ServiceWatcher
	routersLock sync.RWMutex
)

// Init 初始化服务
func Init(serviceNode *dis.ServiceNode, ServiceWatcher *dis.ServiceWatcher, natsURL string, l *logger2.Logger) {
	// 赋值
	logger = l
	node = serviceNode
	watch = ServiceWatcher
	protoo = nprotoo.NewNatsProtoo(util.GenerateNatsUrlString(natsURL))
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
