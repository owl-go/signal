package node

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"mgkj/pkg/log"
	"mgkj/pkg/proto"
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
	rooms    = make(map[string]*RoomObj)
	roomLock sync.RWMutex
)

// InitWebSocket 初始化ws服务
func InitWebSocket(host string, port int, cert, key string) {
	wsServer = server.NewWebSocketServer(handleNewWebSocket)
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
		info := "biz.checkRoom start\n"
		roomLock.Lock()
		for rid, roomobj := range rooms {
			info += fmt.Sprintf("room: %s peers: %d\n", rid, len(roomobj.room.GetPeers()))
			if len(roomobj.room.GetPeers()) == 0 {
				roomobj.room.Close()
				delete(rooms, rid)
			}
		}
		roomLock.Unlock()
		log.Infof(info)
	}
}

func handleNewWebSocket(transport *transport.WebSocketTransport, request *http.Request) {
	//https://127.0.0.1:8443/ws?peer=alice
	params := request.URL.Query()
	peers := params["peer"]
	if peers == nil || len(peers) < 1 {
		return
	}
	peerID := peers[0]
	peerObj := peer.NewPeer(peerID, transport)

	handleRequest := func(request peer.Request, accept peer.RespondFunc, reject peer.RejectFunc) {
		var data = make(map[string]interface{})
		if err := json.Unmarshal(request.Data, &data); err != nil {
			log.Errorf("handleRequest error")
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
		case proto.ClientBroadcast:
			broadcast(peerObj, data, accept, reject)
		}
	}

	handleNotification := func(notification bool, method string, Data json.RawMessage) {

	}

	handleClose := func(code int, err string) {

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
		case resp := <-peerObj.OnClose:
			handleClose(resp.Code, resp.Text)
		}
	}
}

// join room
func join(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("biz.join peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	info := util.Val(msg, "info")
	// 删除以前加入的房间和流
	amqp.RPCCall(proto.IslbID, util.Map("method", proto.IslbOnPeerRemoveAll, "rid", rid, "uid", uid), "")
	// 加入房间
	AddPeer(rid, peer)
	// 通知房间其他人
	amqp.RPCCall(proto.IslbID, util.Map("method", proto.IslbClientOnJoin, "rid", rid, "uid", uid, "info", info), "")

	respHandler := func(resp map[string]interface{}) {
		uid := resp["uid"]
		mid := resp["mid"]
		minfo := resp["minfo"]
		log.Infof("biz.join respHandler mid=%v info=%v", mid, minfo)
		if mid != "" {
			peer.Notify(proto.ClientOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "minfo", minfo))
		}
	}
	// 获取房间所有人的发布流
	amqp.RPCCallWithResp(proto.IslbID, util.Map("method", proto.IslbGetPubs, "rid", rid, "uid", uid), respHandler)
	accept(emptyMap)
}

func leave(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("biz.leave peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	// 删除加入的房间和流
	amqp.RPCCall(proto.IslbID, util.Map("method", proto.IslbOnPeerRemoveAll, "rid", rid, "uid", uid), "")
	accept(emptyMap)
	// 删除这个人
	DelPeer(rid, uid)
}

func publish(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("biz.publish peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "rid", reject) || invalid(msg, "jsep", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")

	jsep := msg["jsep"].(map[string]interface{})
	if invalid(jsep, "sdp", reject) {
		return
	}

	roomobj := GetRoom(rid)
	if roomobj == nil {
		reject(codeRoomErr, codeStr(codeRoomErr))
		return
	}

	//sdp := util.Val(jsep, "sdp")
	options := msg["options"].(map[string]interface{})
	mid := getMID(uid)
	amqp.RPCCall(proto.IslbID, util.Map("method", proto.IslbOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "minfo", options), "")
	accept(emptyMap)
}

// unpublish from app
func unpublish(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("signal.unpublish peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	amqp.RPCCall(proto.IslbID, util.Map("method", proto.IslbOnStreamRemove, "rid", rid, "uid", uid, "mid", mid), "")
	accept(emptyMap)
}

func subscribe(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("biz.subscribe peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) || invalid(msg, "jsep", reject) {
		return
	}
	jsep := msg["jsep"].(map[string]interface{})
	if invalid(jsep, "sdp", reject) {
		return
	}
	accept(emptyMap)
}

func unsubscribe(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("biz.unsubscribe peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}
	accept(emptyMap)
}

func broadcast(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("biz.broadcast peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "rid", reject) || invalid(msg, "uid", reject) {
		return
	}
	rid, uid, data := util.Val(msg, "rid"), util.Val(msg, "uid"), util.Val(msg, "data")
	amqp.RPCCall(proto.IslbID, util.Map("method", proto.IslbOnBroadcast, "rid", rid, "uid", uid, "data", data), "")
	accept(emptyMap)
}
