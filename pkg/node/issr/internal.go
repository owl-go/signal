package issr

import (
	"encoding/json"
	"fmt"
	"signal/infra/monitor"
	"signal/pkg/proto"
	"signal/pkg/timing"
	"signal/util"
	"time"

	nprotoo "github.com/gearghost/nats-protoo"
)

// 处理广播消息
func handleBroadcast(msg map[string]interface{}, subj string) {
	go func(msg map[string]interface{}, subj string) {
		logger.Infof(fmt.Sprintf("issr.handleBroadcast recv msg=%v", msg))
		method := util.Val(msg, "method")
		data := msg["data"].(map[string]interface{})
		switch method {
		case proto.SfuToIssrOnSubscribeAdd:
			subStreamStart(data)
		case proto.SfuToIssrOnSubscribeRemove:
			subStreamEnd(data)
		}
	}(msg, subj)
}

// handleRPCMsgs 处理其他模块发送过来的消息
func handleRPCRequest(rpcID string) {

	logger.Infof(fmt.Sprintf("issr.handleRequest: rpcID=%s", rpcID), "rpcid", rpcID)

	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		go func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
			defer util.Recover("issr.handleRPCRequest")
			//logger.Infof(fmt.Sprintf("issr.handleRPCRequest recv request=%v", request), "rpcid", rpcID)
			method := request["method"].(string)
			data := request["data"].(map[string]interface{})

			var result map[string]interface{}
			err := &nprotoo.Error{Code: 400, Reason: fmt.Sprintf("Unkown method [%s]", method)}

			if method != "" {
				processingTime := monitor.NewProcessingTimeGauge(method)
				processingTime.Start()
				switch method {
				case proto.BizToIssrReportStreamState:
					result, err = report(data)
				default:
					logger.Warnf(fmt.Sprintf("issr.handleRPCRequest invalid protocol method=%s, data=%v", method, data), "rpcid", rpcID)
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
	"method", proto.BizToSsReportStreamState, "rid", rid, "uid", uid, "mid", mid, "sid", sid, "resolution", hd, "seconds", seconds
*/
// report 上报拉流计时数据
func report(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	// 判断参数
	if msg["appid"] == nil {
		return nil, &nprotoo.Error{Code: -1, Reason: "can't find appid"}
	}

	timestamp := time.Now().UnixNano() / 1000
	msg["timestamp"] = timestamp

	str, err := json.Marshal(msg)
	if err != nil {
		logger.Errorf(fmt.Sprintf("issr.report json marshal failed=%v", err))
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("json marshal err:%v", err)}
	}
	err = kafkaProducer.Produce("Livs-Usage-Event", string(str))
	if err != nil {
		logger.Errorf(fmt.Sprintf("issr.report kafka produce error=%v", err))
		err = redis.RPush(failureKey, string(str))
		if err != nil {
			logger.Errorf(fmt.Sprintf("issr.report store failure err=%v", err))
		}
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("kafka produce err:%v", err)}
	}
	logger.Infof(fmt.Sprintf("issr.report msg: %s", string(str)))
	return util.Map(), nil
}

func subStreamStart(msg map[string]interface{}) {
	//logger.Infof(fmt.Sprintf("issr.subStreamStart msg:%v", msg))
	rid, ok := msg["rid"].(string)
	if !ok {
		logger.Errorf(fmt.Sprint("issr.subStreamStart rid can't be empty"))
		return
	}
	uid, ok := msg["uid"].(string)
	if !ok {
		logger.Errorf(fmt.Sprint("issr.subStreamStart uid can't be empty"))
		return
	}
	minfo, ok := msg["minfo"].(map[string]interface{})
	if !ok {
		logger.Errorf(fmt.Sprint("issr.subStreamStart	minfo can't be found"))
		return
	}
	appid := util.InterfaceToString(minfo["appid"])
	if appid == "" {
		logger.Errorf(fmt.Sprint("issr.subStreamStart	minfo can't be empty"))
		return
	}

	resolution := util.InterfaceToString(minfo["resolution"])

	isVideo := util.InterfaceToBool(minfo["video"])
	isAudio := util.InterfaceToBool(minfo["audio"])
	isScreen := util.InterfaceToBool(minfo["screen"])
	isCanvas := util.InterfaceToBool(minfo["canvas"])

	var mode string
	if isVideo || isScreen || isCanvas {
		mode = "video"
	} else if isAudio {
		mode = "audio"
	}

	//get distributed lock for user subscription timing
	lockKey := proto.GetSubStreamTimingLockKey(appid, rid, uid)
	redis.Lock(lockKey)
	defer redis.Unlock(lockKey)
	//for now ,just calc video stream
	if mode == "video" && resolution != "" {
		key := proto.GetSubVideoStreamTimingKey(appid, rid, uid)
		info := redis.Get(key)
		if info == "" {
			val := util.Map("rid", rid, "uid", uid, "appid", appid, "mode", mode,
				"pixels", timing.GetPixelsByResolution(resolution), "count", 1,
				"time", time.Now().Unix())
			err := redis.Set(key, util.Marshal(val), redisKeyTTL)
			if err != nil {
				logger.Errorf(fmt.Sprintf("issr.subStreamStart first set fail v=%v", err), "rid", rid, "uid", uid)
				return
			}
		} else {
			data := util.Unmarshal(info)
			//logger.Infof(fmt.Sprintf("issr.subStreamStart data:%v", data), "rid", rid, "uid", uid)
			count := util.InterfaceToInt64(data["count"])
			count += 1
			lastPixels := util.InterfaceToInt64(data["pixels"])
			lastResolution := timing.TransformResolutionFromPixels(lastPixels)
			pixels := timing.GetPixelsByResolution(resolution)
			totalPixels := lastPixels + pixels
			currentResolution := timing.TransformResolutionFromPixels(totalPixels)
			start := util.InterfaceToInt64(data["time"])
			end := time.Now().Unix()
			isResChanged := currentResolution != lastResolution
			if isResChanged {
				interval := end - start
				if interval > 0 && lastResolution != "" {
					timestamp := time.Now().UnixNano() / 1000
					timingReport := util.Map("appid", appid, "rid", rid, "uid", uid, "mediatype", "video",
						"resolution", lastResolution, "timestamp", timestamp, "seconds", interval,
						"type", timingType)
					str := util.Marshal(timingReport)
					err := kafkaProducer.Produce("Livs-Usage-Event", string(str))
					logger.Infof(fmt.Sprintf("issr.subStreamStart report: %s", string(str)))
					if err != nil {
						logger.Errorf(fmt.Sprintf("issr.subStreamStart kafka produce error=%v", err))
						err = redis.RPush(failureKey, string(str))
						if err != nil {
							logger.Errorf(fmt.Sprintf("issr.subStreamStart store failure err=%v", err))
						}
					}
				}
			}
			var ts int64
			if isResChanged {
				ts = end
			} else {
				ts = start
			}
			val := util.Map("rid", rid, "uid", uid, "appid", appid, "pixels", totalPixels, "time", ts, "count", count)
			err := redis.Set(key, util.Marshal(val), redisKeyTTL)
			if err != nil {
				logger.Errorf(fmt.Sprintf("issr.subStreamStart set fail v=%v", err), "rid", rid, "uid", uid)
				return
			}
		}
	}
}

func subStreamEnd(msg map[string]interface{}) {
	//logger.Infof(fmt.Sprintf("issr.subStreamEnd msg:%v", msg))
	rid, ok := msg["rid"].(string)
	if !ok {
		logger.Errorf(fmt.Sprint("issr.subStreamEnd rid can't be empty"))
		return
	}
	uid, ok := msg["uid"].(string)
	if !ok {
		logger.Errorf(fmt.Sprint("issr.subStreamEnd uid can't be empty"))
		return
	}
	minfo, ok := msg["minfo"].(map[string]interface{})
	if !ok {
		logger.Errorf(fmt.Sprint("issr.subStreamEnd	minfo can't be found"))
		return
	}
	appid := util.InterfaceToString(minfo["appid"])
	if appid == "" {
		logger.Errorf(fmt.Sprint("issr.subStreamEnd	appid can't be empty"))
		return
	}

	resolution := util.InterfaceToString(minfo["resolution"])

	isVideo := util.InterfaceToBool(minfo["video"])
	isAudio := util.InterfaceToBool(minfo["audio"])
	isScreen := util.InterfaceToBool(minfo["screen"])
	isCanvas := util.InterfaceToBool(minfo["canvas"])

	var mode string
	if isVideo || isScreen || isCanvas {
		mode = "video"
	} else if isAudio {
		mode = "audio"
	}

	//get distributed lock for user subscription timing
	lockKey := proto.GetSubStreamTimingLockKey(appid, rid, uid)
	redis.Lock(lockKey)
	defer redis.Unlock(lockKey)
	if mode == "video" && resolution != "" {
		key := proto.GetSubVideoStreamTimingKey(appid, rid, uid)
		info := redis.Get(key)
		if info != "" {
			data := util.Unmarshal(info)
			//logger.Infof(fmt.Sprintf("issr.subStreamEnd data:%v", data), "rid", rid, "uid", uid)
			count := util.InterfaceToInt64(data["count"])
			count -= 1
			lastPixels := util.InterfaceToInt64(data["pixels"])
			lastResolution := timing.TransformResolutionFromPixels(lastPixels)
			pixels := timing.GetPixelsByResolution(resolution)
			totalPixels := lastPixels - pixels
			currentResolution := timing.TransformResolutionFromPixels(totalPixels)
			start := util.InterfaceToInt64(data["time"])
			end := time.Now().Unix()
			isResChanged := lastResolution != currentResolution
			if isResChanged {
				interval := end - start
				if interval > 0 && lastResolution != "" {
					timestamp := time.Now().UnixNano() / 1000
					timingReport := util.Map("appid", appid, "rid", rid, "uid", uid, "mediatype", "video",
						"resolution", lastResolution, "timestamp", timestamp, "seconds", interval,
						"type", timingType)
					str := util.Marshal(timingReport)
					err := kafkaProducer.Produce("Livs-Usage-Event", string(str))
					logger.Infof(fmt.Sprintf("issr.subStreamEnd report: %s", string(str)))
					if err != nil {
						logger.Errorf(fmt.Sprintf("issr.subStreamEnd kafka produce error=%v", err))
						failureKey := proto.GetFailedStreamStateKey()
						err = redis.RPush(failureKey, string(str))
						if err != nil {
							logger.Errorf(fmt.Sprintf("issr.subStreamEnd store failure err=%v", err))
						}
					}
				}
			}
			if count != 0 {
				var ts int64
				if isResChanged {
					ts = end
				} else {
					ts = start
				}
				val := util.Map("rid", rid, "uid", uid, "appid", appid, "pixels", totalPixels, "time", ts, "count", count)
				err := redis.Set(key, util.Marshal(val), redisKeyTTL)
				if err != nil {
					logger.Errorf(fmt.Sprintf("issr.subStreamEnd set fail v=%v", err), "rid", rid, "uid", uid)
					return
				}
			} else {
				err := redis.Del(key)
				if err != nil {
					logger.Errorf(fmt.Sprintf("issr.subStreamEnd del fail v=%v", err), "rid", rid, "uid", uid)
					return
				}
			}
		}
	}
}
