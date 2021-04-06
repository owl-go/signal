package logsvr

import (
	"fmt"

	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/util"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
)

// handleRPCMsgs 处理其他模块发送过来的消息
func handleRPCRequest(rpcID string) {
	log.Infof("handleRPCRequest: rpcID => [%v]", rpcID)
	logger.Info("WatchServiceCallBack node up", "rpcID", rpcID, "nodeId", rpcID)
	nats.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		go func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
			defer util.Recover("logsvr.handleRPCRequest")
			log.Infof("logsvr.handleRPCRequest recv request=%v", request)
			method := request["method"].(string)
			data := request["data"].(map[string]interface{})
			var result map[string]interface{}
			err := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))
			switch method {
			case proto.ToLogsvr:
				logger.Print(data["data"].(string))
			default:
				log.Warnf("logsvr.handleRPCMsgs invalid protocol method=%s", method)
			}
			if err != nil {
				reject(err.Code, err.Reason)
			} else {
				accept(result)
			}
		}(request, accept, reject)
	})
}
