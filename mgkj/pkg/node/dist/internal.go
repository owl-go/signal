package node

import (
	"encoding/json"
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	reg "mgkj/pkg/server"
	"mgkj/pkg/util"
	"net/http"
	"sync"

	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/server"
	"github.com/cloudwebrtc/go-protoo/transport"
)

var (
	wsServer *server.WebSocketServer
	peers    map[string]*peer.Peer
	peerLock sync.RWMutex
)

/*
	"method", proto.DistToDistCall, "caller", caller, "callee", callee, "rid", rid, "biz", biz, "nid", nid
*/
func dist2distCall(msg map[string]interface{}) {
	callee := msg["callee"].(string)
	peer := peers[callee]
	if peer != nil {
		data := make(map[string]interface{})
		data["caller"] = msg["caller"]
		data["rid"] = msg["rid"]
		data["biz"] = msg["biz"]
		data["nid"] = msg["nid"]
		peer.Notify(proto.DistToClientCall, data)
	}
}

/*
	"method", proto.DistToDistAnswer, "caller", caller, "callee", callee, "rid", rid, "biz", biz, "nid", nid
*/
func dist2distAnswer(msg map[string]interface{}) {
	caller := msg["caller"].(string)
	peer := peers[caller]
	if peer != nil {
		data := make(map[string]interface{})
		data["callee"] = msg["callee"]
		data["rid"] = msg["rid"]
		data["biz"] = msg["biz"]
		data["nid"] = msg["nid"]
		peer.Notify(proto.DistToClientAnswer, data)
	}
}

/*
	"method", proto.DistToDistReject, "caller", caller, "callee", callee, "rid", rid, "biz", biz, "nid", nid, "code", code
*/
func dist2distReject(msg map[string]interface{}) {
	caller := msg["caller"].(string)
	peer := peers[caller]
	if peer != nil {
		data := make(map[string]interface{})
		data["callee"] = msg["callee"]
		data["rid"] = msg["rid"]
		data["biz"] = msg["biz"]
		data["nid"] = msg["nid"]
		data["code"] = msg["code"]
		peer.Notify(proto.DistToClientReject, data)
	}
}

// handleRPCMsgs 处理其他模块发送过来的消息
func handleRPCMsgs() {
	rpcMsgs, err := amqp.ConsumeRPC()
	if err != nil {
		log.Errorf("dist handleRPCMsgs ConsumeRPC err = %s", err.Error())
		return
	}

	go func() {
		defer util.Recover("dist.handleRPCMsgs")
		for rpcm := range rpcMsgs {
			var msg map[string]interface{}
			err := json.Unmarshal(rpcm.Body, &msg)
			if err != nil {
				log.Errorf("dist handleRPCMsgs Unmarshal err = %s", err.Error())
			}

			from := rpcm.ReplyTo
			corrID := rpcm.CorrelationId
			log.Infof("dist.handleRPCMsgs recv msg=%v, from=%s, corrID=%s", msg, from, corrID)

			method := util.Val(msg, "method")
			switch method {
			case proto.IslbToDistPeerInfo:
				amqp.Emit(corrID, msg)
			case proto.DistToDistCall:
				dist2distCall(msg)
			case proto.DistToDistAnswer:
				dist2distAnswer(msg)
			case proto.DistToDistReject:
				dist2distReject(msg)
			default:
				log.Warnf("dist.handleRPCMsgs invalid protocol method=%s", method)
			}
		}
	}()
}

// InitWebSocket 初始化ws服务
func InitWebSocket(host string, port int, cert, key string) {
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
	// 处理客户端的请求
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

// loginin 登录服务器
func loginin(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("dist handle loginin uid = %s, msg = %v", peer.ID(), msg)
	// 获取参数
	uid := peer.ID()
	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}
	// 查询biz节点
	biz := FindBizNodeByPayload()
	if biz == nil {
		log.Errorf("biz node is not find")
		reject(codeBizErr, codeStr(codeBizErr))
		return
	}
	// 查询sfu节点
	sfu := FindSfuNodeByPayload()
	if sfu == nil {
		log.Errorf("sfu node is not find")
		reject(codeSfuErr, codeStr(codeSfuErr))
		return
	}
	// 通知islb更新数据库
	nid := node.NodeInfo().Nid
	amqp.RPCCall(reg.GetRPCChannel(*islb), util.Map("method", proto.DistToIslbLoginin, "uid", uid, "nid", nid), "")
	// resp
	resp := make(map[string]interface{})
	resp["biz"] = biz.Nip
	resp["sfu"] = sfu.Nid
	accept(resp)
}

// loginout 退录服务器
func loginout(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("dist handle loginout uid = %s, msg = %v", peer.ID(), msg)
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
	amqp.RPCCall(reg.GetRPCChannel(*islb), util.Map("method", proto.DistToIslbLoginOut, "uid", uid, "nid", nid), "")
	// resp
	accept(emptyMap)
}

// heart 心跳
func heart(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("dist handle heart uid = %s, msg = %v", peer.ID(), msg)
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
	amqp.RPCCall(reg.GetRPCChannel(*islb), util.Map("method", proto.DistToIslbPeerHeart, "uid", uid, "nid", nid), "")
	// resp
	accept(emptyMap)
}

// call 发送呼叫
func call(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("dist handle call uid = %s, msg = %v", peer.ID(), msg)
	// 判断参数正确性
	if invalid(msg, "rid", reject) || invalid(msg, "biz", reject) {
		return
	}

	// 获取参数
	caller := peer.ID()
	rid := msg["rid"]
	biz := msg["biz"]
	peers := msg["peers"].([]string)

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
	for _, callee := range peers {
		ch := make(chan int, 1)
		respIslb := func(resp map[string]interface{}) {
			/* "method", proto.IslbToDistPeerInfo, "errorCode", 0, "nid", dist */
			/* "method", proto.IslbToDistPeerInfo, "errorCode", 1, "errorReason", "uid is not live" */
			err := int(resp["errorCode"].(float64))
			if err == 0 {
				// 获取节点
				nid := resp["nid"].(string)
				dist := FindDistNodeByID(nid)
				if dist != nil {
					amqp.RPCCall(reg.GetRPCChannel(*dist), util.Map("method", proto.DistToDistCall,
						"caller", caller, "callee", callee, "rid", rid, "biz", biz, "nid", node.NodeInfo().Nid), "")
				} else {
					nCount = nCount + 1
					peersTmp = append(peersTmp, callee)
				}
			} else {
				nCount = nCount + 1
				peersTmp = append(peersTmp, callee)
			}
			ch <- 0
		}
		amqp.RPCCallWithResp(reg.GetRPCChannel(*islb), util.Map("method", proto.DistToIslbPeerInfo, "uid", callee), respIslb)
		<-ch
		close(ch)
	}

	// resp
	resp := make(map[string]interface{})
	resp["len"] = nCount
	if nCount != 0 {
		resp["peers"] = peersTmp
	}
	accept(resp)
}

// callanswer 接受呼叫
func callanswer(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("dist handle callanswer uid = %s, msg = %v", peer.ID(), msg)
	// 判断参数正确性
	if invalid(msg, "rid", reject) || invalid(msg, "biz", reject) || invalid(msg, "nid", reject) {
		return
	}

	// 获取参数
	callee := peer.ID()
	caller := util.Val(msg, "caller")
	rid := util.Val(msg, "rid")
	biz := util.Val(msg, "biz")
	nid := util.Val(msg, "nid")

	// 获取节点
	dist := FindDistNodeByID(nid)
	if dist == nil {
		log.Errorf("dist node is not find")
		reject(codeDistErr, codeStr(codeDistErr))
		return
	}

	// 发送消息
	amqp.RPCCall(reg.GetRPCChannel(*dist), util.Map("method", proto.DistToDistAnswer,
		"caller", caller, "callee", callee, "rid", rid, "biz", biz, "nid", nid), "")

	// resp
	accept(emptyMap)
}

// callreject 拒绝呼叫
func callreject(peer *peer.Peer, msg map[string]interface{}, accept peer.RespondFunc, reject peer.RejectFunc) {
	log.Infof("dist handle callreject uid = %s, msg = %v", peer.ID(), msg)
	// 判断参数正确性
	if invalid(msg, "rid", reject) || invalid(msg, "biz", reject) || invalid(msg, "nid", reject) {
		return
	}

	// 获取参数
	callee := peer.ID()
	caller := util.Val(msg, "caller")
	rid := util.Val(msg, "rid")
	biz := util.Val(msg, "biz")
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
	amqp.RPCCall(reg.GetRPCChannel(*dist), util.Map("method", proto.DistToDistReject,
		"caller", caller, "callee", callee, "rid", rid, "biz", biz, "nid", nid, "code", code), "")

	// resp
	accept(emptyMap)
}
