package node

import (
	"mgkj/pkg/log"
	"mgkj/pkg/mq"
	"mgkj/pkg/proto"
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
	node = serviceNode
	watch = ServiceWatcher
	go watch.WatchServiceNode("", WatchServiceNodes)
	amqp = mq.New(node.GetRPCChannel(), node.GetEventChannel(), mqURL)
	// 启动
	handleRPCMsgs()
	handleBroadCastMsgs()
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

// FindIslbNode 查询全局的可用的islb节点
func FindIslbNode() *server.Node {
	servers, find := watch.GetNodes("islb")
	if find {
		for _, node := range servers {
			return &node
		}
	}
	return nil
}

// FindSfuNodeByID 查询指定区域下的可用的sfu节点
func FindSfuNodeByID(nid string) *server.Node {
	sfu, find := watch.GetNodeByID(nid)
	if find {
		return sfu
	}
	return nil
}

// FindSfuNodeByPayload 查询指定区域下的可用的sfu节点
func FindSfuNodeByPayload() *server.Node {
	sfu, find := watch.GetNodeByPayload(node.NodeInfo().Ndc, "sfu")
	if find {
		return sfu
	}
	return nil
}

// FindSfuNodeByMid 根据mid向islb查询指定的sfu节点
func FindSfuNodeByMid(rid, mid string) *server.Node {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindSfuNodeByMid islb not found")
		return nil
	}

	ch := make(chan int, 1)
	var sfu *server.Node
	find := false
	respIslb := func(resp map[string]interface{}) {
		// "method", proto.IslbToBizGetSfuInfo, "rid", rid, "mid", mid, "errorCode", 0, "nid", nid
		// "method", proto.IslbToBizGetSfuInfo, "rid", rid, "mid", mid, "errorCode", 1, "errorReason", "mid is not find"
		nid := util.Val(resp, "nid")
		if nid != "" {
			sfu = FindSfuNodeByID(nid)
			find = true
		}
		ch <- 0
	}
	amqp.RPCCallWithResp(server.GetRPCChannel(*islb), util.Map("method", proto.BizToIslbGetSfuInfo, "rid", rid, "mid", mid), respIslb)
	<-ch
	close(ch)
	if find {
		return sfu
	}
	return nil
}

// FindMediaIndoByMid 根据mid向islb查询指定的流信息
func FindMediaIndoByMid(rid, mid string) (string, bool) {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindMediaIndoByMid islb not found")
		return "", false
	}

	ch := make(chan int, 1)
	var minfo string
	find := false
	respIslb := func(resp map[string]interface{}) {
		// "method", proto.IslbToBizGetMediaInfo, "rid", rid, "mid", mid, "errorCode", 0, "minfo", minfo
		// "method", proto.IslbToBizGetMediaInfo, "rid", rid, "mid", mid, "errorCode", 1, "errorReason", "mid is not find"
		minfo := util.Val(resp, "minfo")
		if minfo != "" {
			find = true
		}
		ch <- 0
	}
	amqp.RPCCallWithResp(server.GetRPCChannel(*islb), util.Map("method", proto.BizToIslbGetMediaInfo, "rid", rid, "mid", mid), respIslb)
	<-ch
	close(ch)
	return minfo, find
}

// FindPeerIsLive 查询peer是否还存活
func FindPeerIsLive(rid, uid string) bool {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindPeerIsLive islb not found")
		return false
	}

	var live bool = false
	ch := make(chan int, 1)
	respIslb := func(resp map[string]interface{}) {
		// "method", proto.IslbToBizPeerLive, "errorCode", 1
		// "method", proto.IslbToBizPeerLive, "errorCode", 0
		nLive := int(resp["errorCode"].(float64))
		if nLive == 0 {
			live = true
		}
		ch <- 0
	}
	amqp.RPCCallWithResp(server.GetRPCChannel(*islb), util.Map("method", proto.BizToIslbPeerLive, "rid", rid, "uid", uid), respIslb)
	<-ch
	close(ch)
	return live
}
