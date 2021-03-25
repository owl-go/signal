package biz

import (
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/util"
	"strings"
)

// handleBroadCastMsgs 处理广播消息
func handleBroadcast(msg map[string]interface{}, subj string) {
	go func(msg map[string]interface{}) {
		defer util.Recover("biz.handleBroadcast")
		log.Infof("biz.handleBroadcast msg=%v", msg)

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

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("SfuRemoveStream islb rpc not found")
		return
	}

	msid := strings.Split(key, "/")
	if len(msid) < 6 {
		log.Errorf("SfuRemoveStream key is err")
		return
	}

	rid := msid[3]
	uid := msid[5]
	mid := msid[7]
	rpc.AsyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
}
