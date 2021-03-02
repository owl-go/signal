package node

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	reg "mgkj/pkg/server"
	"mgkj/pkg/util"

	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/server"
	"github.com/cloudwebrtc/go-protoo/transport"
)

const (
	statCycle = time.Second * 5
)

var (
	wsServer *server.WebSocketServer
	rooms    = make(map[string]*RoomNode)
	roomLock sync.RWMutex
)

// InitWebSocket 初始化ws服务
func InitWebSocket(host string, port int, cert, key string) {
	wsServer = server.NewWebSocketServer(handleWebSocket)
	config := server.DefaultConfig()
	config.Host = host
	config.Port = port
	config.CertFile = cert
	config.KeyFile = key
	go wsServer.Bind(config)
	go checkRoom()
}

// checkRoom 检查所有的房间
func checkRoom() {
	t := time.NewTicker(statCycle)
	defer t.Stop()
	for range t.C {
		var nCount int = 0
		roomLock.Lock()
		for rid, node := range rooms {
			nCount += len(node.room.GetPeers())
			for uid := range node.room.GetPeers() {
				bLive := FindPeerIsLive(rid, uid)
				if !bLive {
					// 查询islb节点
					islb := FindIslbNode()
					if islb == nil {
						log.Errorf("islb node is not find")
					}
					rpc := protoo.NewRequestor(reg.GetRPCChannel(*islb))
					rpc.AsyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
					rpc.AsyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))
					node.room.RemovePeer(uid)
				}
			}
			if len(node.room.GetPeers()) == 0 {
				node.room.Close()
				delete(rooms, rid)
			}
		}
		roomLock.Unlock()
		// 更新负载
		node.UpdateNodePayload(nCount)
	}
}

func handleWebSocket(transport *transport.WebSocketTransport, request *http.Request) {
	//ws://127.0.0.1:8443/ws?peer=alice
	params := request.URL.Query()
	peers := params["peer"]
	if peers == nil || len(peers) < 1 {
		log.Errorf("biz handleWebSocket not find peer id")
		return
	}
	// 创建peer对象
	uid := peers[0]
	peerObj := peer.NewPeer(uid, transport)
	log.Infof("biz handleWebSocket peer = %s", uid)

	/*
		request
		{
		  request : true,
		  id      : 12345678,
		  method  : 'chatmessage',
		  data    :
		  {
		    type  : 'text',
		    value : 'Hi there!'
		  }
		}

		Success response
		{
			response : true,
			id       : 12345678,
			ok       : true,
			data     :
			{
				foo : 'lalala'
			}
		}

		Error response
		{
			response    : true,
			id          : 12345678,
			ok          : false,
			errorCode   : 123,
			errorReason : 'Something failed'
		}
	*/
	handleRequest := func(request peer.Request, accept peer.RespondFunc, reject peer.RejectFunc) {
		var data = make(map[string]interface{})
		err := json.Unmarshal(request.Data, &data)
		if err != nil {
			log.Errorf("biz handleRequest error = %s", err.Error())
		}

		method := request.Method
		switch method {
		case proto.ClientToBizJoin:
			join(peerObj, data, accept, reject)
		case proto.ClientToBizLeave:
			leave(peerObj, data, accept, reject)
		case proto.ClientToBizKeepLive:
			keeplive(peerObj, data, accept, reject)
		case proto.ClientToBizPublish:
			publish(peerObj, data, accept, reject)
		case proto.ClientToBizUnPublish:
			unpublish(peerObj, data, accept, reject)
		case proto.ClientToBizSubscribe:
			subscribe(peerObj, data, accept, reject)
		case proto.ClientToBizUnSubscribe:
			unsubscribe(peerObj, data, accept, reject)
		case proto.ClientToBizTrickleICE:
			trickle(peerObj, data, accept, reject)
		case proto.ClientToBizBroadcast:
			broadcast(peerObj, data, accept, reject)
		default:
			reject(codeUnknownErr, codeStr(codeUnknownErr))
		}
	}

	/*
		Notification
		{
			notification : true,
			method       : 'chatmessage',
			data         :
			{
				foo : 'bar'
			}
		}
	*/
	handleNotification := func(notification bool, method string, Data json.RawMessage) {
		log.Infof("handleNotification method = %s, data = %s", method, string(Data))
	}

	handleClose := func(code int, err string) {
		log.Infof("handleClose err = %d, %s", code, err)
	}

	_, _, _ = handleRequest, handleNotification, handleClose

	for {
		select {
		case resp := <-peerObj.OnNotification:
			handleNotification(resp.Notification, resp.Method, resp.Data)
		case resp := <-peerObj.OnRequest:
			handleRequest(resp.Request, resp.Accept, resp.Reject)
		case resp := <-peerObj.OnClose:
			handleClose(resp.Code, resp.Text)
		case resp := <-peerObj.OnError:
			handleClose(resp.Code, resp.Text)
		}
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
func join(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
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

	rpc := protoo.NewRequestor(reg.GetRPCChannel(*islb))
	// 删除以前加入过的房间数据
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
func leave(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
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
	for _, room := range GetRoomsByPeer(uid) {
		ridTmp := room.GetID()
		rpc := protoo.NewRequestor(reg.GetRPCChannel(*islb))
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
	"method":"keeplive"
	"data":{
		"rid":"room1",
		"info":$info
	}
*/
// keeplive 保活
func keeplive(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
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
		"nid":"shenzhen-sfu-1",
		"jsep": {"type": "offer","sdp": "..."},
	    "minfo": {"codec": "h264", "video": true, "audio": true, "screen": false}
	}
*/
// publish 发布流
func publish(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
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
	minfo := msg["minfo"].(map[string]interface{})
	var sfu *reg.Node
	nid := util.Val(msg, "nid")
	if nid != "" {
		sfu = FindSfuNodeByID(nid)
	} else {
		sfu = FindSfuNodeByPayload()
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

	// 获取sfu节点的resp
	bPublish := false
	rsp := make(map[string]interface{})

	rpc := protoo.NewRequestor(reg.GetRPCChannel(*sfu))
	resp, err := rpc.SyncRequest(proto.BizToSfuPublish, util.Map("rid", rid, "uid", uid, "minfo", minfo, "jsep", jsep))
	if err != nil {
		log.Errorf("rpc to sfu fail")
		reject(codePubErr, codeStr(codePubErr))
		return
	}
	// "method", proto.SfuToBizPublish, "errorCode", 0, "jsep", answer, "mid", mid, "tracks", tracks
	// "method", proto.SfuToBizPublish, "errorCode", 403, "errorReason", "publish: sdp parse failed"
	log.Infof("biz.publish respHandler resp=%v", resp)
	code := int(resp["errorCode"].(float64))
	if code == 0 {
		bPublish = true
		rsp["jsep"] = resp["jsep"]
		rsp["mid"] = resp["mid"]
		rsp["tracks"] = resp["tracks"]
	} else {
		bPublish = false
	}
	if !bPublish {
		log.Errorf("publish is not suc")
		reject(codePubErr, codeStr(codePubErr))
		return
	}

	nid = sfu.Nid
	mid := util.Val(rsp, "mid")
	tracks := rsp["tracks"]
	// 通知islb
	rpc = protoo.NewRequestor(reg.GetRPCChannel(*islb))
	rpc.AsyncRequest(proto.BizToIslbOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "tracks", tracks, "nid", nid))
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
func unpublish(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
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
	}
*/
// subscribe 订阅流
func subscribe(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
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

	find := false
	var rsp map[string]interface{}
	tracks := msg["tracks"].(map[string]interface{})
	if tracks == nil {
		rsp, find = FindMediaIndoByMid(rid, mid)
		if !find {
			log.Errorf("subscribe is not suc")
			reject(codeSubErr, codeStr(codeSubErr))
			return
		}
	} else {
		find = true
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

	// 订阅流回调
	rspSfu := make(map[string]interface{})

	rpc := protoo.NewRequestor(reg.GetRPCChannel(*sfu))
	if find {
		resp, err := rpc.SyncRequest(proto.BizToSfuSubscribe, util.Map("rid", rid, "uid", uid, "mid", mid, "tracks", tracks, "jsep", jsep))
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
	} else {
		resp, err := rpc.SyncRequest(proto.BizToSfuSubscribe, util.Map("rid", rid, "uid", uid, "mid", mid, "tracks", rsp["tracks"], "jsep", jsep))
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
func unsubscribe(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
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
func trickle(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

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
func broadcast(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	data := util.Val(msg, "data")

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	rpc := protoo.NewRequestor(reg.GetRPCChannel(*islb))
	rpc.AsyncRequest(proto.BizToIslbBroadcast, util.Map("rid", rid, "uid", uid, "data", data))
	// resp
	accept(emptyMap)
}
