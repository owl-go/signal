package discovery

import (
	"math/rand"
	"time"

	"mgkj/pkg/log"

	"go.etcd.io/etcd/clientv3"
)

// ServiceNode 管理服务器节点和etcd客户端
type ServiceNode struct {
	reg  *ServiceRegistry
	name string
	node Node
}

// NodeStateType 节点状态
type NodeStateType int32

const (
	serverUp   NodeStateType = 0
	serverDown NodeStateType = 1
)

// ServiceWatchCallback 定义服务节点状态改变回调
type ServiceWatchCallback func(service string, state NodeStateType, nodes Node)

// NewServiceNode 新建一个服务节点和etcd客户端对象
func NewServiceNode(endpoints []string, dc string) *ServiceNode {
	var sn ServiceNode
	sn.reg = NewServiceRegistry(endpoints, "/"+dc+"/node/")
	log.Infof("New Service Node Registry: etcd => %v", endpoints)
	sn.node = Node{
		Info: make(map[string]string),
	}
	return &sn
}

// NodeInfo 返回服务节点信息
func (sn *ServiceNode) NodeInfo() Node {
	return sn.node
}

// GetEventChannel 获取广播对象string
func (sn *ServiceNode) GetEventChannel() string {
	return "event-" + sn.node.Info["id"]
}

// GetRPCChannel 获取RPC对象string
func (sn *ServiceNode) GetRPCChannel() string {
	return "rpc-" + sn.node.Info["id"]
}

// RegisterNode 注册一个新的服务节点
func (sn *ServiceNode) RegisterNode(serviceName string, name string, ID string) {
	sn.node.ID = randomString(12)
	sn.node.Info["name"] = name
	sn.node.Info["service"] = serviceName
	sn.node.Info["id"] = serviceName + "-" + sn.node.ID
	err := sn.reg.RegisterServiceNode(serviceName, sn.node)
	if err != nil {
		log.Panicf("%v", err)
	}
}

// ServiceWatcher 服务发现对象
type ServiceWatcher struct {
	reg      *ServiceRegistry
	nodesMap map[string]map[string]Node
	callback ServiceWatchCallback
}

// NewServiceWatcher 新建一个服务发现对象
func NewServiceWatcher(endpoints []string, dc string) *ServiceWatcher {
	sw := &ServiceWatcher{
		nodesMap: make(map[string]map[string]Node),
		reg:      NewServiceRegistry(endpoints, "/"+dc+"/node/"),
		callback: nil,
	}
	log.Infof("New Service Watcher: etcd => %v", endpoints)
	return sw
}

// GetNodes 根据服务名称获取所有该服务节点的所有对象
func (sw *ServiceWatcher) GetNodes(service string) (map[string]Node, bool) {
	nodes, found := sw.nodesMap[service]
	return nodes, found
}

// GetNodesByID 根据服务节点id获取到服务节点对象
func (sw *ServiceWatcher) GetNodesByID(ID string) (*Node, bool) {
	for _, nodes := range sw.nodesMap {
		for id, node := range nodes {
			if id == ID {
				return &node, true
			}
		}
	}
	return nil, false
}

// DeleteNodesByID 删除指定节点id的服务节点
func (sw *ServiceWatcher) DeleteNodesByID(ID string) bool {
	for service, nodes := range sw.nodesMap {
		for id := range nodes {
			if id == ID {
				delete(sw.nodesMap[service], id)
				return true
			}
		}
	}
	return false
}

// WatchNode 监控到服务节点状态改变
func (sw *ServiceWatcher) WatchNode(ch clientv3.WatchChan) {
	go func() {
		for {
			msg := <-ch
			for _, ev := range msg.Events {
				log.Infof("%s %q:%q", ev.Type, ev.Kv.Key, ev.Kv.Value)
				if ev.Type == clientv3.EventTypeDelete {
					nodeID := string(ev.Kv.Key)
					log.Infof("Node [%s] Down", nodeID)
					n, found := sw.GetNodesByID(nodeID)
					if found {
						service := n.Info["service"]
						if sw.callback != nil {
							sw.callback(service, serverDown, *n)
						}
						sw.DeleteNodesByID(nodeID)
					}
				}
			}
		}
	}()
}

// WatchServiceNode 监控指定服务名称的所有服务节点的状态
func (sw *ServiceWatcher) WatchServiceNode(serviceName string, callback ServiceWatchCallback) {
	log.Infof("Start service watcher => [%s].", serviceName)
	sw.callback = callback
	for {
		nodes, err := sw.reg.GetServiceNodes(serviceName)
		if err != nil {
			log.Warnf("sw.reg.GetServiceNodes err=%v", err)
			continue
		}
		log.Debugf("Nodes: => %v", nodes)

		for _, node := range nodes {
			id := node.ID
			service := node.Info["service"]

			if _, found := sw.nodesMap[service]; !found {
				sw.nodesMap[service] = make(map[string]Node)
			}

			if _, found := sw.GetNodesByID(node.ID); !found {
				log.Infof("New %s node UP => [%s].", service, node.ID)
				callback(service, serverUp, node)

				log.Infof("Start watch for [%s] node => [%s].", service, node.ID)
				Watch(node.ID, sw.WatchNode, true)
				sw.nodesMap[service][id] = node
			}
		}
		time.Sleep(2 * time.Second)
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

// 生成服务节点id
func randomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// GetEventChannel 同上面
func GetEventChannel(node Node) string {
	return "event-" + node.Info["id"]
}

// GetRPCChannel 同上面
func GetRPCChannel(node Node) string {
	return "rpc-" + node.Info["id"]
}
