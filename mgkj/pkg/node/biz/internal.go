package node

import (
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/util"
)

// strToMap make string value to map
/*
func strToMap(msg map[string]interface{}, key string) {
	value := util.Val(msg, key)
	if value != "" {
		mValue := util.Unmarshal(value)
		msg[key] = mValue
	}
}*/

// handleRPCMsgResp response msg from islb
func handleRPCMsgResp(corrID, from, resp string, msg map[string]interface{}) {
	log.Infof("biz.handleRPCMsgResp corrID=%s, from=%s, resp=%s msg=%v", corrID, from, resp, msg)
	switch resp {
	case proto.IslbGetSfuInfo:
		amqp.Emit(corrID, msg)
	case proto.IslbGetMediaInfo:
		amqp.Emit(corrID, msg)
	case proto.IslbGetMediaPubs:
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
			corrID := rpcm.CorrelationId
			log.Infof("biz.handleRPCMsgs msg=%v", msg)

			resp := util.Val(msg, "response")
			if resp != "" {
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
			switch method {
			case proto.IslbClientOnJoin:
				NotifyAllWithoutID(rid, uid, proto.ClientOnJoin, msg)
			case proto.IslbClientOnLeave:
				NotifyAllWithoutID(rid, uid, proto.ClientOnLeave, msg)
			case proto.IslbOnStreamAdd:
				NotifyAllWithoutID(rid, uid, proto.ClientOnStreamAdd, msg)
			case proto.IslbOnStreamRemove:
				NotifyAllWithoutID(rid, uid, proto.ClientOnStreamRemove, msg)
			}
		}
	}()
}
