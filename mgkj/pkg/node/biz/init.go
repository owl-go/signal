package biz

import (
	"errors"
	dis "mgkj/infra/discovery"
	logger2 "mgkj/infra/logger"
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/timing"
	"mgkj/util"

	nprotoo "github.com/gearghost/nats-protoo"
)

var (
	logger *logger2.Logger
	nats   *nprotoo.NatsProtoo
	node   *dis.ServiceNode
	watch  *dis.ServiceWatcher
	rpcs   = make(map[string]*nprotoo.Requestor)
)

// Init 初始化服务
func Init(serviceNode *dis.ServiceNode, ServiceWatcher *dis.ServiceWatcher, natsURL string, log *logger2.Logger) {
	node = serviceNode
	watch = ServiceWatcher
	nats = nprotoo.NewNatsProtoo(util.GenerateNatsUrlString(natsURL))
	logger = log
	handleRPCRequest(node.GetRPCChannel())
	go watch.WatchServiceNode("", WatchServiceCallBack)
}

// Close 关闭连接
func Close() {
	if nats != nil {
		nats.Close()
	}
	if node != nil {
		node.Close()
	}
	if watch != nil {
		watch.Close()
	}
}

// WatchServiceCallBack 查看所有的Node节点
func WatchServiceCallBack(state dis.NodeStateType, node dis.Node) {
	if state == dis.ServerUp {
		// 判断是否广播节点
		if node.Name == "islb" || node.Name == "sfu" {
			eventID := dis.GetEventChannel(node)
			nats.OnBroadcast(eventID, handleBroadcast)
		}

		id := node.Nid
		_, found := rpcs[id]
		if !found {
			rpcID := dis.GetRPCChannel(node)
			rpcs[id] = nats.NewRequestor(rpcID)
		}
	} else if state == dis.ServerDown {
		delete(rpcs, node.Nid)
	}
}

// FindIslbNode 查询全局的可用的islb节点
func FindIslbNode() *dis.Node {
	servers, find := watch.GetNodes("islb")
	if find {
		for _, node := range servers {
			return &node
		}
	}
	return nil
}

// FindBizNodeByID 查询指定id的biz节点
func FindBizNodeByID(nid string) *dis.Node {
	biz, find := watch.GetNodeByID(nid)
	if find {
		return biz
	}
	return nil
}

// FindBizNodeByUid 根据rid, uid查询指定的biz节点
func FindBizNodeByUid(rid, uid string) *dis.Node {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindBizNodeByUid islb not found")
		return nil
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("FindBizNodeByUid islb rpc not found")
		return nil
	}

	resp, err := rpc.SyncRequest(proto.BizToIslbGetBizInfo, util.Map("rid", rid, "uid", uid))
	if err != nil {
		log.Errorf(err.Reason)
		return nil
	}

	log.Infof("FindBizNodeByUid resp ==> %v", resp)

	var biz *dis.Node
	nid := util.Val(resp, "nid")
	if nid != "" {
		biz = FindBizNodeByID(nid)
	}

	return biz
}

// FindSfuNodeByID 查询指定id的sfu节点
func FindSfuNodeByID(nid string) *dis.Node {
	sfu, find := watch.GetNodeByID(nid)
	if find {
		return sfu
	}
	return nil
}

// FindSfuNodeByPayload 查询指定区域下的可用的sfu节点
func FindSfuNodeByPayload() *dis.Node {
	sfu, find := watch.GetNodeByPayload(node.NodeInfo().Ndc, "sfu")
	if find {
		return sfu
	}
	return nil
}

// FindSfuNodeByMid 根据rid, mid查询指定的sfu节点
func FindSfuNodeByMid(rid, mid string) *dis.Node {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindSfuNodeByMid islb not found")
		return nil
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("FindSfuNodeByMid islb rpc not found")
		return nil
	}

	resp, err := rpc.SyncRequest(proto.BizToIslbGetSfuInfo, util.Map("rid", rid, "mid", mid))
	if err != nil {
		log.Errorf(err.Reason)
		return nil
	}

	log.Infof("FindSfuNodeByMid resp ==> %v", resp)

	var sfu *dis.Node
	nid := util.Val(resp, "nid")
	if nid != "" {
		sfu = FindSfuNodeByID(nid)
	}

	return sfu
}

// FindMcuNodeByID 查询指定id的mcu节点
func FindMcuNodeByID(nid string) *dis.Node {
	mcu, find := watch.GetNodeByID(nid)
	if find {
		return mcu
	}
	return nil
}

// FindMcuNodeByPayload 查询指定区域下的可用的mcu节点
func FindMcuNodeByPayload() *dis.Node {
	mcu, find := watch.GetNodeByPayload(node.NodeInfo().Ndc, "mcu")
	if find {
		return mcu
	}
	return nil
}

// SetMcuNodeByRid 设置rid跟mcu绑定关系
func SetMcuNodeByRid(rid, nid string) error {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindMcuNodeByMid islb not found")
		return errors.New("FindMcuNodeByMid islb not found")
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("FindMcuNodeByMid islb rpc not found")
		return errors.New("FindMcuNodeByMid islb rpc not found")
	}

	resp, err := rpc.SyncRequest(proto.BizToIslbSetMcuInfo, util.Map("rid", rid, "nid", nid))
	if err != nil {
		log.Errorf(err.Reason)
		return errors.New(err.Reason)
	}

	log.Infof("FindMcuNodeByMid resp ==> %v", resp)

	return nil

}

// FindMcuNodeByRid 根据rid查询指定的mcu节点
func FindMcuNodeByRid(rid string) *dis.Node {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindMcuNodeByMid islb not found")
		return nil
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("FindMcuNodeByMid islb rpc not found")
		return nil
	}

	resp, err := rpc.SyncRequest(proto.BizToIslbGetMcuInfo, util.Map("rid", rid))
	if err != nil {
		log.Errorf(err.Reason)
		return nil
	}

	log.Infof("FindMcuNodeByMid resp ==> %v", resp)

	var mcu *dis.Node
	nid := util.Val(resp, "nid")
	if nid != "" {
		mcu = FindMcuNodeByID(nid)
	}

	return mcu
}

// FindRoomUsers 查询房间所有人信息
func FindRoomUsers(uid, rid string) (bool, []interface{}) {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindRoomUsers islb not found")
		return false, nil
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("FindRoomUsers islb rpc not found")
		return false, nil
	}

	resp, err := rpc.SyncRequest(proto.BizToIslbGetRoomUsers, util.Map("rid", rid, "uid", uid))
	if err != nil {
		log.Errorf(err.Reason)
		return false, nil
	}

	log.Infof("FindRoomUsers resp ==> %v", resp)

	if resp["users"] == nil {
		log.Errorf("FindRoomUsers pubs is nil")
		return false, nil
	}

	users := resp["users"].([]interface{})
	return true, users
}

// FindMediaPubs 查询房间所有的其他人的发布流
func FindMediaPubs(uid, rid string) (bool, []interface{}) {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindMediaPubs islb not found")
		return false, nil
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("FindMediaPubs islb rpc not found")
		return false, nil
	}

	resp, err := rpc.SyncRequest(proto.BizToIslbGetMediaPubs, util.Map("rid", rid, "uid", uid))
	if err != nil {
		log.Errorf(err.Reason)
		return false, nil
	}

	log.Infof("FindMediaPubs resp ==> %v", resp)

	if resp["pubs"] == nil {
		log.Errorf("FindMediaPubs pubs is nil")
		return false, nil
	}

	pubs := resp["pubs"].([]interface{})
	return true, pubs

	/*
		roomid := resp["rid"].(string)
		pubs := resp["pubs"].([]interface{})
		for _, pub := range pubs {
			uid := pub.(map[string]interface{})["uid"].(string)
			mid := pub.(map[string]interface{})["mid"].(string)
			nid := pub.(map[string]interface{})["nid"].(string)
			minfo := pub.(map[string]interface{})["minfo"].(map[string]interface{})
			if mid != "" {
				peer.Notify(proto.BizToClientOnStreamAdd, util.Map("rid", roomid, "uid", uid, "mid", mid, "nid", nid, "minfo", minfo))
			}
		}
		return true*/
}

// findIssrNode 查询全局的可用的Issr节点
func findIssrNode() *dis.Node {
	servers, find := watch.GetNodes("issr")
	if find {
		for _, node := range servers {
			return &node
		}
	}
	return nil
}

// getIssrRequestor 查询issr服务的节点id
func getIssrRequestor() *nprotoo.Requestor {
	issr := findIssrNode()
	if issr == nil {
		log.Errorf("issr node not found")
		return nil
	}

	find := false
	rpc, find := rpcs[issr.Nid]
	if !find {
		log.Errorf("issr rpc not found")
		return nil
	}
	return rpc
}

// getIslbRequestor 查询islb服务的节点ID
func getIslbRequestor() *nprotoo.Requestor {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("islb node not found")
		return nil
	}
	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("islb rpc not found")
		return nil
	}
	return rpc
}

func reportStreamTiming(timer *timing.StreamTimer, isVideo, isInterval bool) error {
	log.Infof("reportStreamTiming:uid:%s,count:%d,lastmode:%s,mode:%s,lastres:%s,res:%s", timer.UID, timer.GetStreamsCount(), timer.GetLastMode(), timer.GetCurrentMode(),
		timer.GetLastResolution(), timer.GetCurrentResolution())

	issrRpc := getIssrRequestor()
	if issrRpc == nil {
		return errors.New("can't found issr node")
	}

	var resolution string
	var mode string
	if isVideo {
		mode = "video"
		if isInterval {
			resolution = timer.GetLastResolution()
		} else {
			resolution = timer.GetCurrentResolution()
		}
	} else {
		mode = "audio"
	}

	seconds := timer.GetTotalSeconds()

	if seconds != 0 {
		if mode == "audio" {
			_, err := issrRpc.SyncRequest(proto.BizToIssrReportStreamState, util.Map("appid", timer.AppID, "rid", timer.RID, "uid", timer.UID,
				"mediatype", mode, "seconds", seconds))
			if err != nil {
				return errors.New(err.Reason)
			}
		} else {
			if resolution != "" {
				_, err := issrRpc.SyncRequest(proto.BizToIssrReportStreamState, util.Map("appid", timer.AppID, "rid", timer.RID, "uid", timer.UID,
					"mediatype", mode, "resolution", resolution, "seconds", seconds))
				if err != nil {
					return errors.New(err.Reason)
				}
			} else {
				return errors.New("resolution is empty")
			}
		}
	}

	return nil
}
