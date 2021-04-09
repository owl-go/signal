package biz

import (
	"fmt"
	"mgkj/pkg/proto"
	"mgkj/pkg/timing"
	"mgkj/pkg/util"
	"mgkj/pkg/ws"
	"sync"
	"time"
)

const (
	statCycle         = 10 * time.Second
	streamTimingCycle = 5 * time.Second
)

var (
	substreams     = make(map[string]*timing.StreamTimer)
	substreamsLock sync.RWMutex
	rooms          = make(map[string]*RoomNode)
	roomLock       sync.RWMutex
	wsReq          func(method string, peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc)
)

// InitSignalServer 初始化biz服务器
func InitSignalServer(host string, port int, cert, key string) {
	initWebSocket(host, port, cert, key, Entry)
	go checkRoom()
	go checkStreamState()
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
				bLive := FindPeerIsLive(rid, uid)
				if !bLive {
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

					rpc.AsyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
					rpc.AsyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))
					node.room.RemovePeer(uid)
					//stop all this user's timers
					stopAllStreamTimer(rid, uid)
				}
			}
			if len(node.room.GetPeers()) == 0 {
				node.room.Close()
				delete(rooms, rid)
			}
		}
		roomLock.Unlock()
	}
}

func stopAllStreamTimer(rid, uid string) {
	for _, timer := range substreams {
		if timer.RID == rid && timer.UID == uid && !timer.IsStopped() {
			timer.Stop()
			//log.Infof("biz.stopAllStreamTimer room %s uid =%s sid = %s stopped.", timer.RID,timer.UID, timer.SID)
			logger.Infof(fmt.Sprintf("biz.stopAllStreamTimer room %s uid=%s mid=%s sid=%s stopped.", timer.RID, timer.UID, timer.MID, timer.SID),
				"uid", timer.UID, "rid", timer.RID, "mid", timer.MID, "sid", timer.SID)
		}
	}
}

func checkStreamState() {
	t := time.NewTicker(streamTimingCycle)
	defer t.Stop()
	for range t.C {
		for sid, timer := range substreams {
			if timer.IsStopped() {
				//log.Infof("checkStreamState uid = %s,mid = %s,sid = %s stream was stopped", timer.UID, timer.MID, timer.SID)
				logger.Infof(fmt.Sprintf("checkStreamState room %s uid=%s,mid=%s,sid=%s stream was stopped", timer.RID, timer.UID, timer.MID, timer.SID),
					"uid", timer.UID, "rid", timer.RID, "mid", timer.MID, "sid", timer.SID)

				rpc := getIssrRequestor()
				if rpc == nil {
					logger.Errorf("biz.checkStreamState get issr requestor failed", "uid", timer.UID, "rid", timer.RID, "mid", timer.MID, "sid", timer.SID)
					continue
				}

				_, err := rpc.SyncRequest(proto.BizToIssrReportStreamState, util.Map("rid", timer.RID, "appid", timer.AppID, "uid", timer.UID, "mid", timer.MID,
					"sid", timer.SID, "mediatype", timer.MediaType, "resolution", timer.Resolution, "seconds", timer.GetTotalTime()))

				if err != nil {
					logger.Errorf(fmt.Sprintf("biz.checkStreamState rpc err=%v", err.Reason), "uid", timer.UID, "rid", timer.RID, "mid", timer.MID, "sid", timer.SID)
					logger.Errorf(fmt.Sprintf("biz.checkStreamState report room %s uid=%s,mid=%s, sid=%s stream state failed", timer.RID, timer.UID,
						timer.MID, timer.SID), "uid", timer.UID, "rid", timer.RID, "mid", timer.MID, "sid", timer.SID)
				} else {
					substreamsLock.Lock()
					delete(substreams, sid)
					substreamsLock.Unlock()
				}
			}
		}
	}
}
