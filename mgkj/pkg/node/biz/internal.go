package biz

import (
	"fmt"
	"mgkj/pkg/proto"
	"mgkj/util"
	"strings"
)

// handleBroadCastMsgs 处理广播消息
func handleBroadcast(msg map[string]interface{}, subj string) {
	//go func(msg map[string]interface{}) {
	func(msg map[string]interface{}) {
		defer util.Recover("biz.handleBroadcast")
		//log.Infof("biz.handleBroadcast msg=%v", msg)
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
	}(msg)
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
