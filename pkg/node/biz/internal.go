package biz

import (
	"fmt"
	"signal/pkg/proto"
	"signal/util"

	nprotoo "github.com/gearghost/nats-protoo"
)

// 接收biz消息处理
func handleRPCRequest(rpcID string) {
	logger.Infof(fmt.Sprintf("biz.handleRequest: rpcID=%s", rpcID), "rpcid", rpcID)
	nats.OnRequest(rpcID, handleRpcMsg)
}

// 处理biz的rpc请求
func handleRpcMsg(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	go func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		defer util.Recover("biz.handleRPCRequest")
		logger.Infof(fmt.Sprintf("biz.handleRPCRequest recv request=%v", request))

		method := request["method"].(string)
		data := request["data"].(map[string]interface{})
		var result map[string]interface{}
		err := &nprotoo.Error{Code: 400, Reason: fmt.Sprintf("Unkown method [%s]", method)}

		switch method {
		/* 处理和biz服务器通信 */
		case proto.BizToBizOnKick:
			result, err = peerKick(data)
		}
		if err != nil {
			reject(err.Code, err.Reason)
		} else {
			accept(result)
		}
	}(request, accept, reject)
}

/*
	"method", proto.BizToBizOnKick, "rid", rid, "uid", uid
*/
// 踢出房间
func peerKick(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		logger.Errorf("biz.peerKick islb node not found", "uid", uid, "rid", rid)
		return nil, &nprotoo.Error{Code: -1, Reason: "islb node not found"}
	}
	rpc, find := rpcs[islb.Nid]
	if !find {
		logger.Errorf("biz.peerKick islb rpc not found", "uid", uid, "rid", rid)
		return nil, &nprotoo.Error{Code: -1, Reason: "islb node not found"}
	}

	rpc.SyncRequest(proto.BizToIslbOnLiveRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
	rpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
	rpc.SyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))

	peer := GetPeer(rid, uid)
	if peer != nil {
		peer.Notify(proto.BizToClientOnKick, util.Map("rid", rid, "uid", uid))
		peer.Close()
	}
	DelPeer(rid, uid)
	return util.Map(), nil
}

// handleBroadCastMsgs 处理广播消息
func handleBroadcast(msg map[string]interface{}, subj string) {
	defer util.Recover("biz.handleBroadcast")
	logger.Infof(fmt.Sprintf("biz.handleBroadcast msg=%v", msg))

	method := util.Val(msg, "method")
	data := msg["data"].(map[string]interface{})

	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	switch method {
	case proto.IslbToBizOnJoin:
		/* "method", proto.IslbToBizOnJoin, "rid", rid, "uid", uid, "nid", nid, "info", info */
		NotifyAllWithoutID(rid, uid, proto.BizToClientOnJoin, data)
	case proto.IslbToBizOnLeave:
		/* "method", proto.IslbToBizOnLeave, "rid", rid, "uid", uid */
		NotifyAllWithoutID(rid, uid, proto.BizToClientOnLeave, data)
	case proto.IslbToBizOnStreamAdd:
		/* "method", proto.IslbToBizOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "nid", nid, "minfo", data["minfo"] */
		NotifyAllWithoutID(rid, uid, proto.BizToClientOnStreamAdd, data)
	case proto.IslbToBizOnStreamRemove:
		/* "method", proto.IslbToBizOnStreamRemove, "rid", rid, "uid", uid, "mid", mid */
		NotifyAllWithoutID(rid, uid, proto.BizToClientOnStreamRemove, data)
	case proto.IslbToBizBroadcast:
		/* "method", proto.IslbToBizBroadcast, "rid", rid, "uid", uid, "data", data */
		NotifyAllWithoutID(rid, uid, proto.BizToClientBroadcast, data)
	case proto.IslbToBizOnLiveAdd:
		NotifyAllWithoutID(rid, uid, proto.BizToClientOnLiveStreamAdd, data)
	case proto.IslbToBizOnLiveRemove:
		NotifyAllWithoutID(rid, uid, proto.BizToClientOnLiveStreamRemove, data)
		stopLiveStreamTimerByRIDUID(rid, uid)
	}
}

func stopLiveStreamTimerByRIDUID(rid, uid string) {
	roomNode := GetRoom(rid)
	if roomNode != nil {
		peer := roomNode.room.GetPeer(uid)
		if peer != nil && peer.GetLiveStreamTimer() != nil {
			if !peer.GetLiveStreamTimer().IsStopped() {
				peer.GetLiveStreamTimer().Stop()
				reportLiveStreamTiming(peer.GetLiveStreamTimer())
				peer.SetLiveStreamTimer(nil)
			}
		}
	}
}
