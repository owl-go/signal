package node

import (
	"mgkj/pkg/log"
	"mgkj/pkg/mq"
	"mgkj/pkg/proto"
	"mgkj/pkg/server"
	"mgkj/pkg/util"
	"sync"
)

var (
	amqp     *mq.Amqp
	node     *server.ServiceNode
	watch    *server.ServiceWatcher
	rooms    = make(map[string]*RoomNode)
	roomLock sync.RWMutex
)

// Init 初始化服务
func Init(serviceNode *server.ServiceNode, ServiceWatcher *server.ServiceWatcher, mqURL string) {
	node = serviceNode
	watch = ServiceWatcher
	watch.WatchServiceNode("", WatchServiceNodes)
	amqp = mq.New(node.GetRPCChannel(), node.GetEventChannel(), mqURL)
	handleRPCMsgs()
	handleBroadCastMsgs()
}

// Close 关闭连接
func Close() {
	if amqp != nil {
		amqp.Close()
	}
}

// WatchServiceNodes 查看所有的Node节点
func WatchServiceNodes(state server.NodeStateType, node server.Node) {
	if state == server.ServerUp {
		log.Infof("WatchServiceNodes node up %v", node)
	} else if state == server.ServerDown {
		log.Infof("WatchServiceNodes node down %v", node)
	}
}

// FindIslbNode 查询全局的可用的islb节点
func FindIslbNode() *server.Node {
	servers, find := watch.GetNodes("islb")
	if find {
		for _, node := range servers {
			return &node
		}
	}
	return nil
}

// FindSfuNodeByID 查询指定区域下的可用的sfu节点
func FindSfuNodeByID(nid string) *server.Node {
	sfu, find := watch.GetNodeByID(nid)
	if find {
		return sfu
	}
	return nil
}

// FindSfuNodeByPayload 查询指定区域下的可用的sfu节点
func FindSfuNodeByPayload() *server.Node {
	sfu, find := watch.GetNodeByPayload(node.NodeInfo().Ndc, "sfu")
	if find {
		return sfu
	}
	return nil
}

// FindSfuNodeByMid 根据mid向islb查询指定的sfu节点
func FindSfuNodeByMid(mid string) *server.Node {
	islb := FindIslbNode()
	if islb == nil {
		log.Errorf("FindSfuNodeByMid islb not found")
		return nil
	}

	ch := make(chan int, 1)
	sfu := server.Node{}
	find := false
	respIslb := func(resp map[string]interface{}) {
		sfu.Ndc = resp["Ndc"].(string)
		sfu.Nid = resp["Nid"].(string)
		sfu.Name = resp["Name"].(string)
		sfu.Nip = resp["Nip"].(string)
		sfu.Npayload = resp["Nip"].(string)
		find = true
		ch <- 0
	}
	amqp.RPCCallWithResp(server.GetRPCChannel(*islb), util.Map("method", proto.IslbGetSfuInfo, "service", "sfu", "mid", mid), respIslb)
	<-ch
	close(ch)
	if find {
		return &sfu
	}
	return nil
}
