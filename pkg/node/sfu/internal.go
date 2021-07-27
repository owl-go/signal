package sfu

import (
	"fmt"
	"signal/infra/monitor"
	"signal/pkg/proto"
	"signal/pkg/rtc"
	"signal/util"

	nprotoo "github.com/gearghost/nats-protoo"
)

// handleRPCMsgs 处理其他模块发送过来的消息
func handleRPCRequest(rpcID string) {

	logger.Infof(fmt.Sprintf("sfu.handleRequest: rpcID=%s", rpcID), "rpcid", rpcID)

	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		go func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
			defer util.Recover("sfu.handleRPCRequest")
			//log.Infof("sfu.handleRPCRequest recv rpc=%s, request=%v", rpcID, request)
			logger.Infof(fmt.Sprintf("sfu.handleRPCRequest recv request=%v", request), "rpcid", rpcID)

			method := request["method"].(string)
			data := request["data"].(map[string]interface{})

			var result map[string]interface{}
			err := &nprotoo.Error{Code: 400, Reason: fmt.Sprintf("Unkown method [%s]", method)}
			if method != "" {
				processingTime := monitor.NewProcessingTimeGauge(method)
				processingTime.Start()
				switch method {
				case proto.BizToSfuPublish:
					result, err = publish(data)
				case proto.BizToSfuUnPublish:
					result, err = unpublish(data)
				case proto.BizToSfuSubscribe:
					result, err = subscribe(data)
				case proto.BizToSfuUnSubscribe:
					result, err = unsubscribe(data)
				default:
					//log.Warnf("sfu.handleRPCRequest invalid protocol method=%s data=%v", method, data)
					logger.Warnf(fmt.Sprintf("sfu.handleRPCRequest invalid protocol method=%s data=%v", method, data), "rpcid", rpcID)
				}
				processingTime.Stop()
				rpcProcessingTimeGauge.WithLabelValues(method).Set(processingTime.GetDuration())
			}
			if err != nil {
				reject(err.Code, err.Reason)
			} else {
				accept(result)
			}
		}(request, accept, reject)
	})
}

/*
	"method", proto.BizToSfuPublish, "rid", rid, "uid", uid, "minfo", minfo, "jsep", jsep
*/
// publish 处理发布流
func publish(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Infof(fmt.Sprintf("sfu.publish msg=%v", msg))
	// 获取参数
	if msg["jsep"] == nil {
		return nil, &nprotoo.Error{Code: -1, Reason: "can't find jsep"}
	}

	jsep, ok := msg["jsep"].(map[string]interface{})
	if !ok {
		return nil, &nprotoo.Error{Code: -1, Reason: "jsep can't transform to map"}
	}

	sdp := util.Val(jsep, "sdp")
	rid := util.Val(msg, "rid")
	uid := util.Val(msg, "uid")
	mid := fmt.Sprintf("%s#%s", uid, util.RandStr(6))

	options := msg["minfo"]
	if options != nil {
		options, ok := msg["minfo"].(map[string]interface{})
		if ok {
			key := proto.GetMediaPubKey(rid, uid, mid)
			router := rtc.GetOrNewRouter(key)
			resp, err := router.AddPub(sdp, mid, node.NodeInfo().Nip, options)
			if err != nil {
				return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("AddPub err:%v", err)}
			}
			return util.Map("jsep", util.Map("type", "answer", "sdp", resp), "mid", mid), nil
		}
	}
	return nil, &nprotoo.Error{Code: -1, Reason: "can't find media info"}
}

/*
	"method", proto.BizToSfuUnPublish, "rid", rid, "uid", uid, "mid", mid
*/
// unpublish 处理取消发布流
func unpublish(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Infof(fmt.Sprintf("sfu.unpublish msg=%v", msg))
	// 获取参数
	rid := util.Val(msg, "rid")
	uid := util.Val(msg, "uid")
	mid := util.Val(msg, "mid")

	key := proto.GetMediaPubKey(rid, uid, mid)
	rtc.DelRouter(key)
	return util.Map(), nil
}

/*
	"method", proto.BizToSfuSubscribe, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo, "jsep", jsep
*/
// subscribe 处理订阅流
func subscribe(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Infof(fmt.Sprintf("sfu.subscribe msg=%v", msg))
	// 获取参数
	if msg["jsep"] == nil {
		return nil, &nprotoo.Error{Code: -1, Reason: "can't find jsep"}
	}

	jsep, ok := msg["jsep"].(map[string]interface{})
	if !ok {
		return nil, &nprotoo.Error{Code: -1, Reason: "jsep can't transform to map"}
	}

	sdp := util.Val(jsep, "sdp")
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	sid := util.Val(msg, "uid")
	uid := proto.GetUIDFromMID(mid)
	subID := fmt.Sprintf("%s#%s", sid, util.RandStr(6))

	options := msg["minfo"]
	if options != nil {
		minfo, ok := options.(map[string]interface{})
		if ok {
			key := proto.GetMediaPubKey(rid, uid, mid)
			router := rtc.GetRouter(key)
			if router == nil {
				return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("can't get router:%s", key)}
			}

			resp, err := router.AddSub(sdp, subID, node.NodeInfo().Nip, minfo)
			if err != nil {
				return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("AddSub err:%v", err)}
			}
			return util.Map("jsep", util.Map("type", "answer", "sdp", resp), "mid", subID, "uid", uid), nil
		}
	}
	return nil, &nprotoo.Error{Code: -1, Reason: "can't find media info"}
}

/*
	"method", proto.BizToSfuUnSubscribe, "rid", rid, "uid", uid, "mid", mid
*/
// unsubscribe 处理取消订阅流
func unsubscribe(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Infof(fmt.Sprintf("sfu.unsubscribe msg=%v", msg))
	// 获取参数
	mid := util.Val(msg, "mid")
	rtc.MapRouter(func(id string, r *rtc.Router) {
		subs := r.GetSubs()
		for sid := range subs {
			if sid == mid {
				r.DelSub(mid)
				return
			}
		}
	})
	return util.Map(), nil
}
