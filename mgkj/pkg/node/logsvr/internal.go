package logsvr

import (
	"fmt"

	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/util"

	nprotoo "github.com/gearghost/nats-protoo"
)

// handleRPCMsgs 处理其他模块发送过来的消息
func handleRPCRequest(rpcID string) {
	log.Infof("handleRPCRequest: rpcID => [%v]", rpcID)
	nats.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		go func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
			defer util.Recover("logsvr.handleRPCRequest")
			method := request["method"].(string)
			data := request["data"].(map[string]interface{})
			var result map[string]interface{}
			err := &nprotoo.Error{Code: 400, Reason: fmt.Sprintf("Unkown method [%s]", method)}
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
