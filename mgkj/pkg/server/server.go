package server

import (
	"encoding/json"
	"fmt"
	"time"

	"mgkj/pkg/log"
	"mgkj/pkg/util"

	"go.etcd.io/etcd/clientv3"
)

// NodeStateType 节点状态
type NodeStateType int32

const (
	serverUp   NodeStateType = 0
	serverDown NodeStateType = 1
)

// ServiceWatchCallback 定义服务节点状态改变回调
type ServiceWatchCallback func(state NodeStateType, node Node)

// Node 服务节点对象
type Node struct {
	Nid  string
	Name string
	Nip  string
	Info string
}

// getID 返回保存节点完整的key
func (n *Node) getKey() string {
	return "/" + n.Name + "/node/" + n.Nid
}

// serviceKey 返回查询节点的前缀
func serviceKey(serviceName string) string {
	if serviceName == "" {
		return "/*/node/"
	}
	return "/" + serviceName + "/node/"
}

// GetServiceNodes 返回指定serviceName的活动的服务节点列表
func GetServiceNodes(serviceName string, etcd *Etcd) ([]Node, error) {
	rsp, err := etcd.GetResponseByPrefix(serviceKey(serviceName))
	if err != nil {
		return nil, err
	}
	nodes := make([]Node, 0)
	if len(rsp.Kvs) == 0 {
		return nodes, nil
	}

	for _, n := range rsp.Kvs {
		nodeobj := decode(n.Value)
		node := Node{}
		node.Nid = nodeobj["Nid"]
		node.Name = nodeobj["Name"]
		node.Nip = nodeobj["Nip"]
		node.Info = nodeobj["Info"]
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// encode 将map格式转换成string
func encode(m map[string]string) string {
	if m != nil {
		b, _ := json.Marshal(m)
		return string(b)
	}
	return ""
}

// decode 将string格式转换成map
func decode(ds []byte) map[string]string {
	if ds != nil && len(ds) > 0 {
		var s map[string]string
		json.Unmarshal(ds, &s)
		return s
	}
	return nil
}

// ServiceNode 服务注册对象
type ServiceNode struct {
	etcd *Etcd
	node Node
}

// NewServiceNode 新建一个服务注册对象
func NewServiceNode(endpoints []string, nid, name, info string) *ServiceNode {
	var serverNode ServiceNode
	etcd, _ := newEtcd(endpoints)
	serverNode.etcd = etcd
	serverNode.node = Node{
		Nid:  nid,
		Name: name,
		Nip:  util.GetIntefaceIP(),
		Info: info,
	}
	return &serverNode
}

// NodeInfo 返回服务节点信息
func (serverNode *ServiceNode) NodeInfo() Node {
	return serverNode.node
}

// GetEventChannel 获取广播对象string
func (serverNode *ServiceNode) GetEventChannel() string {
	return "event-" + serverNode.node.Nid
}

// GetRPCChannel 获取RPC对象string
func (serverNode *ServiceNode) GetRPCChannel() string {
	return "rpc-" + serverNode.node.Nid
}

// RegisterNode 注册服务节点
func (serverNode *ServiceNode) RegisterNode() error {
	if serverNode.node.Nid == "" || serverNode.node.Name == "" {
		return fmt.Errorf("Node id or name must be non empty")
	}
	go serverNode.keepRegistered(serverNode.node)
	return nil
}

// keepRegistered 注册一个服务节点到etcd服务管理上
func (serverNode *ServiceNode) keepRegistered(node Node) {
	for {
		nodeKey := node.getKey()
		err := serverNode.etcd.keep(nodeKey, encode(util.MapStr("Nid", node.Nid, "Name", node.Name, "Nip", node.Nip, "Info", node.Info)))
		if err != nil {
			log.Warnf("Registration got errors. Restarting. err=%s", err)
			time.Sleep(5 * time.Second)
		} else {
			log.Infof("Node [%s] registration success!", nodeKey)
			return
		}
	}
}

// ServiceWatcher 服务发现对象
type ServiceWatcher struct {
	etcd     *Etcd
	nodesMap map[string]map[string]Node // 第一级服务名,第二级服务Nid
	callback ServiceWatchCallback
}

// NewServiceWatcher 新建一个服务发现对象
func NewServiceWatcher(endpoints []string) *ServiceWatcher {
	watch, _ := newEtcd(endpoints)
	sw := &ServiceWatcher{
		nodesMap: make(map[string]map[string]Node),
		etcd:     watch,
		callback: nil,
	}
	log.Infof("New Service Watcher: etcd => %v", endpoints)
	return sw
}

// GetNodes 根据服务名称获取所有该服务节点的所有对象
func (sw *ServiceWatcher) GetNodes(serviceName string) (map[string]Node, bool) {
	nodes, found := sw.nodesMap[serviceName]
	return nodes, found
}

// GetNodesByID 根据服务节点id获取到服务节点对象
func (sw *ServiceWatcher) GetNodesByID(Nid string) (*Node, bool) {
	for _, nodes := range sw.nodesMap {
		for id, node := range nodes {
			if id == Nid {
				return &node, true
			}
		}
	}
	return nil, false
}

// DeleteNodesByID 删除指定节点id的服务节点
func (sw *ServiceWatcher) DeleteNodesByID(Nid string) bool {
	for service, nodes := range sw.nodesMap {
		for id := range nodes {
			if id == Nid {
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
					nodemap := decode(ev.Kv.Value)
					node := &Node{
						Nid:  nodemap["Nid"],
						Nip:  nodemap["Nip"],
						Name: nodemap["Name"],
						Info: nodemap["Info"],
					}

					log.Infof("Node [%s] Down", node.Nid)
					n, found := sw.GetNodesByID(node.Nid)
					if found {
						if sw.callback != nil {
							sw.callback(serverDown, *n)
						}
						sw.DeleteNodesByID(node.Nid)
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
		nodes, err := GetServiceNodes(serviceName, sw.etcd)
		if err != nil {
			log.Warnf("sw.reg.GetServiceNodes err=%v", err)
			continue
		}
		log.Debugf("Nodes: => %v", nodes)

		for _, node := range nodes {
			nid := node.Nid
			name := node.Name

			if _, found := sw.nodesMap[name]; !found {
				sw.nodesMap[name] = make(map[string]Node)
			}

			if _, found := sw.GetNodesByID(nid); !found {
				log.Infof("New %s node UP => [%s].", name, nid)
				callback(serverUp, node)

				log.Infof("Start watch for [%s] node => [%s].", name, nid)
				sw.etcd.watch(node.getKey(), sw.WatchNode, true)
				sw.nodesMap[name][nid] = node
			}
		}
		time.Sleep(2 * time.Second)
	}
}
