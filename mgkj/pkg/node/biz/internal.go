package node

import (
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/util"
)

// strToMap make string value to map
func strToMap(msg map[string]interface{}, key string) {
	value := util.Val(msg, key)
	if value != "" {
		mValue := util.Unmarshal(value)
		msg[key] = mValue
	}
}

// handleRPCMsgResp response msg from islb
func handleRPCMsgResp(corrID, from, resp string, msg map[string]interface{}) {
	log.Infof("biz.handleRPCMsgResp corrID=%s, from=%s, resp=%s msg=%v", corrID, from, resp, msg)
	switch resp {
	case proto.IslbGetPeers:
		strToMap(msg, "info")
		amqp.Emit(corrID, msg)
	case proto.IslbGetPubs:
		strToMap(msg, "minfo")
		amqp.Emit(corrID, msg)
	default:
		log.Warnf("biz.handleRPCMsgResp invalid protocol corrID=%s, from=%s, resp=%s msg=%v", corrID, from, resp, msg)
	}
}

// handleRPCMsgs 处理其他模块发送过来的消息
func handleRPCMsgs() {
	rpcMsgs, err := amqp.ConsumeRPC()
	if err != nil {
		log.Errorf(err.Error())
		return
	}

	go func() {
		for rpcm := range rpcMsgs {
			msg := util.Unmarshal(string(rpcm.Body))
			from := rpcm.ReplyTo
			log.Infof("biz.handleRPCMsgs msg=%v", msg)
			resp := util.Val(msg, "response")
			if resp != "" {
				corrID := rpcm.CorrelationId
				handleRPCMsgResp(corrID, from, resp, msg)
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
			msg := util.Unmarshal(string(rpcm.Body))
			method := util.Val(msg, "method")
			if method == "" {
				continue
			}
			log.Infof("biz.handleBroadCastMsgs msg=%v", msg)

			rid := util.Val(msg, "rid")
			uid := util.Val(msg, "uid")
			strToMap(msg, "data")
			switch method {
			case proto.IslbClientOnJoin:
				strToMap(msg, "info")
				NotifyAllWithoutID(rid, uid, proto.ClientOnJoin, msg)
			case proto.IslbClientOnLeave:
				strToMap(msg, "info")
				NotifyAllWithoutID(rid, uid, proto.ClientOnLeave, msg)
			case proto.IslbOnStreamAdd:
				strToMap(msg, "minfo")
				NotifyAllWithoutID(rid, uid, proto.ClientOnStreamAdd, msg)
			case proto.IslbOnStreamRemove:
				strToMap(msg, "minfo")
				NotifyAllWithoutID(rid, uid, proto.ClientOnStreamRemove, msg)
			case proto.IslbOnBroadcast:
				strToMap(msg, "data")
				NotifyAllWithoutID(rid, uid, proto.ClientBroadcast, msg)
			}
		}
	}()
}
