package issr

import (
	"encoding/json"
	"fmt"
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/util"
	"time"
)

// handleRPCMsgs 处理其他模块发送过来的消息
func handleRPCRequest(rpcID string) {
	log.Infof("handleRequest: rpcID => [%v]", rpcID)
	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		go func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
			defer util.Recover("issr.handleRPCRequest")
			log.Infof("issr.handleRPCRequest recv request=%v", request)
			method := request["method"].(string)
			data := request["data"].(map[string]interface{})
			log.Infof("method => %s, data => %v", method, data)

			var result map[string]interface{}
			err := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))

			if method != "" {
				switch method {
				case proto.BizToIssrReportStreamState:
					result, err = report(data)
				default:
					log.Warnf("issr.handleRPCRequest invalid protocol method=%s data=%v", method, data)
				}
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
	"method", proto.BizToSsReportStreamState, "rid", rid, "uid", uid, "mid", mid, "sid", sid, "resolution", hd, "seconds", seconds
*/
// report 上报拉流计时数据
func report(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("ss report msg => %v", msg)
	// 判断参数
	if msg["appid"] == nil {
		return util.Map("errorCode", 401), nil
	}
	islb := getIslbRequestor()
	if islb == nil {
		log.Errorf("can't find islb requestor")
		return util.Map("errorCode", 402), nil
	}

	resp, nerr := islb.SyncRequest(proto.IssrToIslbReportStreamState, msg)
	if nerr != nil {
		log.Errorf("report rpc err => %v", nerr)
		return util.Map("errorCode", 403), nil
	}

	code := int(resp["errorCode"].(float64))

	if code != 0 {
		log.Errorf("report stream state fail")
		return util.Map("errorCode", 404), nil
	}
	seconds := int64(msg["seconds"].(float64))
	//change seconds to minutes
	delete(msg, "seconds")
	minutes := seconds / 60
	if seconds%60 != 0 {
		minutes += 1
	}
	msg["usage"] = minutes
	timestamp := time.Now().Unix()
	msg["timestamp"] = timestamp
	//
	str, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("json marshal failed => %v", err)
		return util.Map("errorCode", 405), nil
	}
	log.Infof("report json => %s", string(str))
	err = kafkaProducer.Produce("Livs-Usage-Event", string(str))
	if err != nil {
		log.Infof("report produce error => %v", err)
		return util.Map("errorCode", 406), nil
	}
	log.Infof("kafka message sent.")
	return util.Map("errorCode", 0), nil
}
