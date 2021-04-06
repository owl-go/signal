package server

import (
	"mgkj/pkg/log"
	"strconv"
	"sync"

	"go.etcd.io/etcd/clientv3"
)

// ServiceWatchCallback 定义服务节点状态改变回调
type ServiceWatchCallback func(state NodeStateType, node Node)

// ServiceWatcher 服务发现对象
type ServiceWatcher struct {
	etcd     *Etcd
	bStop    bool
	nodes    map[string]Node
	nodeLook sync.Mutex
	callback ServiceWatchCallback
}

// NewServiceWatcher 新建一个服务发现对象
func NewServiceWatcher(endpoints []string) *ServiceWatcher {
	watch, _ := NewEtcd(endpoints)
	serviceWatcher := &ServiceWatcher{
		bStop:    false,
		nodes:    make(map[string]Node),
		etcd:     watch,
		callback: nil,
	}
	log.Infof("New Service Watcher: etcd => %v", endpoints)
	return serviceWatcher
}

// Close 关闭资源
func (serviceWatcher *ServiceWatcher) Close() {
	serviceWatcher.bStop = true
	if serviceWatcher.etcd != nil {
		serviceWatcher.etcd.Close()
	}
}

// GetNodes 根据服务名称获取所有该服务节点的所有对象
func (serviceWatcher *ServiceWatcher) GetNodes(serviceName string) (map[string]Node, bool) {
	serviceWatcher.nodeLook.Lock()
	defer serviceWatcher.nodeLook.Unlock()
	mapNodes := make(map[string]Node)
	for _, node := range serviceWatcher.nodes {
		if node.Name == serviceName {
			mapNodes[node.Nid] = node
		}
	}
	if len(mapNodes) > 0 {
		return mapNodes, true
	}
	return mapNodes, false
}

// GetNodeByID 根据服务节点id获取到服务节点对象
func (serviceWatcher *ServiceWatcher) GetNodeByID(nid string) (*Node, bool) {
	serviceWatcher.nodeLook.Lock()
	defer serviceWatcher.nodeLook.Unlock()
	node, find := serviceWatcher.nodes[nid]
	if find {
		return &node, true
	}
	return nil, false
}

// GetNodeByPayload 获取指定区域内指定服务节点负载最低的节点
func (serviceWatcher *ServiceWatcher) GetNodeByPayload(dc, name string) (*Node, bool) {
	var tempObj Node
	var nodeObj *Node = nil
	var payload int = 0
	serviceWatcher.nodeLook.Lock()
	defer serviceWatcher.nodeLook.Unlock()
	for _, node := range serviceWatcher.nodes {
		if node.Ndc == dc && node.Name == name {
			pay, _ := strconv.Atoi(node.Npayload)
			if pay >= payload {
				tempObj = node
				nodeObj = &tempObj
				payload = pay
			}
		}
	}
	if nodeObj == nil {
		return nil, false
	}
	return nodeObj, true
}

// DeleteNodesByID 删除指定节点id的服务节点
func (serviceWatcher *ServiceWatcher) DeleteNodesByID(nid string) bool {
	serviceWatcher.nodeLook.Lock()
	defer serviceWatcher.nodeLook.Unlock()
	_, find := serviceWatcher.nodes[nid]
	if find {
		delete(serviceWatcher.nodes, nid)
	}
	return true
}

// WatchNode 监控到服务节点状态改变
func (serviceWatcher *ServiceWatcher) WatchNode(ch clientv3.WatchChan) {
	go func() {
		for {
			if serviceWatcher.bStop {
				return
			}
			msg := <-ch
			for _, ev := range msg.Events {
				if ev.Type == clientv3.EventTypePut {
					nid := string(ev.Kv.Key)
					nodeObj := Decode(ev.Kv.Value)
					if nodeObj["Nid"] != "" && nodeObj["Nid"] == nid {
						node := Node{}
						node.Ndc = nodeObj["Ndc"]
						node.Nid = nodeObj["Nid"]
						node.Name = nodeObj["Name"]
						node.Nip = nodeObj["Nip"]
						node.Npayload = nodeObj["Npayload"]
						serviceWatcher.nodeLook.Lock()
						serviceWatcher.nodes[nid] = node
						serviceWatcher.nodeLook.Unlock()
						if serviceWatcher.callback != nil {
							serviceWatcher.callback(ServerUp, node)
						}
					}
				}
				if ev.Type == clientv3.EventTypeDelete {
					nid := string(ev.Kv.Key)
					node, find := serviceWatcher.GetNodeByID(nid)
					if find {
						log.Infof("Node [%s] Down", nid)
						if serviceWatcher.callback != nil {
							serviceWatcher.callback(ServerDown, *node)
						}
						serviceWatcher.DeleteNodesByID(nid)
					}
				}
			}
		}
	}()
}

// WatchServiceNode 监控指定服务名称的所有服务节点的状态
func (serviceWatcher *ServiceWatcher) WatchServiceNode(prefix string, callback ServiceWatchCallback) {
	serviceWatcher.callback = callback
	serviceWatcher.etcd.Watch(prefix, serviceWatcher.WatchNode, true)
}
