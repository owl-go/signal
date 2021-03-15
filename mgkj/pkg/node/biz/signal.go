package biz

import (
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/util"
	"mgkj/pkg/ws"
	"sync"
	"time"
)

const (
	statCycle = 5 * time.Second
)

var (
	rooms    = make(map[string]*RoomNode)
	roomLock sync.RWMutex
	wsReq    func(method string, peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc)
)

// InitSignalServer 初始化biz服务器
func InitSignalServer(host string, port int, cert, key string) {
	initWebSocket(host, port, cert, key, Entry)
	go checkRoom()
}

func initWebSocket(host string, port int, cert, key string, handler interface{}) {
	wsServer := ws.NewWebSocketServer(in)
	config := ws.DefaultConfig()
	config.Host = host
	config.Port = port
	config.CertFile = cert
	config.KeyFile = key
	wsReq = handler.(func(method string, peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc))
	go wsServer.Bind(config)
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

					find := false
					rpc, find := rpcs[islb.Nid]
					if !find {
						log.Errorf("FindPeerIsLive islb rpc not found")
					} else {
						rpc.AsyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
						rpc.AsyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))
						node.room.RemovePeer(uid)
					}
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
