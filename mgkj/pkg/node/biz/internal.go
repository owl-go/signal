package node

import (
	"encoding/json"
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/util"
)

// handleRPCMsgs 处理其他模块发送过来的消息
func handleRPCMsgs() {
	rpcMsgs, err := amqp.ConsumeRPC()
	if err != nil {
		log.Errorf(err.Error())
		return
	}

	go func() {
		defer util.Recover("biz.handleRPCMsgs")
		for rpcm := range rpcMsgs {
			var msg map[string]interface{}
			err := json.Unmarshal(rpcm.Body, &msg)
			if err != nil {
				log.Errorf("biz handleRPCMsgs Unmarshal err = %s", err.Error())
			}

			from := rpcm.ReplyTo
			corrID := rpcm.CorrelationId
			log.Infof("biz.handleRPCMsgs msg=%v", msg)

			resp := util.Val(msg, "method")
			if resp != "" {
				switch resp {
				case proto.IslbToBizGetSfuInfo:
					amqp.Emit(corrID, msg)
				case proto.IslbToBizGetMediaInfo:
					amqp.Emit(corrID, msg)
				case proto.IslbToBizGetMediaPubs:
					amqp.Emit(corrID, msg)
				case proto.SfuToBizPublish:
					amqp.Emit(corrID, msg)
				case proto.SfuToBizSubscribe:
					amqp.Emit(corrID, msg)
				case proto.SfuToBizOnStreamRemove:
					amqp.Emit(corrID, msg)
				default:
					log.Warnf("biz.handleRPCMsgResp invalid protocol corrID=%s, from=%s, resp=%s msg=%v", corrID, from, resp, msg)
				}
			}
		}
	}()
}

// handleBroadCastMsgs 处理广播消息
func handleBroadCastMsgs() {
	broadCastMsgs, err := amqp.ConsumeBroadcast()
	if err != nil {
		log.Errorf(err.Error())
	}

	go func() {
		defer util.Recover("biz.handleBroadCastMsgs")
		for rpcm := range broadCastMsgs {
			var msg map[string]interface{}
			err := json.Unmarshal(rpcm.Body, &msg)
			if err != nil {
				log.Errorf("biz handleBroadCastMsgs Unmarshal err = %s", err.Error())
			}

			method := util.Val(msg, "method")
			if method == "" {
				continue
			}
			log.Infof("biz.handleBroadCastMsgs msg=%v", msg)

			rid := util.Val(msg, "rid")
			uid := util.Val(msg, "uid")
			switch method {
			case proto.IslbToBizOnJoin:
				/* "method", proto.IslbToBizOnJoin, "rid", rid, "uid", uid, "info", info */
				NotifyAllWithoutID(rid, uid, proto.BizToClientOnJoin, msg)
			case proto.IslbToBizOnLeave:
				/* "method", proto.IslbToBizOnLeave, "rid", rid, "uid", uid, "info", info */
				NotifyAllWithoutID(rid, uid, proto.BizToClientOnLeave, msg)
			case proto.IslbToBizOnStreamAdd:
				/* "method", proto.IslbToBizOnStreamAdd, "rid", rid, "uid", uid, "nid", nid, "mid", mid, "minfo", minfo */
				NotifyAllWithoutID(rid, uid, proto.BizToClientOnStreamAdd, msg)
			case proto.IslbToBizOnStreamRemove:
				/* "method", proto.IslbToBizOnStreamRemove, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo */
				NotifyAllWithoutID(rid, uid, proto.BizToClientOnStreamRemove, msg)
			case proto.IslbToBizBroadcast:
				/* "method", proto.IslbToBizBroadcast, "rid", rid, "uid", uid, "data", data */
				NotifyAllWithoutID(rid, uid, proto.BizToClientBroadcast, msg)
			}
		}
	}()
}
