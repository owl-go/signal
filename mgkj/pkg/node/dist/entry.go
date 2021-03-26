package dist

import (
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/util"
	"mgkj/pkg/ws"
)

func Entry(method string, peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	log.Infof("method => %s, data => %v", method, msg)
	switch method {
	case proto.ClientToDistLogin:
		login(peer, msg, accept, reject)
	case proto.ClientToDistLogout:
		logout(peer, msg, accept, reject)
	case proto.ClientToDistHeartbeat:
		heartbeat(peer, msg, accept, reject)
	case proto.ClientToDistCall:
		call(peer, msg, accept, reject)
	case proto.ClientToDistAnswer:
		acceptcall(peer, msg, accept, reject)
	case proto.ClientToDistReject:
		rejectcall(peer, msg, accept, reject)
	default:
		reject(codeUnknownErr, codeStr(codeUnknownErr))
	}
}

/*
  "request":true
  "id":3764139
  "method":"login"
  "data":{}
*/
// login 登录服务器
func login(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	// 获取参数
	uid := peer.ID()
	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("islb rpc not found")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	// 通知islb更新数据库
	nid := node.NodeInfo().Nid
	rpc.AsyncRequest(proto.DistToIslbLogin, util.Map("uid", uid, "nid", nid))
	// resp
	accept(emptyMap)
}

/*
  "request":true
  "id":3764139
  "method":"logout"
  "data":{}
*/
// logout 登出服务器
func logout(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	// 获取参数
	uid := peer.ID()
	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("islb rpc not found")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	// 通知islb更新数据库
	nid := node.NodeInfo().Nid
	rpc.AsyncRequest(proto.DistToIslbLogout, util.Map("uid", uid, "nid", nid))
	// resp
	accept(emptyMap)
}

/*
  "request":true
  "id":3764139
  "method":"heartbeat"
  "data":{}
*/
// heartbeat 心跳
func heartbeat(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	// 获取参数
	uid := peer.ID()
	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("islb rpc not found")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	// 通知islb更新数据库
	nid := node.NodeInfo().Nid
	rpc.AsyncRequest(proto.DistToIslbPeerHeartbeat, util.Map("uid", uid, "nid", nid))
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
func call(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	// 判断参数正确性
	if invalid(msg, "rid", reject) {
		return
	}

	// 获取参数
	caller := peer.ID()
	rid := msg["rid"]
	peers := msg["peers"].([]interface{})
	ctype := msg["type"].(string)
	// 查询islb节点
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node is not find")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("islb rpc not found")
		reject(codeIslbErr, codeStr(codeIslbErr))
		return
	}

	// 分析数据
	nCount := 0
	peersTmp := make([]string, 0)
	for _, calleer := range peers {
		callee := calleer.(string)
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
				find := false
				rpc, find := rpcs[dist.Nid]
				if !find {
					nCount = nCount + 1
					peersTmp = append(peersTmp, callee)
				} else {
					rpc.AsyncRequest(proto.DistToDistCall, util.Map("caller", caller, "callee", callee, "rid", rid, "nid", node.NodeInfo().Nid, "type", ctype, "peers", peers))
				}
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
    "caller":"$caller"  // 主叫者uid
    "rid":"$rid"    // 主叫者房间
    "nid":"$nid"    // 主叫者所在服务器的id
  }
*/
// acceptcall 接受呼叫
func acceptcall(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
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

	find := false
	rpc, find := rpcs[dist.Nid]
	if !find {
		log.Errorf("dist rpc not found")
		reject(codeDistErr, codeStr(codeDistErr))
		return
	}

	// 发送消息
	rpc.AsyncRequest(proto.DistToDistAnswer, util.Map("caller", caller, "callee", callee, "rid", rid, "nid", nid))

	// resp
	accept(emptyMap)
}

/*
  "request":true
  "id":3764139
  "method":"reject"
  "data":{
    "caller":"$caller"  // 主叫者uid
    "rid":"$rid"    // 主叫者房间
    "nid":"$nid"    // 主叫者所在服务器的id
    "code":1,       // 1-自己拒绝，2-忙线
  }
*/
// rejectcall 拒绝呼叫
func rejectcall(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
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

	find := false
	rpc, find := rpcs[dist.Nid]
	if !find {
		log.Errorf("dist rpc not found")
		reject(codeDistErr, codeStr(codeDistErr))
		return
	}

	// 发送消息
	rpc.AsyncRequest(proto.DistToDistReject, util.Map("caller", caller, "callee", callee, "rid", rid, "nid", nid, "code", code))

	// resp
	accept(emptyMap)
}
