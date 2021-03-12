package biz

import (
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	reg "mgkj/pkg/server"
	"mgkj/pkg/util"
	"mgkj/pkg/ws"
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
	log.Infof("biz.join", "msg => %v", msg)
	if invalid(msg, "rid", reject) {
		return
	}

	// 获取参数
	uid := peer.ID()
	rid := util.Val(msg, "rid")
	info := util.Val(msg, "info")
	log.Infof("biz.join uid=%s msg=%v", uid, msg)

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	// 删除以前加入过的房间数据
	rpc := protoo.NewRequestor(reg.GetRPCChannel(*islb))
	for _, room := range GetRoomsByPeer(uid) {
		ridTmp := room.GetID()
		rpc.AsyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", ridTmp, "uid", uid, "mid", ""))
		rpc.AsyncRequest(proto.BizToIslbOnLeave, util.Map("rid", ridTmp, "uid", uid))
		DelPeer(ridTmp, uid)
	}

	// 重新加入房间
	AddPeer(rid, peer)
	// 通知房间其他人
	rpc.AsyncRequest(proto.BizToIslbOnJoin, util.Map("rid", rid, "uid", uid, "info", info))
	// 查询房间存在的发布流
	FindMediaPubs(peer, rid)
	// resp
	accept(emptyMap)
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
	if invalid(msg, "rid", reject) {
		return
	}

	// 获取参数
	uid := peer.ID()
	//rid := util.Val(msg, "rid")
	log.Infof("biz.leave uid=%s msg=%v", uid, msg)

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	// 删除加入的房间和流
	rpc := protoo.NewRequestor(reg.GetRPCChannel(*islb))
	for _, room := range GetRoomsByPeer(uid) {
		ridTmp := room.GetID()
		rpc.AsyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", ridTmp, "uid", uid, "mid", ""))
		rpc.AsyncRequest(proto.BizToIslbOnLeave, util.Map("rid", ridTmp, "uid", uid))
		DelPeer(ridTmp, uid)
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

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	// 通知islb
	rpc := protoo.NewRequestor(reg.GetRPCChannel(*islb))
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
	if invalid(msg, "rid", reject) || invalid(msg, "jsep", reject) {
		return
	}

	// 获取参数
	uid := peer.ID()
	rid := util.Val(msg, "rid")
	log.Infof("biz.publish uid=%s msg=%v", uid, msg)

	jsep := msg["jsep"].(map[string]interface{})
	if invalid(jsep, "sdp", reject) {
		return
	}

	room := GetRoom(rid)
	if room == nil {
		reject(codeRIDErr, codeStr(codeRIDErr))
		return
	}

	sfu := FindSfuNodeByPayload()
	if sfu == nil {
		log.Errorf("sfu node is not find")
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	// 获取sfu节点的resp
	minfo := msg["minfo"]
	if minfo == nil {
		log.Errorf("minfo node is not find")
		reject(codePubErr, codeStr(codePubErr))
		return
	}

	minfo = msg["minfo"].(map[string]interface{})
	rpc := protoo.NewRequestor(reg.GetRPCChannel(*sfu))
	resp, err := rpc.SyncRequest(proto.BizToSfuPublish, util.Map("rid", rid, "uid", uid, "minfo", minfo, "jsep", jsep))
	if err != nil {
		log.Errorf("rpc to sfu fail")
		reject(codePubErr, codeStr(codePubErr))
		return
	}
	// "method", proto.SfuToBizPublish, "errorCode", 0, "jsep", answer, "mid", mid
	// "method", proto.SfuToBizPublish, "errorCode", 403, "errorReason", "publish: sdp parse failed"
	log.Infof("biz.publish respHandler resp=%v", resp)

	bPublish := false
	rsp := make(map[string]interface{})
	code := int(resp["errorCode"].(float64))
	if code == 0 {
		bPublish = true
		rsp["jsep"] = resp["jsep"]
		rsp["mid"] = resp["mid"]
	} else {
		bPublish = false
	}
	if !bPublish {
		log.Errorf("publish is not suc")
		reject(codePubErr, codeStr(codePubErr))
		return
	}

	nid := sfu.Nid
	mid := util.Val(rsp, "mid")
	// 通知islb
	rpc = protoo.NewRequestor(reg.GetRPCChannel(*islb))
	rpc.AsyncRequest(proto.BizToIslbOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "nid", nid, "minfo", minfo))
	// resp
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
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	// 获取参数
	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	log.Infof("biz.unpublish uid=%s msg=%v", uid, msg)

	var sfu *reg.Node
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

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	rpc := protoo.NewRequestor(reg.GetRPCChannel(*sfu))
	rpc.AsyncRequest(proto.BizToSfuUnPublish, util.Map("rid", rid, "uid", uid, "mid", mid))
	rpc = protoo.NewRequestor(reg.GetRPCChannel(*islb))
	rpc.AsyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
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
	log.Infof("biz.subscribe msg=%v", msg)

	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) || invalid(msg, "jsep", reject) {
		return
	}

	// 获取参数
	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	log.Infof("biz.subscribe uid=%s", uid)

	jsep := msg["jsep"].(map[string]interface{})
	if invalid(jsep, "sdp", reject) {
		return
	}

	var sfu *reg.Node
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

	minfo := msg["minfo"]
	if minfo == nil {
		log.Errorf("minfo node is not find")
		reject(codePubErr, codeStr(codePubErr))
		return
	}

	// 获取sfu节点的resp
	find := false
	minfo = msg["minfo"].(map[string]interface{})
	rspSfu := make(map[string]interface{})
	rpc := protoo.NewRequestor(reg.GetRPCChannel(*sfu))
	resp, err := rpc.SyncRequest(proto.BizToSfuSubscribe, util.Map("rid", rid, "uid", uid, "mid", mid, "jsep", jsep, "minfo", minfo))
	//resp, err := rpc.SyncRequest(proto.BizToSfuSubscribe, util.Map("rid", rid, "uid", uid, "mid", mid, "jsep", jsep))
	if err != nil {
		log.Errorf("rpc to sfu fail")
		reject(codeSubErr, codeStr(codeSubErr))
		return
	}
	code := int(resp["errorCode"].(float64))
	if code == 0 {
		find = true
		rspSfu["jsep"] = resp["jsep"]
		rspSfu["sid"] = resp["mid"]
	} else {
		find = false
	}

	if !find {
		log.Errorf("subscribe is not suc")
		reject(codeSubErr, codeStr(codeSubErr))
		return
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
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	log.Infof("biz.unsubscribe uid=%s msg=%v", uid, msg)

	var sfu *reg.Node
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
	rpc := protoo.NewRequestor(reg.GetRPCChannel(*sfu))
	rpc.AsyncRequest(proto.BizToSfuUnSubscribe, util.Map("rid", rid, "uid", uid, "mid", mid))
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

		var sfu *reg.Node
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

		rpc := protoo.NewRequestor(reg.GetRPCChannel(*sfu))
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
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	rpc := protoo.NewRequestor(reg.GetRPCChannel(*islb))
	rpc.AsyncRequest(proto.BizToIslbBroadcast, util.Map("rid", rid, "uid", uid, "data", msg["data"]))
	// resp
	accept(emptyMap)
}
