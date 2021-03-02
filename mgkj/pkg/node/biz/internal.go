package node

import (
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/server"
	"mgkj/pkg/util"
	"strings"
)

// handleBroadCastMsgs 处理广播消息
func handleBroadcast(msg map[string]interface{}, subj string) {
	go func(msg map[string]interface{}) {
		defer util.Recover("biz.handleBroadcast")
		method := util.Val(msg, "method")
		data := msg["data"].(map[string]interface{})
		log.Infof("OnBroadcast: method=%s, data=%v", method, data)
		rid := util.Val(data, "rid")
		uid := util.Val(data, "uid")
		switch method {
		case proto.IslbToBizOnJoin:
			/* "method", proto.IslbToBizOnJoin, "rid", rid, "uid", uid, "info", info */
			NotifyAllWithoutID(rid, uid, proto.BizToClientOnJoin, data)
		case proto.IslbToBizOnLeave:
			/* "method", proto.IslbToBizOnLeave, "rid", rid, "uid", uid */
			NotifyAllWithoutID(rid, uid, proto.BizToClientOnLeave, data)
		case proto.IslbToBizOnStreamAdd:
			/* "method", proto.IslbToBizOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "tracks", tracks, "nid", nid */
			NotifyAllWithoutID(rid, uid, proto.BizToClientOnStreamAdd, data)
		case proto.IslbToBizOnStreamRemove:
			/* "method", proto.IslbToBizOnStreamRemove, "rid", rid, "uid", uid, "mid", mid */
			NotifyAllWithoutID(rid, uid, proto.BizToClientOnStreamRemove, data)
		case proto.IslbToBizBroadcast:
			/* "method", proto.IslbToBizBroadcast, "rid", rid, "uid", uid, "data", data */
			NotifyAllWithoutID(rid, uid, proto.BizToClientBroadcast, data)
		case proto.SfuToBizOnStreamRemove:
			mid := util.Val(data, "mid")
			SfuRemoveStream(mid)
		}
	}(msg)
}

// SfuRemoveStream 处理移除流
func SfuRemoveStream(key string) {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		return
	}

	msid := strings.Split(key, "/")
	if len(msid) < 6 {
		log.Errorf("key is err")
		return
	}
	rid := msid[3]
	uid := msid[5]
	mid := msid[7]
	rpc := protoo.NewRequestor(server.GetRPCChannel(*islb))
	rpc.AsyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
}
