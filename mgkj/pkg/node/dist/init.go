package dist

import (
	"mgkj/pkg/server"

	"mgkj/util"

	nprotoo "github.com/gearghost/nats-protoo"
)

var (
	nats  *nprotoo.NatsProtoo
	node  *server.ServiceNode
	watch *server.ServiceWatcher
	rpcs  = make(map[string]*nprotoo.Requestor)
)

// Init 初始化服务
func Init(serviceNode *server.ServiceNode, ServiceWatcher *server.ServiceWatcher, natsURL string) {
	node = serviceNode
	watch = ServiceWatcher
	nats = nprotoo.NewNatsProtoo(util.GenerateNatsUrlString(natsURL))
	go watch.WatchServiceNode("", WatchServiceCallBack)
	// 启动消息接收
	handleRPCRequest(node.GetRPCChannel())
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
func WatchServiceCallBack(state server.NodeStateType, node server.Node) {
	if state == server.ServerUp {
		if node.Name == "islb" || node.Name == "dist" {
			id := node.Nid
			_, found := rpcs[id]
			if !found {
				rpcID := server.GetRPCChannel(node)
				rpcs[id] = nats.NewRequestor(rpcID)
			}
		}
	} else if state == server.ServerDown {
		delete(rpcs, node.Nid)
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

// FindDistNodeByID 查询指定id的dist节点
func FindDistNodeByID(nid string) *server.Node {
	dist, find := watch.GetNodeByID(nid)
	if find {
		return dist
	}
	return nil
}

// FindBizNodeByPayload 查询指定区域下的可用的biz节点
func FindBizNodeByPayload() *server.Node {
	biz, find := watch.GetNodeByPayload(node.NodeInfo().Ndc, "biz")
	if find {
		return biz
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
