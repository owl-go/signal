package biz

import (
	"fmt"
	dis "signal/infra/discovery"
	"signal/pkg/proto"
	"signal/pkg/timing"
	"signal/pkg/ws"
	"signal/util"
)

// Entry 信令处理
func Entry(method string, peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	switch method {
	case proto.ClientToBizJoin:
		join(peer, msg, accept, reject)
	case proto.ClientToBizLeave:
		leave(peer, msg, accept, reject)
	case proto.ClientToBizKeepAlive:
		keepalive(peer, msg, accept, reject)
	case proto.ClientToBizPublish:
		publish(peer, msg, accept, reject)
	case proto.ClientToBizUnPublish:
		unpublish(peer, msg, accept, reject)
	case proto.ClientToBizSubscribe:
		subscribe(peer, msg, accept, reject)
	case proto.ClientToBizUnSubscribe:
		unsubscribe(peer, msg, accept, reject)
	case proto.ClientToBizStartLivestream:
		startlivestream(peer, msg, accept, reject)
	case proto.ClientToBizStopLivestream:
		stoplivestream(peer, msg, accept, reject)
	case proto.ClientToBizBroadcast:
		broadcast(peer, msg, accept, reject)
	case proto.ClientToBizGetRoomUsers:
		listusers(peer, msg, accept, reject)
	case proto.ClientToBizGetRoomLives:
		listlives(peer, msg, accept, reject)
	default:
		ws.DefaultReject(codeUnknownErr, codeStr(codeUnknownErr))
	}
}

/*
  "request":true
  "id":3764139
  "method":"join"
  "data":{
    "rid":"room1",
    "info":$info
  }
*/
// 用户加入房间
func join(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.join uid=%s msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	info := util.Val(msg, "info")

	// create stream timer and add the dummy audio stream,then start
	timer := timing.NewStreamTimer(rid, uid, peer.GetAppID())
	peer.SetStreamTimer(timer)
	dummyAudioStream := timing.NewStreamInfo("dummy-audio", "dummy-audio", "audio", "")
	timer.AddStream(dummyAudioStream)
	timer.Start()

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		logger.Errorf("biz.join islb node not found", "uid", uid, "rid", rid)
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	rpc, find := rpcs[islb.Nid]
	if !find {
		logger.Errorf("biz.join islb rpc not found", "uid", uid, "rid", rid)
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}
	// 查询uid是否在房间中
	resp, err := rpc.SyncRequest(proto.BizToIslbGetBizInfo, util.Map("rid", rid, "uid", uid))
	if err == nil {
		// uid已经存在，先删除
		biz := resp["nid"].(string)
		if biz != node.NodeInfo().Nid {
			// 不在当前节点
			rpcBiz := rpcs[biz]
			if rpcBiz != nil {
				rpcBiz.SyncRequest(proto.BizToBizOnKick, util.Map("rid", rid, "uid", uid))
			}
		} else {
			// 在当前节点
			rpc.SyncRequest(proto.BizToIslbOnLiveRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
			rpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
			rpc.SyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))
			// 删除老的peer数据
			oldpeer := GetPeer(rid, uid)
			if oldpeer != nil {
				oldpeer.Notify(proto.BizToClientOnKick, util.Map("rid", rid, "uid", uid))
				oldpeer.Close()
			}
			DelPeer(rid, uid)
		}
	}

	// 重新加入房间
	AddPeer(rid, peer)
	// 通知房间其他人
	rpc.SyncRequest(proto.BizToIslbOnJoin, util.Map("rid", rid, "uid", uid, "nid", node.NodeInfo().Nid, "info", info))

	// 查询房间其他所有用户
	_, users := FindRoomUsers(uid, rid)
	_, lives := FindRoomLives(uid, rid)
	result := util.Map("users", users, "lives", lives)
	// resp
	accept(result)
}

/*
  "request":true
  "id":3764139
  "method":"leave"
  "data":{
      "rid":"room1"
  }
*/
// leave 离开房间
func leave(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.leave uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		logger.Errorf("biz.leave islb node not found", "uid", uid)
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	rpc, find := rpcs[islb.Nid]
	if !find {
		logger.Errorf("biz.leave islb rpc not found", "uid", uid)
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}
	// 删除加入的房间和流
	rpc.SyncRequest(proto.BizToIslbOnLiveRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
	rpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
	rpc.SyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))
	DelPeer(rid, uid)

	//stop timer if didn't stop then report
	timer := peer.GetStreamTimer()
	if timer != nil && !timer.IsStopped() {
		timer.Stop()
		isVideo := timer.GetCurrentMode() == "video"
		reportStreamTiming(timer, isVideo, false)
	}
	// resp
	accept(emptyMap)
}

/*
  "request":true
  "id":3764139
  "method":"keepalive"
  "data":{
    "rid":"room1",
    "info":$info
  }
*/
// keepalive 保活
func keepalive(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	info := util.Val(msg, "info")

	// 判断是否在房间里面
	room := GetRoom(rid)
	if room == nil {
		logger.Errorf("biz.keepalive room doesn't exist", "uid", uid, "rid", rid)
		reject(codeRIDErr, codeStr(codeRIDErr))
		return
	}

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		logger.Errorf("biz.keepalive islb node found", "uid", uid, "rid", rid)
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	rpc, find := rpcs[islb.Nid]
	if !find {
		logger.Errorf("biz.keepalive islb rpc not found", "uid", uid, "rid", rid)
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}
	// 通知islb
	rpc.AsyncRequest(proto.BizToIslbKeepAlive, util.Map("rid", rid, "uid", uid, "info", info))
	// resp
	accept(emptyMap)
}

/*
  "request":true
  "id":3764139
  "method":"publish"
  "data":{
      "rid":"room1",
      "jsep": {"type": "offer","sdp": "..."},
      "minfo": {
	  	"video": true,
		"audio": true,
		"screen": false
		"resolution": "480p"//目前支持分辨率配置，240/360/480p/720p/1080p
	  }
  }
*/
// publish 发布流
func publish(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.publish uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) || invalid(msg, "jsep", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	jsep := msg["jsep"].(map[string]interface{})
	if invalid(jsep, "sdp", reject) {
		return
	}
	minfo := msg["minfo"]
	if minfo == nil {
		logger.Errorf("biz.publish minfo not found", "uid", uid, "rid", rid)
		reject(codeMinfoErr, codeStr(codeMinfoErr))
		return
	}
	minfo = msg["minfo"].(map[string]interface{})

	// 判断是否在房间里面
	room := GetRoom(rid)
	if room == nil {
		logger.Errorf("biz.publish room doesn't exist", "uid", uid, "rid", rid)
		reject(codeRIDErr, codeStr(codeRIDErr))
		return
	}

	// 查询sfu节点
	sfu := FindSfuNodeByPayload()
	if sfu == nil {
		logger.Errorf("biz.publish sfu node not found", "uid", uid, "rid", rid)
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}
	rpcSfu, find := rpcs[sfu.Nid]
	if !find {
		logger.Errorf("biz.publish sfu rpc not found", "uid", uid, "rid", rid)
		reject(codeSfuRpcErr, codeStr(codeSfuRpcErr))
		return
	}
	// 获取sfu节点的resp
	resp, err := rpcSfu.SyncRequest(proto.BizToSfuPublish, util.Map("rid", rid, "uid", uid, "minfo", minfo, "jsep", jsep))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.publish request sfu err=%v", err.Reason), "uid", uid, "rid", rid)
		reject(err.Code, err.Reason)
		return
	}

	logger.Infof(fmt.Sprintf("biz.publish request sfu resp=%v", resp), "uid", uid, "rid", rid)

	nid := sfu.Nid
	mid := util.Val(resp, "mid")
	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		logger.Errorf("biz.publish islb node not found", "uid", uid, "rid", rid, "mid", mid)
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	rpcIslb, find := rpcs[islb.Nid]
	if !find {
		logger.Errorf("biz.publish islb rpc not found", "uid", uid, "rid", rid, "mid", mid)
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}
	// 通知islb
	rpcIslb.SyncRequest(proto.BizToIslbOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "nid", nid, "minfo", minfo))
	// resp
	rsp := make(map[string]interface{})
	rsp["jsep"] = resp["jsep"]
	rsp["mid"] = mid
	rsp["nid"] = nid
	rsp["minfo"] = minfo
	accept(rsp)
}

/*
  "request":true
  "id":3764139
  "method":"unpublish"
  "data":{
      "rid": "room1",
      "nid":"shenzhen-sfu-1",
      "mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF"
  }
*/
// unpublish 取消发布流
func unpublish(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.unpublish uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")

	// 查询sfu节点
	var sfu *dis.Node
	nid := util.Val(msg, "nid")
	if nid != "" {
		sfu = FindSfuNodeByID(nid)
	} else {
		sfu = FindSfuNodeByMid(rid, mid)
	}
	if sfu == nil {
		logger.Errorf("biz.unpublish sfu node not found", "uid", uid, "rid", rid, "mid", mid)
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}
	rpcSfu, find := rpcs[sfu.Nid]
	if !find {
		logger.Errorf("biz.unpublish sfu rpc not found", "uid", uid, "rid", rid, "mid", mid)
		reject(codeSfuRpcErr, codeStr(codeSfuRpcErr))
		return
	}
	rpcSfu.SyncRequest(proto.BizToSfuUnPublish, util.Map("rid", rid, "uid", uid, "mid", mid))

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		logger.Errorf("biz.unpublish islb node not found", "uid", uid, "rid", rid, "mid", mid)
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	rpcIslb, find := rpcs[islb.Nid]
	if !find {
		logger.Errorf("biz.unpublish islb rpc not found", "uid", uid, "rid", rid, "mid", mid)
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}
	rpcIslb.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
	// resp
	accept(emptyMap)
}

/*
  "request":true
  "id":3764139
  "method":"subscribe"
  "data":{
    "rid":"room1",
    "mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF"
    "nid":"shenzhen-sfu-1",
    "jsep": {"type": "offer","sdp": "..."},
	"minfo": {
		"video": true,
		"audio": true,
		"resolution": "480p"
	}
  }
*/
// subscribe 订阅流
func subscribe(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.subscribe uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) || invalid(msg, "jsep", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	jsep := msg["jsep"].(map[string]interface{})
	if invalid(jsep, "sdp", reject) {
		return
	}
	minfo := msg["minfo"]
	if minfo == nil {
		logger.Errorf("biz.subscribe minfo not found", "uid", uid, "rid", rid, "mid", mid)
		reject(codeMinfoErr, codeStr(codeMinfoErr))
		return
	}
	minfo = msg["minfo"].(map[string]interface{})

	// 判断是否在房间里面
	room := GetRoom(rid)
	if room == nil {
		logger.Errorf("biz.subscribe room doesn't exist", "uid", uid, "rid", rid)
		reject(codeRIDErr, codeStr(codeRIDErr))
		return
	}

	// 获取sfu节点信息
	var sfu *dis.Node
	nid := util.Val(msg, "nid")
	if nid != "" {
		sfu = FindSfuNodeByID(nid)
	} else {
		sfu = FindSfuNodeByMid(rid, mid)
	}
	if sfu == nil {
		logger.Errorf("biz.subscribe sfu not found", "uid", uid, "rid", rid, "mid", mid)
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}
	rpcSfu, find := rpcs[sfu.Nid]
	if !find {
		logger.Errorf("biz.subscribe sfu rpc not found", "uid", uid, "rid", rid, "mid", mid)
		reject(codeSfuRpcErr, codeStr(codeSfuRpcErr))
		return
	}
	// 获取sfu节点的resp
	resp, err := rpcSfu.SyncRequest(proto.BizToSfuSubscribe, util.Map("rid", rid, "uid", uid, "mid", mid, "jsep", jsep, "minfo", minfo))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.subscribe request sfu err=%v", err.Reason), "uid", uid, "rid", rid, "mid", mid)
		reject(err.Code, err.Reason)
		return
	}

	logger.Infof(fmt.Sprintf("biz.subscribe request sfu resp=%v", resp), "uid", uid, "rid", rid, "mid", mid)

	rspSfu := make(map[string]interface{})
	rspSfu["jsep"] = resp["jsep"]
	rspSfu["sid"] = resp["mid"]
	rspSfu["uid"] = resp["uid"]

	//add stream to timer then start
	var mediatype string
	sid := rspSfu["sid"].(string)
	isVideo := minfo.(map[string]interface{})["video"].(bool)
	if !isVideo {
		isVideo = minfo.(map[string]interface{})["screen"].(bool)
	}
	isAudio := minfo.(map[string]interface{})["audio"].(bool)
	if !isVideo && isAudio {
		mediatype = "audio"
	} else if isVideo {
		mediatype = "video"
	}
	resolution := minfo.(map[string]interface{})["resolution"].(string)
	timer := peer.GetStreamTimer()
	if timer != nil {
		isModeChanged := timer.AddStream(timing.NewStreamInfo(mid, sid, mediatype, resolution))
		if isModeChanged {
			//this must be audio report
			timer.UpdateResolution()
			timer.Stop()
			reportStreamTiming(timer, false, false)
			timer.Renew()
		} else {
			if mediatype == "video" {
				isResolutionChanged := timer.UpdateResolution()
				if isResolutionChanged {
					timer.Stop()
					//report this interval
					reportStreamTiming(timer, true, true)
					//then renew timer
					timer.Renew()
				}
			}
		}
	}
	// resp
	accept(rspSfu)
}

/*
  "request":true
  "id":3764139
  "method":"unsubscribe"
  "data":{
    "rid": "room1",
    "nid":"shenzhen-sfu-1",
    "mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF" (sid)
  }
*/
// unsubscribe 取消订阅流
func unsubscribe(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.unsubscribe uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")

	// 获取sfu节点
	var sfu *dis.Node
	nid := util.Val(msg, "nid")
	if nid != "" {
		sfu = FindSfuNodeByID(nid)
	}
	if sfu == nil {
		logger.Errorf("biz.unsubscribe sfu node not found", "uid", uid, "rid", rid, "sid", mid)
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}
	rpcSfu, find := rpcs[sfu.Nid]
	if !find {
		logger.Errorf("biz.unsubscribe sfu rpc not found", "uid", uid, "rid", rid, "sid", mid)
		reject(codeSfuRpcErr, codeStr(codeSfuRpcErr))
		return
	}
	rpcSfu.SyncRequest(proto.BizToSfuUnSubscribe, util.Map("rid", rid, "uid", uid, "mid", mid))

	//remove stream from timer then according to the state,decide what to do
	timer := peer.GetStreamTimer()
	if timer != nil {
		removed, isModeChanged := timer.RemoveStreamBySID(mid)
		if removed != nil {
			if isModeChanged {
				//this must be video to audio
				timer.Stop()
				reportStreamTiming(timer, true, true)
				timer.Renew()
			} else {
				isResolutionChanged := timer.UpdateResolution()
				//check whether total resolution change or not,to determine timer stop or not
				if isResolutionChanged {
					timer.Stop()
					//report this interval
					reportStreamTiming(timer, true, true)
					//then timer renew
					timer.Renew()
				}
			}
		}
	}
	// resp
	accept(emptyMap)
}

/*
	{
		"request":true,
		"id":3764139,
		"method":"startlivestream",
		"data": {
			"rid": "room1",
			"mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF", // 媒体流id
			"nid": "shenzhen-sfu-1" 	// 媒体流所在sfu节点id
			"record":1,					// 0,不启用录制，1,启用录制
			"index":1,					// 1,主播，0,连麦者
		}
	}
*/
// 启动直播
func startlivestream(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.startlivestream uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	record := util.InterfaceToInt(msg["record"])
	index := util.InterfaceToInt(msg["index"])

	// 查找sfu节点
	var sfu *dis.Node
	nid := util.Val(msg, "nid")
	if nid != "" {
		sfu = FindSfuNodeByID(nid)
	} else {
		sfu = FindSfuNodeByMid(rid, mid)
	}
	if sfu == nil {
		logger.Errorf("biz.startlivestream sfu not found", "uid", uid, "rid", rid, "mid", mid)
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}
	rpcSfu, find := rpcs[sfu.Nid]
	if !find {
		logger.Errorf("biz.startlivestream sfu rpc not found", "uid", uid, "rid", rid, "mid", mid)
		reject(codeSfuRpcErr, codeStr(codeSfuRpcErr))
		return
	}

	// 查找mcu节点
	var mcu *dis.Node
	mcu = FindMcuNodeByRid(rid)
	if mcu == nil {
		mcu = FindMcuNodeByPayload()
		if mcu == nil {
			reject(-1, "mcu finding fail")
			return
		}
		mcu = SetMcuNodeByRid(rid, mcu.Nid)
		if mcu == nil {
			reject(-1, "mcu binding fail")
			return
		}
	}
	rpcMcu, find := rpcs[mcu.Nid]
	if !find {
		logger.Errorf("biz.startlivestream mcu rpc not found", "uid", uid, "rid", rid, "mid", mid)
		reject(codeMcuRpcErr, codeStr(codeMcuRpcErr))
		return
	}

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		logger.Errorf("biz.startlivestream islb node not found", "uid", uid, "rid", rid)
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	rpcIslb, find := rpcs[islb.Nid]
	if !find {
		logger.Errorf("biz.startlivestream islb rpc not found", "uid", uid, "rid", rid)
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}

	// 获取该流minfo
	islbresp, err := rpcIslb.SyncRequest(proto.BizToIslbGetMediaInfo, util.Map("rid", rid, "uid", uid, "mid", mid))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.startlivestream request islb err =%v", err.Reason), "uid", uid, "rid", rid, "mid", mid)
		reject(err.Code, err.Reason)
		return
	}
	minfo := islbresp["minfo"].(map[string]interface{})
	minfo["index"] = index

	logger.Infof(fmt.Sprintf("biz.startlivestream request islb resp=%v", islbresp), "uid", uid, "rid", rid, "mid", mid)

	// 获取sfu节点的resp
	sfuresp, err := rpcSfu.SyncRequest(proto.BizToSfuSubscribeRTP, util.Map("rid", rid, "uid", mcu.Nid, "mid", mid))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.startlivestream request sfu offer err=%v", err.Reason), "uid", uid, "rid", rid, "mid", mid)
		reject(err.Code, err.Reason)
		return
	}

	logger.Infof(fmt.Sprintf("biz.startlivestream request sfu offer resp=%v", sfuresp), "uid", uid, "rid", rid, "mid", mid)

	// 获取mcu节点的resp
	mcuresp, err := rpcMcu.SyncRequest(proto.BizToMcuPublishRTP, util.Map("appid", peer.GetAppID(), "rid", rid, "record", record, "uid", sfu.Nid, "jsep", sfuresp["jsep"], "minfo", minfo))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.startlivestream request mcu answer err=%v", err.Reason), "uid", uid, "rid", rid, "mid", mid)
		reject(err.Code, err.Reason)
		return
	}

	logger.Infof(fmt.Sprintf("biz.startlivestream request mcu answer resp=%v", mcuresp), "uid", uid, "rid", rid, "mid", mid)

	// 再次获取sfu节点resp
	resp, err := rpcSfu.SyncRequest(proto.BizToSfuSubscribeRTP, util.Map("mid", sfuresp["mid"], "rid", rid, "jsep", mcuresp["jsep"]))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.startlivestream request sfu answer err=%v", err.Reason), "uid", uid, "rid", rid, "mid", mid)
		reject(err.Code, err.Reason)
		return
	}

	logger.Infof(fmt.Sprintf("biz.startlivestream request sfu answer resp=%v", resp), "uid", uid, "rid", rid, "mid", mid)

	// 发送给islb保存
	_, err = rpcIslb.SyncRequest(proto.BizToIslbOnLiveAdd, util.Map("rid", rid, "uid", uid, "mid", mcuresp["mid"], "nid", mcu.Nid, "minfo", minfo))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.startlivestream request islb for liveStreamAdd err=%v", err.Reason), "uid", uid, "rid", rid)
		reject(err.Code, err.Reason)
		return
	}
	//start live streaming timer
	if record == 1 && peer.GetLiveStreamTimer() == nil {
		livestreamtimer := timing.NewLiveStreamTimer(rid, uid, peer.GetAppID(), "FHD")
		peer.SetLiveStreamTimer(livestreamtimer)
		livestreamtimer.Start()
	}
	// resp
	accept(util.Map("mcu", mcu.Nid, "mid", mcuresp["mid"]))
}

/*
	{
		"request":true,
		"id":3764139,
		"method":"stoplivestream",
		"data": {
			"rid": "room1",
			"mid": "sfu1#xxxxxx",		// startlivestream方法返回的mid
			"nid": "shenzhen-sfu-1"		// 媒体流所在sfu节点ID
			"mcu": "shenzhen-mcu-1", 	// 启动直播时，返回的mcu字段
		}
	}
*/
// 停止直播
func stoplivestream(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.stoplivestream uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")

	// 查询sfu节点
	nid := util.Val(msg, "nid")
	if nid == "" {
		reject(-1, "sfu nid can't be empty")
		return
	} else {
		sfu := FindSfuNodeByID(nid)
		if sfu == nil {
			reject(-1, fmt.Sprintf("can't find sfu node by nid:%s", nid))
			return
		}
	}

	// 查询mcu节点
	var mcu *dis.Node
	mcuid := util.Val(msg, "mcu")
	if mcuid != "" {
		mcu = FindMcuNodeByID(mcuid)
	} else {
		mcu = FindMcuNodeByRid(rid)
	}
	if mcu == nil {
		logger.Errorf("biz.stoplivestream mcu node not found", "uid", uid, "rid", rid, "sid", mid)
		reject(-1, fmt.Sprintf("can't find mcu node by nid :%s or rid:%s", mcuid, rid))
		return
	}
	rpcMcu, find := rpcs[mcu.Nid]
	if !find {
		logger.Errorf("biz.stoplivestream mcu rpc not found", "uid", uid, "rid", rid, "sid", mid)
		reject(codeMcuRpcErr, codeStr(codeMcuRpcErr))
		return
	}
	rpcMcu.AsyncRequest(proto.BizToMcuUnpublish, util.Map("rid", rid, "uid", nid, "mid", mid))

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		logger.Errorf("biz.stoplivestream islb node not found", "uid", uid, "rid", rid)
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	rpcIslb, find := rpcs[islb.Nid]
	if !find {
		logger.Errorf("biz.stoplivestream islb rpc not found", "uid", uid, "rid", rid)
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}
	_, err := rpcIslb.SyncRequest(proto.BizToIslbOnLiveRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.stoplivestream request islb for liveStreamRemove err=%v", err.Reason), "uid", uid, "rid", rid)
		reject(err.Code, err.Reason)
		return
	}
	// stop live streaming timer
	if peer.GetLiveStreamTimer() != nil {
		livestreamtimer := peer.GetLiveStreamTimer()
		if !livestreamtimer.IsStopped() {
			livestreamtimer.Stop()
			reportLiveStreamTiming(livestreamtimer) //sync operation
			peer.SetStreamTimer(nil)                //del live streaming timer
		}
	}
	// resp
	accept(emptyMap)
}

/*
	"request":true
	"id":3764139
	"method":"broadcast"
	"data":{
		"rid": "room1",
		"data": "$date"
	}
*/
// broadcast 客户端发送广播给对方
func broadcast(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.broadcast uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		logger.Errorf("biz.broadcast islb node not found", "uid", uid, "rid", rid)
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	rpcIslb, find := rpcs[islb.Nid]
	if !find {
		logger.Errorf("biz.broadcast islb rpc not found", "uid", uid, "rid", rid)
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}
	rpcIslb.AsyncRequest(proto.BizToIslbBroadcast, util.Map("rid", rid, "uid", uid, "data", msg["data"]))
	// resp
	accept(emptyMap)
}

// 获取房间其他用户实时流
func listusers(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.listusers uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	// 查询房间其他用户实时流
	_, users := FindRoomUsers(uid, rid)
	result := util.Map("users", users)
	accept(result)
}

// 获取房间其他用户直播流
func listlives(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.listlives uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	// 查询房间其他用户直播流
	_, lives := FindRoomLives(uid, rid)
	result := util.Map("lives", lives)
	accept(result)
}
