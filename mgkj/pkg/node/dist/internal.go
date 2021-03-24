package dist

import (
	"fmt"
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/util"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
)

/*
	"method", proto.DistToDistCall, "caller", caller, "callee", callee, "rid", rid, "nid", nid
*/
func dist2distCall(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	callee := msg["callee"].(string)
	peer := peers[callee]
	if peer != nil {
		data := make(map[string]interface{})
		data["caller"] = msg["caller"]
		data["rid"] = msg["rid"]
		data["nid"] = msg["nid"]
		data["type"] = msg["type"]
		peer.Notify(proto.DistToClientCall, data)
	}
	return util.Map(), nil
}

/*
	"method", proto.DistToDistAnswer, "caller", caller, "callee", callee, "rid", rid, "nid", nid
*/
func dist2distAnswer(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	caller := msg["caller"].(string)
	peer := peers[caller]
	if peer != nil {
		data := make(map[string]interface{})
		data["callee"] = msg["callee"]
		data["rid"] = msg["rid"]
		data["nid"] = msg["nid"]
		peer.Notify(proto.DistToClientAnswer, data)
	}
	return util.Map(), nil
}

/*
	"method", proto.DistToDistReject, "caller", caller, "callee", callee, "rid", rid, "nid", nid, "code", code
*/
func dist2distReject(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	caller := msg["caller"].(string)
	peer := peers[caller]
	if peer != nil {
		data := make(map[string]interface{})
		data["callee"] = msg["callee"]
		data["rid"] = msg["rid"]
		data["nid"] = msg["nid"]
		data["code"] = msg["code"]
		peer.Notify(proto.DistToClientReject, data)
	}
	return util.Map(), nil
}

// handleRPCMsgs 处理其他模块发送过来的消息
func handleRPCRequest(rpcID string) {
	log.Infof("handleRPCRequest: rpcID => [%v]", rpcID)
	nats.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		go func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
			defer util.Recover("dist.handleRPCRequest")
			log.Infof("dist.handleRPCRequest recv request=%v", request)

			method := request["method"].(string)
			data := request["data"].(map[string]interface{})
			log.Infof("method => %s, data => %v", method, data)

			var result map[string]interface{}
			err := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))

			switch method {
			case proto.DistToDistCall:
				result, err = dist2distCall(data)
			case proto.DistToDistAnswer:
				result, err = dist2distAnswer(data)
			case proto.DistToDistReject:
				result, err = dist2distReject(data)
			default:
				log.Warnf("dist.handleRPCMsgs invalid protocol method=%s", method)
			}
			if err != nil {
				reject(err.Code, err.Reason)
			} else {
				accept(result)
			}
		}(request, accept, reject)
	})
}
