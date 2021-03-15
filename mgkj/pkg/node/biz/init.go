package biz

import (
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/server"
	"mgkj/pkg/util"
	"mgkj/pkg/ws"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
)

var (
	nats  *nprotoo.NatsProtoo
	rpcs  = make(map[string]*nprotoo.Requestor)
	node  *server.ServiceNode
	watch *server.ServiceWatcher
)

// Init 初始化服务
func Init(serviceNode *server.ServiceNode, ServiceWatcher *server.ServiceWatcher, natsURL string) {
	node = serviceNode
	watch = ServiceWatcher
	nats = nprotoo.NewNatsProtoo(natsURL)
	rpcs = make(map[string]*nprotoo.Requestor)
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
func WatchServiceCallBack(state server.NodeStateType, node server.Node) {
	if state == server.ServerUp {
		log.Infof("WatchServiceCallBack node up %v", node)
		if node.Name == "islb" || node.Name == "sfu" {
			eventID := server.GetEventChannel(node)
			log.Infof("handleIslbBroadCast: eventID => [%s]", eventID)
			nats.OnBroadcast(eventID, handleBroadcast)
		}

		id := node.Nid
		_, found := rpcs[id]
		if !found {
			rpcID := server.GetRPCChannel(node)
			rpcs[id] = nats.NewRequestor(rpcID)
		}
	} else if state == server.ServerDown {
		log.Infof("WatchServiceCallBack node down %v", node.Nid)
		if _, found := rpcs[node.Nid]; found {
			delete(rpcs, node.Nid)
		}
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
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("FindSfuNodeByMid islb rpc not found")
		return nil
	}

	resp, err := rpc.SyncRequest(proto.BizToIslbGetSfuInfo, util.Map("rid", rid, "mid", mid))
	if err != nil {
		log.Errorf(err.Reason)
		return nil
	}

	log.Infof("FindSfuNodeByMid resp ==> %v", resp)

	find = false
	var sfu *server.Node
	nErr := int(resp["errorCode"].(float64))
	if nErr == 0 {
		nid := util.Val(resp, "nid")
		if nid != "" {
			sfu = FindSfuNodeByID(nid)
			find = true
		}
	}

	if find {
		return sfu
	}
	return nil
}

// FindMediaPubs 查询房间所有的其他人的发布流
func FindMediaPubs(peer *ws.Peer, rid string) bool {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindMediaPubs islb not found")
		return false
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("FindMediaPubs islb rpc not found")
		return false
	}

	resp, err := rpc.SyncRequest(proto.BizToIslbGetMediaPubs, util.Map("rid", rid, "uid", peer.ID()))
	if err != nil {
		log.Errorf(err.Reason)
		return false
	}

	log.Infof("FindMediaPubs resp ==> %v", resp)

	nErr := int(resp["errorCode"].(float64))
	if nErr != 0 {
		log.Errorf("FindMediaPubs errorCode = %d", nErr)
		return false
	}

	if resp["pubs"] == nil {
		log.Errorf("FindMediaPubs pubs = nil")
		return false
	}

	roomid := resp["rid"].(string)
	pubs := resp["pubs"].([]interface{})
	for _, pub := range pubs {
		uid := pub.(map[string]interface{})["uid"].(string)
		mid := pub.(map[string]interface{})["mid"].(string)
		nid := pub.(map[string]interface{})["nid"].(string)
		minfo := pub.(map[string]interface{})["minfo"].(map[string]interface{})
		if mid != "" {
			peer.Notify(proto.BizToClientOnStreamAdd, util.Map("rid", roomid, "uid", uid, "mid", mid, "nid", nid, "minfo", minfo))
		}
	}
	return true
}

// FindPeerIsLive 查询peer是否还存活
func FindPeerIsLive(rid, uid string) bool {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindPeerIsLive islb not found")
		return false
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("FindPeerIsLive islb rpc not found")
		return false
	}

	resp, err := rpc.SyncRequest(proto.BizToIslbPeerLive, util.Map("rid", rid, "uid", uid))
	if err != nil {
		log.Errorf(err.Reason)
		return false
	}

	// "method", proto.IslbToBizPeerLive, "errorCode", 1
	// "method", proto.IslbToBizPeerLive, "errorCode", 0
	log.Infof("FindMediaPubs resp ==> %v", resp)

	find = false
	nErr := int(resp["errorCode"].(float64))
	if nErr == 0 {
		find = true
	}
	return find
}
