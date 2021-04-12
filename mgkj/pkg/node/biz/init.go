package biz

import (
	dis "mgkj/infra/discovery"
	"mgkj/pkg/log"
	lgr "mgkj/pkg/logger"
	"mgkj/pkg/proto"
	"mgkj/pkg/ws"
	"mgkj/util"

	nprotoo "github.com/gearghost/nats-protoo"
)

var (
	logger *lgr.Logger
	nats   *nprotoo.NatsProtoo
	node   *dis.ServiceNode
	watch  *dis.ServiceWatcher
	rpcs   = make(map[string]*nprotoo.Requestor)
)

// Init 初始化服务
func Init(serviceNode *dis.ServiceNode, ServiceWatcher *dis.ServiceWatcher, natsURL string, l *lgr.Logger) {
	node = serviceNode
	watch = ServiceWatcher
	nats = nprotoo.NewNatsProtoo(util.GenerateNatsUrlString(natsURL))
	logger = l
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
		// 判断是否广播节点
		if node.Name == "islb" || node.Name == "sfu" {
			eventID := dis.GetEventChannel(node)
			nats.OnBroadcast(eventID, handleBroadcast)
		}

		id := node.Nid
		_, found := rpcs[id]
		if !found {
			rpcID := dis.GetRPCChannel(node)
			rpcs[id] = nats.NewRequestor(rpcID)
		}
	} else if state == dis.ServerDown {
		delete(rpcs, node.Nid)
	}
}

// FindIslbNode 查询全局的可用的islb节点
func FindIslbNode() *dis.Node {
	servers, find := watch.GetNodes("islb")
	if find {
		for _, node := range servers {
			return &node
		}
	}
	return nil
}

// FindSfuNodeByID 查询指定区域下的可用的sfu节点
func FindSfuNodeByID(nid string) *dis.Node {
	sfu, find := watch.GetNodeByID(nid)
	if find {
		return sfu
	}
	return nil
}

// FindSfuNodeByPayload 查询指定区域下的可用的sfu节点
func FindSfuNodeByPayload() *dis.Node {
	sfu, find := watch.GetNodeByPayload(node.NodeInfo().Ndc, "sfu")
	if find {
		return sfu
	}
	return nil
}

// FindSfuNodeByMid 根据mid向islb查询指定的sfu节点
func FindSfuNodeByMid(rid, mid string) *dis.Node {
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

	var sfu *dis.Node
	nid := util.Val(resp, "nid")
	if nid != "" {
		sfu = FindSfuNodeByID(nid)
	}

	return sfu
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

	if resp["pubs"] == nil {
		log.Errorf("FindMediaPubs pubs is nil")
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

	log.Infof("FindPeerIsLive resp ==> %v", resp)

	return true
}

// findIssrNode 查询全局的可用的Issr节点
func findIssrNode() *dis.Node {
	servers, find := watch.GetNodes("issr")
	if find {
		for _, node := range servers {
			return &node
		}
	}
	return nil
}

// getIssrRequestor 查询issr服务的节点id
func getIssrRequestor() *nprotoo.Requestor {
	issr := findIssrNode()
	if issr == nil {
		log.Errorf("find issr node not found")
		return nil
	}

	find := false
	rpc, find := rpcs[issr.Nid]
	if !find {
		log.Errorf("issr rpc not found")
		return nil
	}
	return rpc
}
