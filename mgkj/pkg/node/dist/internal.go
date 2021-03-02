package node

import (
	"encoding/json"
	"fmt"
	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/server"
	"github.com/cloudwebrtc/go-protoo/transport"
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	reg "mgkj/pkg/server"
	"mgkj/pkg/util"
	"net/http"
	"sync"
)

var (
	wsServer *server.WebSocketServer
	peers    map[string]*peer.Peer
	peerLock sync.RWMutex
)

/*
	"method", proto.DistToDistCall, "caller", caller, "callee", callee, "rid", rid, "nid", nid
*/
func dist2distCall(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	callee := msg["callee"].(string)
	peer := peers[callee]
	if peer != nil {
		data := make(map[string]interface{})
		data["caller"] = msg["caller"]
		data["rid"] = msg["rid"]
		data["nid"] = msg["nid"]
		peer.Notify(proto.DistToClientCall, data)
	}
	return util.Map(), nil
}

/*
	"method", proto.DistToDistAnswer, "caller", caller, "callee", callee, "rid", rid, "nid", nid
*/
func dist2distAnswer(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	caller := msg["caller"].(string)
	peer := peers[caller]
	if peer != nil {
		data := make(map[string]interface{})
		data["callee"] = msg["callee"]
		data["rid"] = msg["rid"]
		data["nid"] = msg["nid"]
		peer.Notify(proto.DistToClientAnswer, data)
	}
	return util.Map(), nil
}

/*
	"method", proto.DistToDistReject, "caller", caller, "callee", callee, "rid", rid, "nid", nid, "code", code
*/
func dist2distReject(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	caller := msg["caller"].(string)
	peer := peers[caller]
	if peer != nil {
		data := make(map[string]interface{})
		data["callee"] = msg["callee"]
		data["rid"] = msg["rid"]
		data["nid"] = msg["nid"]
		data["code"] = msg["code"]
		peer.Notify(proto.DistToClientReject, data)
	}
	return util.Map(), nil
}

// handleRPCMsgs 处理其他模块发送过来的消息
func handleRPCRequest(rpcID string) {

	log.Infof("handleRPCRequest: rpcID => [%v]", rpcID)

	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		go func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {

			defer util.Recover("dist.handleRPCRequest")

			log.Infof("dist.handleRPCRequest recv request=%v", request)
			method := request["method"].(string)
			data := request["data"].(map[string]interface{})
			log.Infof("method => %s, data => %v", method, data)

			var result map[string]interface{}
			err := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))

			switch method {
			case proto.DistToDistCall:
				result, err = dist2distCall(data)
			case proto.DistToDistAnswer:
				result, err = dist2distAnswer(data)
			case proto.DistToDistReject:
				result, err = dist2distReject(data)
			default:
				log.Warnf("dist.handleRPCMsgs invalid protocol method=%s", method)
			}
			if err != nil {
				reject(err.Code, err.Reason)
			} else {
				accept(result)
			}
		}(request, accept, reject)
	})
}

// InitWebSocket 初始化ws服务
func InitWebSocket(host string, port int, cert, key string) {
	peers = make(map[string]*peer.Peer)
	wsServer = server.NewWebSocketServer(handleWebSocket)
	config := server.DefaultConfig()
	config.Host = host
	config.Port = port
	config.CertFile = cert
	config.KeyFile = key
	go wsServer.Bind(config)
}

func handleWebSocket(transport *transport.WebSocketTransport, request *http.Request) {
	//ws://127.0.0.1:8443/ws?peer=alice
	params := request.URL.Query()
	param := params["peer"]
	if param == nil || len(param) < 1 {
		log.Errorf("dist handleWebSocket not find peer id")
		return
	}
	// 创建peer对象
	uid := param[0]
	peerobj := peer.NewPeer(uid, transport)
	log.Infof("dist handleWebSocket peer = %s", uid)
	// 加到map中
	peerLock.Lock()
	if peers[uid] != nil {
		peers[uid].Close()
	}
	peers[uid] = peerobj
	peerLock.Unlock()

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
			log.Errorf("dist handleRequest error = %s", err.Error())
		}

		method := request.Method
		switch method {
		case proto.ClientToDistLoginin:
			loginin(peerobj, data, accept, reject)
		case proto.ClientToDistLoginOut:
			loginout(peerobj, data, accept, reject)
		case proto.ClientToDistHeart:
			heart(peerobj, data, accept, reject)
		case proto.ClientToDistCall:
			call(peerobj, data, accept, reject)
		case proto.ClientToDistAnswer:
			callanswer(peerobj, data, accept, reject)
		case proto.ClientToDistReject:
			callreject(peerobj, data, accept, reject)
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
		log.Infof("dist handleNotification data = %s", string(Data))
	}

	handleClose := func(code int, err string) {
		log.Infof("handleClose err = %d, %s", code, err)
		// 加到map中
		peerLock.Lock()
		if peers[uid] != nil {
			peers[uid].Close()
		}
		delete(peers, uid)
		peerLock.Unlock()

	}

	_, _, _ = handleRequest, handleNotification, handleClose

	for {
		select {
		case resp := <-peerobj.OnNotification:
			handleNotification(resp.Notification, resp.Method, resp.Data)
		case resp := <-peerobj.OnRequest:
			handleRequest(resp.Request, resp.Accept, resp.Reject)
		case resp := <-peerobj.OnClose:
			handleClose(resp.Code, resp.Text)
		case resp := <-peerobj.OnError:
			handleClose(resp.Code, resp.Text)
		}
	}
}

/*
	"request":true
	"id":3764139
	"method":"loginin"
	"data":{}
*/
// loginin 登录服务器
func loginin(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	// 获取参数
	uid := peer.ID()
	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	// 通知islb更新数据库
	nid := node.NodeInfo().Nid
	rpc := protoo.NewRequestor(reg.GetRPCChannel(*islb))
	rpc.AsyncRequest(proto.DistToIslbLoginin, util.Map("uid", uid, "nid", nid))
	// resp
	accept(emptyMap)
}

/*
	"request":true
	"id":3764139
	"method":"loginout"
	"data":{}
*/
// loginout 退录服务器
func loginout(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	// 获取参数
	uid := peer.ID()
	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	// 通知islb更新数据库
	nid := node.NodeInfo().Nid
	rpc := protoo.NewRequestor(reg.GetRPCChannel(*islb))
	rpc.AsyncRequest(proto.DistToIslbLoginOut, util.Map("uid", uid, "nid", nid))
	// resp
	accept(emptyMap)
}

/*
	"request":true
	"id":3764139
	"method":"heart"
	"data":{}
*/
// heart 心跳
func heart(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	// 获取参数
	uid := peer.ID()
	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	// 通知islb更新数据库
	nid := node.NodeInfo().Nid
	rpc := protoo.NewRequestor(reg.GetRPCChannel(*islb))
	rpc.AsyncRequest(proto.DistToIslbPeerHeart, util.Map("uid", uid, "nid", nid))
	// resp
	accept(emptyMap)
}

/*
	"request":true
	"id":3764139
	"method":"call"
	"data":{
		"rid":"$rid"
		"peers": {
			"64236c21-21e8-4a3d-9f80-c767d1e1d67f",
			"65236c21-21e8-4a3d-9f80-c767d1e1d67f",
			"66236c21-21e8-4a3d-9f80-c767d1e1d67f",
			"67236c21-21e8-4a3d-9f80-c767d1e1d67f",
		}
	}
*/
// call 发送呼叫
func call(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	// 判断参数正确性
	if invalid(msg, "rid", reject) {
		return
	}

	// 获取参数
	caller := peer.ID()
	rid := msg["rid"]
	peers := msg["peers"].([]interface{})

	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	// 分析数据
	nCount := 0
	peersTmp := make([]string, 0)
	for _, calleer := range peers {
		callee := calleer.(string)

		rpc := protoo.NewRequestor(reg.GetRPCChannel(*islb))
		resp, err := rpc.SyncRequest(proto.DistToIslbPeerInfo, util.Map("uid", callee))
		if err != nil {
			return
		}
		/* "method", proto.IslbToDistPeerInfo, "errorCode", 0, "nid", dist */
		/* "method", proto.IslbToDistPeerInfo, "errorCode", 1 */
		nErr := int(resp["errorCode"].(float64))
		if nErr == 0 {
			// 获取节点
			nid := resp["nid"].(string)
			dist := FindDistNodeByID(nid)
			if dist != nil {
				rpc := protoo.NewRequestor(reg.GetRPCChannel(*dist))
				rpc.AsyncRequest(proto.DistToDistCall, util.Map("caller", caller, "callee", callee, "rid", rid, "nid", node.NodeInfo().Nid))
			} else {
				nCount = nCount + 1
				peersTmp = append(peersTmp, callee)
			}
		} else {
			nCount = nCount + 1
			peersTmp = append(peersTmp, callee)
		}
	}

	// resp
	resp := make(map[string]interface{})
	resp["len"] = nCount
	if nCount != 0 {
		resp["peers"] = peersTmp
	}
	accept(resp)
}

/*
	"request":true
	"id":3764139
	"method":"answer"
	"data":{
		"caller":"$caller"	// 主叫者uid
		"rid":"$rid"		// 主叫者房间
		"nid":"$nid"		// 主叫者所在服务器的id
	}
*/
// callanswer 接受呼叫
func callanswer(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	// 判断参数正确性
	if invalid(msg, "rid", reject) || invalid(msg, "nid", reject) {
		return
	}

	// 获取参数
	callee := peer.ID()
	caller := util.Val(msg, "caller")
	rid := util.Val(msg, "rid")
	nid := util.Val(msg, "nid")

	// 获取节点
	dist := FindDistNodeByID(nid)
	if dist == nil {
		log.Errorf("dist node is not find")
		reject(codeDistErr, codeStr(codeDistErr))
		return
	}

	// 发送消息
	rpc := protoo.NewRequestor(reg.GetRPCChannel(*dist))
	rpc.AsyncRequest(proto.DistToDistAnswer, util.Map("caller", caller, "callee", callee, "rid", rid, "nid", nid))

	// resp
	accept(emptyMap)
}

/*
	"request":true
	"id":3764139
	"method":"reject"
	"data":{
		"caller":"$caller"	// 主叫者uid
		"rid":"$rid"		// 主叫者房间
		"nid":"$nid"		// 主叫者所在服务器的id
		"code":1, 			// 1-自己拒绝，2-忙线
	}
*/
// callreject 拒绝呼叫
func callreject(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	// 判断参数正确性
	if invalid(msg, "rid", reject) || invalid(msg, "nid", reject) {
		return
	}

	// 获取参数
	callee := peer.ID()
	caller := util.Val(msg, "caller")
	rid := util.Val(msg, "rid")
	nid := util.Val(msg, "nid")
	code := int(msg["code"].(float64))

	// 获取节点
	dist := FindDistNodeByID(nid)
	if dist == nil {
		log.Errorf("dist node is not find")
		reject(codeDistErr, codeStr(codeDistErr))
		return
	}

	// 发送消息
	rpc := protoo.NewRequestor(reg.GetRPCChannel(*dist))
	rpc.AsyncRequest(proto.DistToDistReject, util.Map("caller", caller, "callee", callee, "rid", rid, "nid", nid, "code", code))

	// resp
	accept(emptyMap)
}
