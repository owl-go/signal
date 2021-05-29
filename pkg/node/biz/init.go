package biz

import (
	"errors"
	"fmt"
	dis "signal/infra/discovery"
	logger2 "signal/infra/logger"
	"signal/pkg/log"
	"signal/pkg/proto"
	"signal/pkg/timing"
	"signal/util"

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
	logger = log
	node = serviceNode
	watch = ServiceWatcher
	nats = nprotoo.NewNatsProtoo(util.GenerateNatsUrlString(natsURL))
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
		if nid == node.NodeInfo().Nid {
			tmpNode := node.NodeInfo()
			return &tmpNode
		}
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
func SetMcuNodeByRid(rid, nid string) *dis.Node {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("SetMcuNodeByMid islb not found")
		return nil
	}

	find := false
	rpc, find := rpcs[islb.Nid]
	if !find {
		log.Errorf("SetMcuNodeByMid islb rpc not found")
		return nil
	}

	resp, err := rpc.SyncRequest(proto.BizToIslbSetMcuInfo, util.Map("rid", rid, "nid", nid))
	if err != nil {
		log.Errorf(err.Reason)
		return nil
	}

	log.Infof("SetMcuNodeByMid resp ==> %v", resp)

	var mcu *dis.Node
	id := util.Val(resp, "nid")
	if id != "" {
		mcu = FindMcuNodeByID(id)
	}
	return mcu

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
		log.Errorf("FindRoomUsers users is nil")
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
		var msg map[string]interface{}
		if mode == "audio" {
			msg = util.Map("appid", timer.AppID, "rid", timer.RID, "uid", timer.UID,
				"mediatype", mode, "seconds", seconds)
		} else {
			if resolution != "" {
				msg = util.Map("appid", timer.AppID, "rid", timer.RID, "uid", timer.UID,
					"mediatype", mode, "resolution", resolution, "seconds", seconds)
			} else {
				log.Errorf("resolution is empty")
				return errors.New("resolution is empty")
			}
		}
		issrRpc := getIssrRequestor()
		if issrRpc == nil {
			storeFailure(msg)
			log.Errorf("can't found issr node")
			return errors.New("can't found issr node")
		}
		_, err := issrRpc.SyncRequest(proto.BizToIssrReportStreamState, msg)
		if err != nil {
			storeFailure(msg)
			log.Errorf(err.Reason)
			return errors.New(err.Reason)
		}
	}
	return nil
}

func storeFailure(data map[string]interface{}) error {
	islb := getIslbRequestor()
	if islb == nil {
		logger.Errorf("biz.storeFailure can't find islb requestor")
		return errors.New("biz.storeFailure can't find islb requestor")
	}
	_, nerr := islb.SyncRequest(proto.IssrToIslbStoreFailedStreamState, data)
	if nerr != nil {
		logger.Errorf(fmt.Sprintf("biz.storeFailure islb rpc err=%v", nerr))
		//return errors.New(fmt.Sprintf("biz.storeFailure islb rpc err=%v", nerr))
		return fmt.Errorf("biz.storeFailure islb rpc err=%v", nerr)
	}
	return nil
}
