package biz

import (
	"fmt"
	"signal/pkg/proto"
	"signal/pkg/ws"
	"signal/util"
	"sync"
	"time"
)

const (
	monitorCycle      = 5 * time.Second
	statCycle         = 10 * time.Second
	streamTimingCycle = 5 * time.Second
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
	go checkConnections()
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
		roomLock.Lock()
		for rid, node := range rooms {
			for uid := range node.room.GetPeers() {
				biz := FindBizNodeByUid(rid, uid)
				if biz == nil {
					// 查询islb节点
					islb := FindIslbNode()
					if islb == nil {
						logger.Errorf("biz.checkRoom islb node not found", "uid", uid, "rid", rid)
						continue
					}

					find := false
					rpc, find := rpcs[islb.Nid]
					if !find {
						logger.Errorf("biz.checkRoom islb rpc not found", "uid", uid, "rid", rid)
						continue
					}

					rpc.SyncRequest(proto.BizToIslbOnLiveRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
					rpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
					rpc.SyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))

					node.room.RemovePeer(uid)
				}
			}
			if len(node.room.GetPeers()) == 0 {
				logger.Infof(fmt.Sprintf("no peer in room:%s now", rid), "rid", rid)
				node.room.Close()
				delete(rooms, rid)
			}
		}
		roomLock.Unlock()
	}
}

func checkConnections() {
	t := time.NewTicker(monitorCycle)
	defer t.Stop()
	for range t.C {
		roomLock.Lock()
		count := 0
		for _, node := range rooms {
			count += len(node.room.GetPeers())
		}
		roomLock.Unlock()
		totalConnections.WithLabelValues("peers").Set(float64(count))
	}
}
