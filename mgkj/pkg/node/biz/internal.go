package biz

import (
	"fmt"
	"mgkj/pkg/proto"
	"mgkj/util"
	"strings"

	nprotoo "github.com/gearghost/nats-protoo"
)

// 接收biz消息处理
func handleRPCRequest(rpcID string) {
	logger.Infof(fmt.Sprintf("biz.handleRequest: rpcID=%s", rpcID), "rpcid", rpcID)
	nats.OnRequest(rpcID, handleRpcMsg)
}

// 处理biz的rpc请求
func handleRpcMsg(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
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

	room := GetRoom(rid)
	if room != nil {
		peer := room.room.GetPeer(uid)
		if peer != nil {
			peer.Notify(proto.BizToClientOnKick, util.Map("rid", rid, "uid", uid))
			peer.Close()
		}
	}

	rpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
	rpc.SyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))
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
		/* "method", proto.IslbToBizOnJoin, "rid", rid, "uid", uid, "info", info */
		NotifyAllWithoutID(rid, uid, proto.BizToClientOnJoin, data)
	case proto.IslbToBizOnLeave:
		/* "method", proto.IslbToBizOnLeave, "rid", rid, "uid", uid, "info", info */
		NotifyAllWithoutID(rid, uid, proto.BizToClientOnLeave, data)
	case proto.IslbToBizOnStreamAdd:
		/* "method", proto.IslbToBizOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "nid", nid, "minfo", data["minfo"] */
		NotifyAllWithoutID(rid, uid, proto.BizToClientOnStreamAdd, data)
	case proto.IslbToBizOnStreamRemove:
		/* "method", proto.IslbToBizOnStreamRemove, "rid", rid, "uid", uid, "mid", mid */
		NotifyAllWithoutID(rid, uid, proto.BizToClientOnStreamRemove, data)
		//when publisher's stream remove,stop all the stream timer
		mid := util.Val(data, "mid")
		updateSubTimersByMID(rid, mid)
	case proto.IslbToBizBroadcast:
		/* "method", proto.IslbToBizBroadcast, "rid", rid, "uid", uid, "data", data */
		NotifyAllWithoutID(rid, uid, proto.BizToClientBroadcast, data)
	case proto.SfuToBizOnStreamRemove:
		mid := util.Val(data, "mid")
		SfuRemoveStream(mid)
	}
}

// SfuRemoveStream 处理移除流
func SfuRemoveStream(key string) {
	islb := FindIslbNode()
	if islb == nil {
		logger.Errorf("biz.SfuRemoveStream islb node not found", "mid", key)
		return
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		logger.Errorf("biz.SfuRemoveStream islb rpc not found", "mid", key)
		return
	}

	msid := strings.Split(key, "/")
	if len(msid) < 6 {
		logger.Errorf("biz.SfuRemoveStream key is err", "mid", key)
		return
	}

	rid := msid[3]
	uid := msid[5]
	mid := msid[7]
	rpc.AsyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
}

func updateSubTimersByMID(rid, mid string) {
	roomNode := GetRoom(rid)
	if roomNode != nil {
		peers := roomNode.room.GetPeers()
		for _, peer := range peers {
			timer := peer.GetStreamTimer()
			if timer != nil && !timer.IsStopped() {
				removedStreams, isModeChanged := timer.RemoveStreamByMID(mid)
				//it must be video change to audio,cuz it at least have one last dummy audio stream in the end.
				if isModeChanged {
					timer.Stop()
					err := reportStreamTiming(timer, true, false)
					if err != nil {
						logger.Errorf(fmt.Sprintf("biz.removeSubStreamByMID reportStreamTiming when removed MID:%s stream, err:%v", mid, err), "rid", timer.RID, "uid", timer.UID,
							"mid", mid)
					}
					timer.Renew()
				} else {
					if removedStreams != nil && removedStreams[0].MediaType != "audio" && timer.GetCurrentMode() != "audio" {
						isResolutionChanged := timer.UpdateResolution()
						if isResolutionChanged {
							timer.Stop()
							isNotLastStream := timer.GetStreamsCount() > 1
							err := reportStreamTiming(timer, true, isNotLastStream)
							if err != nil {
								logger.Errorf(fmt.Sprintf("biz.removeSubStreamByMID reportStreamTiming when removed MID:%s stream, err:%v", mid, err), "rid", timer.RID, "uid", timer.UID,
									"mid", mid)
							}
							timer.Renew()
						}
					}
				}
			}
		}
	}
}
