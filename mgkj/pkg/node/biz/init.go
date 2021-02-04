package node

import (
	"mgkj/pkg/log"
	"mgkj/pkg/mq"
	"mgkj/pkg/proto"
	"mgkj/pkg/server"
	"mgkj/pkg/util"

	"github.com/cloudwebrtc/go-protoo/peer"
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
	go watch.WatchServiceNode("", WatchServiceCallBack)
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
	if node != nil {
		node.Close()
	}
	if watch != nil {
		watch.Close()
	}
}

// WatchServiceCallBack 查看所有的Node节点
func WatchServiceCallBack(state server.NodeStateType, node server.Node) {
	if state == server.ServerUp {
		log.Infof("WatchServiceCallBack node up %v", node)
	} else if state == server.ServerDown {
		log.Infof("WatchServiceCallBack node down %v", node)
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

	find := false
	var sfu *server.Node
	ch := make(chan int, 1)
	respIslb := func(resp map[string]interface{}) {
		// "method", proto.IslbToBizGetSfuInfo, "errorCode", 0, "rid", rid, "mid", mid, "nid", nid
		// "method", proto.IslbToBizGetSfuInfo, "errorCode", 1, "rid", rid, "mid", mid
		nErr := int(resp["errorCode"].(float64))
		if nErr == 0 {
			nid := util.Val(resp, "nid")
			if nid != "" {
				sfu = FindSfuNodeByID(nid)
				find = true
			}
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
func FindMediaIndoByMid(rid, mid string) (map[string]interface{}, bool) {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindMediaIndoByMid islb not found")
		return nil, false
	}

	find := false
	ch := make(chan int, 1)
	rsp := make(map[string]interface{})
	respIslb := func(resp map[string]interface{}) {
		// "method", proto.IslbToBizGetMediaInfo, "errorCode", 0, "rid", rid, "mid", mid, "tracks", tracks
		// "method", proto.IslbToBizGetMediaInfo, "errorCode", 1, "rid", rid, "mid", mid
		nErr := int(resp["errorCode"].(float64))
		if nErr == 0 {
			find = true
			rsp["tracks"] = resp["tracks"]
		}
		ch <- 0
	}
	amqp.RPCCallWithResp(server.GetRPCChannel(*islb), util.Map("method", proto.BizToIslbGetMediaInfo, "rid", rid, "mid", mid), respIslb)
	<-ch
	close(ch)
	return rsp, find
}

// FindMediaPubs 查询房间所有的其他人的发布流
func FindMediaPubs(peer *peer.Peer, rid string) bool {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindMediaPubs islb not found")
		return false
	}

	// 查询房间存在的发布流
	find := false
	ch := make(chan int, 1)
	respIslb := func(resp map[string]interface{}) {
		// "method", proto.IslbToBizGetMediaPubs, "errorCode", 1, "rid", rid
		// "method", proto.IslbToBizGetMediaPubs, "errorCode", 0, "rid", rid, "pubs", pubs
		nErr := int(resp["errorCode"].(float64))
		if nErr != 0 {
			ch <- 0
			return
		}

		if resp["pubs"] == nil {
			ch <- 0
			return
		}

		rid := resp["rid"].(string)
		pubs := resp["pubs"].([]interface{})
		for _, pub := range pubs {
			uid := pub.(map[string]interface{})["uid"].(string)
			mid := pub.(map[string]interface{})["mid"].(string)
			nid := pub.(map[string]interface{})["nid"].(string)
			tracks := pub.(map[string]interface{})["tracks"].(map[string]interface{})
			if mid != "" {
				peer.Notify(proto.BizToClientOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "nid", nid, "tracks", tracks))
			}
		}
		find = true
		ch <- 0
	}
	// 获取房间所有人的发布流
	amqp.RPCCallWithResp(server.GetRPCChannel(*islb), util.Map("method", proto.BizToIslbGetMediaPubs, "rid", rid, "uid", peer.ID()), respIslb)
	<-ch
	close(ch)
	return find
}

// FindPeerIsLive 查询peer是否还存活
func FindPeerIsLive(rid, uid string) bool {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindPeerIsLive islb not found")
		return false
	}

	find := false
	ch := make(chan int, 1)
	respIslb := func(resp map[string]interface{}) {
		// "method", proto.IslbToBizPeerLive, "errorCode", 1
		// "method", proto.IslbToBizPeerLive, "errorCode", 0
		nErr := int(resp["errorCode"].(float64))
		if nErr == 0 {
			find = true
		}
		ch <- 0
	}
	amqp.RPCCallWithResp(server.GetRPCChannel(*islb), util.Map("method", proto.BizToIslbPeerLive, "rid", rid, "uid", uid), respIslb)
	<-ch
	close(ch)
	return find
}
