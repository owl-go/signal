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
	case proto.ClientToBizTrickleICE:
		trickle(peer, msg, accept, reject)
	case proto.ClientToBizBroadcast:
		broadcast(peer, msg, accept, reject)
	case proto.ClientToBizGetRoomUsers:
		listusers(peer, msg, accept, reject)
	case proto.ClientToBizStartLivestream:
		startlivestream(peer, msg, accept, reject)
	case proto.ClientToBizStopLivestream:
		stoplivestream(peer, msg, accept, reject)
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
    "info":{"name":"zhou","head":""}
  }
*/
// join 加入房间
func join(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.join uid=%s msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) {
		return
	}

	// 获取参数
	uid := peer.ID()
	rid := util.Val(msg, "rid")
	info := util.Val(msg, "info")

	//create stream timer and add the dummy audio stream,then start
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
		nid := resp["nid"].(string)
		if nid != node.NodeInfo().Nid {
			// 不在当前节点
			rpcBiz := rpcs[nid]
			if rpcBiz != nil {
				rpcBiz.SyncRequest(proto.BizToBizOnKick, util.Map("rid", rid, "uid", uid))
			}
		} else {
			// 在当前节点
			rpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
			rpc.SyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))
			// 获取老的peer数据
			room := GetRoom(rid)
			if room != nil {
				tmpPeer := room.room.GetPeer(uid)
				if tmpPeer != nil {
					tmpPeer.Notify(proto.BizToClientOnKick, util.Map("rid", rid, "uid", uid))
					tmpPeer.Close()
				}
			}
			DelPeer(rid, uid)
		}
	}

	// 重新加入房间
	AddPeer(rid, peer)
	// 通知房间其他人
	rpc.SyncRequest(proto.BizToIslbOnJoin, util.Map("rid", rid, "uid", uid, "nid", node.NodeInfo().Nid, "info", info))

	// 查询房间所有用户
	_, users := FindRoomUsers(uid, rid)
	result := util.Map("users", users)
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

	// 获取参数
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
	rpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
	rpc.SyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))
	DelPeer(rid, uid)

	/*
		for _, room := range GetRoomsByPeer(uid) {
			ridTmp := room.GetID()
			rpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", ridTmp, "uid", uid, "mid", ""))
			rpc.SyncRequest(proto.BizToIslbOnLeave, util.Map("rid", ridTmp, "uid", uid))
			DelPeer(ridTmp, uid)
		}*/
	//stop timer if didn't stop then report
	timer := peer.GetStreamTimer()
	if !timer.IsStopped() {
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

	// 获取参数
	uid := peer.ID()
	rid := util.Val(msg, "rid")
	info := util.Val(msg, "info")

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
	rpc.AsyncRequest(proto.BizToIslbKeepLive, util.Map("rid", rid, "uid", uid, "info", info))
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
      "minfo": {"codec": "h264", "video": true, "audio": true, "screen": false}
  }
*/
// publish 发布流
func publish(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.publish uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) || invalid(msg, "jsep", reject) {
		return
	}

	// 获取参数
	uid := peer.ID()
	rid := util.Val(msg, "rid")

	jsep := msg["jsep"].(map[string]interface{})
	if invalid(jsep, "sdp", reject) {
		return
	}

	room := GetRoom(rid)
	if room == nil {
		logger.Errorf("biz.publish room doesn't exist", "uid", uid, "rid", rid)
		reject(codeRIDErr, codeStr(codeRIDErr))
		return
	}

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

	minfo := msg["minfo"]
	if minfo == nil {
		logger.Errorf("biz.publish minfo not found", "uid", uid, "rid", rid)
		reject(codeMinfoErr, codeStr(codeMinfoErr))
		return
	}

	minfo = msg["minfo"].(map[string]interface{})
	// 获取sfu节点的resp
	resp, err := rpcSfu.SyncRequest(proto.BizToSfuPublish, util.Map("rid", rid, "uid", uid, "minfo", minfo, "jsep", jsep))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.publish request sfu err=%v", err.Reason), "uid", uid, "rid", rid)
		reject(err.Code, err.Reason)
		return
	}

	logger.Infof(fmt.Sprintf("biz.publish request sfu resp=%v", resp), "uid", uid, "rid", rid)

	rsp := make(map[string]interface{})
	rsp["jsep"] = resp["jsep"]
	rsp["mid"] = resp["mid"]

	nid := sfu.Nid
	mid := util.Val(rsp, "mid")
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

	// 获取参数
	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")

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
	"minfo": {"video": true, "audio": true}
  }
*/
// subscribe 订阅流
func subscribe(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.subscribe uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) || invalid(msg, "jsep", reject) {
		return
	}

	// 获取参数
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
	minfo = msg["minfo"].(map[string]interface{})
	rspSfu := make(map[string]interface{})
	resp, err := rpcSfu.SyncRequest(proto.BizToSfuSubscribe, util.Map("rid", rid, "uid", uid, "mid", mid, "jsep", jsep, "minfo", minfo))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.subscribe request sfu err=%v", err.Reason), "uid", uid, "rid", rid, "mid", mid)
		reject(err.Code, err.Reason)
		return
	}

	logger.Infof(fmt.Sprintf("biz.subscribe request sfu resp=%v", resp), "uid", uid, "rid", rid, "mid", mid)

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
  "request":true
  "id":3764139
  "method":"trickle"
  "data":{
      "rid": "room1",
      "nid":"shenzhen-sfu-1",
      "mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF"
      "sid": "$sid"
      "ice": "$icecandidate"
      "ispub": "true"
  }
*/
// trickle ice数据
func trickle(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	/*
		uid := peer.ID()
		rid := util.Val(msg, "rid")
		mid := util.Val(msg, "mid")
		sid := util.Val(msg, "sid")
		ice := util.Val(msg, "ice")
		ispub := util.Val(msg, "ispub")
		log.Infof("biz.trickle uid=%s msg=%v", uid, msg)

		var sfu *dis.Node
		nid := util.Val(msg, "nid")
		if nid != "" {
			sfu = FindSfuNodeByID(nid)
		} else {
			sfu = FindSfuNodeByMid(rid, mid)
		}
		if sfu == nil {
			log.Errorf("sfu node is not find")
			reject(codeSfuErr, codeStr(codeSfuErr))
			return
		}

		rpc := protoo.NewRequestor(dis.GetRPCChannel(*sfu))
		rpc.AsyncRequest(proto.BizToSfuTrickleICE, util.Map("rid", rid, "sid", sid, "mid", mid, "ice", ice, "ispub", ispub))
	*/
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

// 查询房间所有数据
func listusers(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.listusers uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	// 查询房间所有用户
	_, users := FindRoomUsers(uid, rid)

	result := util.Map("users", users)
	accept(result)
}

func startlivestream(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	logger.Infof(fmt.Sprintf("biz.startlivestream uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")

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

	var mcu *dis.Node
	mcu = FindMcuNodeByRid(rid)
	if mcu == nil {
		mcu = FindMcuNodeByPayload()
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

	// 获取sfu节点的resp
	sfuresp, err := rpcSfu.SyncRequest(proto.BizToSfuSubscribeRTP, util.Map("rid", rid, "uid", mcu.Nid, "mid", mid))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.startlivestream request sfu offer err=%v", err.Reason), "uid", uid, "rid", rid, "mid", mid)
		reject(err.Code, err.Reason)
		return
	}

	logger.Infof(fmt.Sprintf("biz.startlivestream request sfu offer resp=%v", sfuresp), "uid", uid, "rid", rid, "mid", mid)

	mcuresp, err := rpcMcu.SyncRequest(proto.BizToMcuPublishRTP, util.Map("rid", rid, "uid", sfu.Nid, "jsep", sfuresp["jsep"]))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.startlivestream request mcu answer err=%v", err.Reason), "uid", uid, "rid", rid, "mid", mid)
		reject(err.Code, err.Reason)
		return
	}

	logger.Infof(fmt.Sprintf("biz.startlivestream request mcu answer resp=%v", mcuresp), "uid", uid, "rid", rid, "mid", mid)

	resp, err := rpcSfu.SyncRequest(proto.BizToSfuSubscribeRTP, util.Map("sid", mid, "jsep", mcuresp["jsep"]))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.startlivestream request sfu answer err=%v", err.Reason), "uid", uid, "rid", rid, "mid", mid)
		reject(err.Code, err.Reason)
		return
	}

	logger.Infof(fmt.Sprintf("biz.startlivestream request sfu answer resp=%v", resp), "uid", uid, "rid", rid, "mid", mid)

	accept(emptyMap)
}

func stoplivestream(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {

	logger.Infof(fmt.Sprintf("biz.stoplivestream uid=%s,msg=%v", peer.ID(), msg), "uid", peer.ID())

	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")

	var sfu *dis.Node
	nid := util.Val(msg, "nid")
	if nid != "" {
		sfu = FindSfuNodeByID(nid)
	} else {
		sfu = FindSfuNodeByMid(rid, mid)
	}

	if sfu == nil {
		logger.Errorf("biz.stoplivestream sfu node not found", "uid", uid, "rid", rid, "sid", mid)
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}

	rpcSfu, find := rpcs[sfu.Nid]
	if !find {
		logger.Errorf("biz.stoplivestream sfu rpc not found", "uid", uid, "rid", rid, "sid", mid)
		reject(codeSfuRpcErr, codeStr(codeSfuRpcErr))
		return
	}

	//var mcu *dis.Node
	mcu := FindMcuNodeByRid(rid)
	if mcu == nil {
		reject(-1, fmt.Sprintf("can't find mcu node by rid:%s", rid))
		return
	}

	mid = mcu.Nid + "#" + mid
	rpcSfu.AsyncRequest(proto.BizToSfuUnSubscribe, util.Map("rid", rid, "uid", mcu.Nid, "mid", mid))

	// resp
	accept(emptyMap)
}
