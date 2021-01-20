package node

import (
	"encoding/json"
	"net/http"
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
	statCycle = time.Second * 3
)

var (
	wsServer *server.WebSocketServer
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
		//info := "biz.checkRoom start\n"
		var nCount int = 0
		roomLock.Lock()
		for rid, node := range rooms {
			nCount += len(node.room.GetPeers())
			//info += fmt.Sprintf("room: %s peers: %d\n", rid, len(node.room.GetPeers()))
			if len(node.room.GetPeers()) == 0 {
				node.room.Close()
				delete(rooms, rid)
			}
		}
		roomLock.Unlock()
		//log.Infof(info)
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
	// 处理客户端的请求
	handleRequest := func(request peer.Request, accept peer.RespondFunc, reject peer.RejectFunc) {
		var data = make(map[string]interface{})
		if err := json.Unmarshal(request.Data, &data); err != nil {
			log.Errorf("biz handleRequest error = %s", err.Error())
		}

		method := request.Method
		switch method {
		case proto.ClientJoin:
			join(peerObj, data, accept, reject)
		case proto.ClientLeave:
			leave(peerObj, data, accept, reject)
		case proto.ClientPublish:
			publish(peerObj, data, accept, reject)
		case proto.ClientUnPublish:
			unpublish(peerObj, data, accept, reject)
		case proto.ClientSubscribe:
			subscribe(peerObj, data, accept, reject)
		case proto.ClientUnSubscribe:
			unsubscribe(peerObj, data, accept, reject)
		case proto.ClientTrickleICE:
			trickle(peerObj, data, accept, reject)
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
		if method == proto.ClientBroadcast {
			broadcast(peerObj, Data)
		}
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

	// 删除以前加入过的房间数据
	for _, room := range GetRoomsByPeer(uid) {
		ridTmp := room.GetID()
		amqp.RPCCall(reg.GetRPCChannel(*islb), util.Map("method", proto.IslbOnStreamRemove, "rid", ridTmp, "uid", uid, "mid", ""), "")
		amqp.RPCCall(reg.GetRPCChannel(*islb), util.Map("method", proto.IslbClientOnLeave, "rid", ridTmp, "uid", uid), "")
		DelPeer(ridTmp, uid)
	}

	// 重新加入房间
	AddPeer(rid, peer)
	// 通知房间其他人
	amqp.RPCCall(reg.GetRPCChannel(*islb), util.Map("method", proto.IslbClientOnJoin, "rid", rid, "uid", uid, "info", info), "")

	// 查询房间存在的发布流
	ch := make(chan int, 1)
	nCount := 0
	respIslb := func(resp map[string]interface{}) {
		nLen := int(resp["len"].(float64))
		if nLen > 0 {
			uid := resp["uid"]
			mid := resp["mid"]
			nid := resp["nid"]
			minfo := resp["minfo"]
			if mid != "" {
				peer.Notify(proto.ClientOnStreamAdd, util.Map("rid", rid, "uid", uid, "nid", nid, "mid", mid, "minfo", minfo))
			}
			nCount = nCount + 1
			if nLen == nCount {
				ch <- 0
			}
		} else {
			ch <- 0
		}
	}
	// 获取房间所有人的发布流
	amqp.RPCCallWithResp(reg.GetRPCChannel(*islb), util.Map("method", proto.IslbGetMediaPubs, "rid", rid, "uid", uid), respIslb)
	<-ch
	close(ch)
	accept(emptyMap)
}

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
		amqp.RPCCall(reg.GetRPCChannel(*islb), util.Map("method", proto.IslbOnStreamRemove, "rid", ridTmp, "uid", uid, "mid", ""), "")
		amqp.RPCCall(reg.GetRPCChannel(*islb), util.Map("method", proto.IslbClientOnLeave, "rid", ridTmp, "uid", uid), "")
		DelPeer(ridTmp, uid)
	}
	accept(emptyMap)
}

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

	mid := getMID(uid)
	minfo := msg["minfo"].(map[string]interface{})

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	var sfu *reg.Node
	nid := util.Val(msg, "nid")
	if nid != "" {
		sfu = FindSfuNodeByID(nid)
	} else {
		sfu = FindSfuNodeByMid(mid)
	}
	if sfu == nil {
		log.Errorf("sfu node is not find")
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}

	// 获取sfu节点的resp
	ch := make(chan int, 1)
	rsp := make(map[string]interface{})
	respsfu := func(resp map[string]interface{}) {
		log.Infof("biz.publish respHandler resp=%v", resp)
		rsp = resp
		ch <- 0
	}
	amqp.RPCCallWithResp(reg.GetRPCChannel(*sfu), util.Map("method", proto.ClientPublish, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo, "jsep", jsep), respsfu)
	<-ch
	close(ch)

	amqp.RPCCall(reg.GetRPCChannel(*islb), util.Map("method", proto.IslbOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo, "nid", nid), "")
	accept(rsp)
}

// unpublish from app
func unpublish(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	// 获取参数
	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	log.Infof("biz.unpublish uid=%s msg=%v", uid, msg)

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	var sfu *reg.Node
	nid := util.Val(msg, "nid")
	if nid != "" {
		sfu = FindSfuNodeByID(nid)
	} else {
		sfu = FindSfuNodeByMid(mid)
	}
	if sfu == nil {
		log.Errorf("sfu node is not find")
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}

	amqp.RPCCall(reg.GetRPCChannel(*sfu), util.Map("method", proto.ClientUnPublish, "rid", rid, "uid", uid, "mid", mid), "")
	amqp.RPCCall(reg.GetRPCChannel(*islb), util.Map("method", proto.IslbOnStreamRemove, "rid", rid, "uid", uid, "mid", mid), "")
	accept(emptyMap)
}

func subscribe(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) || invalid(msg, "jsep", reject) {
		return
	}

	// 获取参数
	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	log.Infof("biz.subscribe uid=%s msg=%v", uid, msg)

	jsep := msg["jsep"].(map[string]interface{})
	if invalid(jsep, "sdp", reject) {
		return
	}

	var sfu *reg.Node
	nid := util.Val(msg, "nid")
	if nid != "" {
		sfu = FindSfuNodeByID(nid)
	} else {
		sfu = FindSfuNodeByMid(mid)
	}
	if sfu == nil {
		log.Errorf("sfu node is not find")
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}

	// 订阅流回调
	ch := make(chan int, 1)
	rsp := make(map[string]interface{})
	respSfu := func(resp map[string]interface{}) {
		rsp = resp
	}
	amqp.RPCCallWithResp(reg.GetRPCChannel(*sfu), util.Map("method", proto.ClientSubscribe, "rid", rid, "uid", uid, "mid", mid), respSfu)
	<-ch
	close(ch)
	accept(rsp)
}

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
		sfu = FindSfuNodeByMid(mid)
	}
	if sfu == nil {
		log.Errorf("sfu node is not find")
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}

	amqp.RPCCall(reg.GetRPCChannel(*sfu), util.Map("method", proto.ClientUnSubscribe, "rid", rid, "uid", uid, "mid", mid), "")
	accept(emptyMap)
}

func trickle(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	ice := util.Val(msg, "ice")
	ispub := util.Val(msg, "ispub")
	log.Infof("biz.trickle uid=%s msg=%v", uid, msg)

	var sfu *reg.Node
	nid := util.Val(msg, "nid")
	if nid != "" {
		sfu = FindSfuNodeByID(nid)
	} else {
		sfu = FindSfuNodeByMid(mid)
	}
	if sfu == nil {
		log.Errorf("sfu node is not find")
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}

	amqp.RPCCall(reg.GetRPCChannel(*sfu), util.Map("method", proto.ClientTrickleICE, "rid", rid, "uid", uid, "mid", mid, "ice", ice, "ispub", ispub), "")
	accept(emptyMap)
}

// broadcast 客户端发送广播给对方
func broadcast(peer *peer.Peer, data json.RawMessage) {
	uid := peer.ID()
	str := string(data)
	// 发送给房间其他人
	for _, room := range GetRoomsByPeer(uid) {
		rid := room.GetID()
		NotifyAllWithoutPeer(rid, peer, proto.ClientBroadcast, util.Unmarshal(str))
	}
}
