package sfu

import (
	"mgkj/pkg/log"
	"mgkj/pkg/mq"
	"mgkj/pkg/proto"
	"mgkj/pkg/rtc"
	"mgkj/pkg/server"
	"mgkj/pkg/util"
)

var (
	amqp  *mq.Amqp
	node  *server.ServiceNode
	watch *server.ServiceWatcher
)

// Init 初始化服务
func Init(serviceNode *server.ServiceNode, ServiceWatcher *server.ServiceWatcher, mqURL string) {
	// 赋值
	node = serviceNode
	watch = ServiceWatcher
	amqp = mq.New(node.GetRPCChannel(), node.GetEventChannel(), mqURL)
	// 启动
	handleRPCMsgs()
	checkRTC()
}

// Close 关闭连接
func Close() {
	if amqp != nil {
		amqp.Close()
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
			msg := util.Map("method", proto.SfuToBizOnStreamRemove, "mid", mid)
			amqp.BroadCast(msg)
		}
	}()
}
