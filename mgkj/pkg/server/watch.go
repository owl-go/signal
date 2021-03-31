package server

import (
	"mgkj/pkg/log"
	"strconv"
	"time"

	"go.etcd.io/etcd/clientv3"
)

// ServiceWatchCallback 定义服务节点状态改变回调
type ServiceWatchCallback func(state NodeStateType, node Node)

// ServiceWatcher 服务发现对象
type ServiceWatcher struct {
	etcd     *Etcd
	nodesMap map[string]map[string]Node // 第一级服务名,第二级服务Nid
	callback ServiceWatchCallback
}

// NewServiceWatcher 新建一个服务发现对象
func NewServiceWatcher(endpoints []string) *ServiceWatcher {
	watch, _ := NewEtcd(endpoints)
	serviceWatcher := &ServiceWatcher{
		nodesMap: make(map[string]map[string]Node),
		etcd:     watch,
		callback: nil,
	}
	log.Infof("New Service Watcher: etcd => %v", endpoints)
	return serviceWatcher
}

// Close 关闭资源
func (serviceWatcher *ServiceWatcher) Close() {
	if serviceWatcher.etcd != nil {
		serviceWatcher.etcd.Close()
	}
}

// GetNodes 根据服务名称获取所有该服务节点的所有对象
func (serviceWatcher *ServiceWatcher) GetNodes(serviceName string) (map[string]Node, bool) {
	nodes, found := serviceWatcher.nodesMap[serviceName]
	return nodes, found
}

// GetNodeByID 根据服务节点id获取到服务节点对象
func (serviceWatcher *ServiceWatcher) GetNodeByID(nid string) (*Node, bool) {
	for _, nodes := range serviceWatcher.nodesMap {
		for id, node := range nodes {
			if id == nid {
				return &node, true
			}
		}
	}
	return nil, false
}

// GetNodeByPayload 获取指定区域内指定服务节点负载最低的节点
func (serviceWatcher *ServiceWatcher) GetNodeByPayload(dc, name string) (*Node, bool) {
	log.Infof("GetNodeByPayload dc => %s,name => %s", dc, name)
	var nodeObj *Node = nil
	var payload int = 0
	for _, nodes := range serviceWatcher.nodesMap {
		for _, node := range nodes {
			if node.Name == name && node.Ndc == dc {
				pay, _ := strconv.Atoi(node.Npayload)
				if pay >= payload {
					nodeObj = &node
					payload = pay
				}
			}
		}
	}
	if nodeObj == nil {
		return nil, false
	}
	log.Infof("GetNodeByPayload find node => %v", nodeObj)
	return nodeObj, true
}

// DeleteNodesByID 删除指定节点id的服务节点
func (serviceWatcher *ServiceWatcher) DeleteNodesByID(nid string) bool {
	for service, nodes := range serviceWatcher.nodesMap {
		for id := range nodes {
			if id == nid {
				delete(serviceWatcher.nodesMap[service], id)
				return true
			}
		}
	}
	return false
}

// WatchNode 监控到服务节点状态改变
func (serviceWatcher *ServiceWatcher) WatchNode(ch clientv3.WatchChan) {
	go func() {
		for {
			msg := <-ch
			for _, ev := range msg.Events {
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
	for {
		nodes, err := serviceWatcher.GetServiceNodes(prefix)
		if err != nil {
			log.Errorf("serviceWatcher.GetServiceNodes err=%v", err)
			continue
		}

		for _, node := range nodes {
			nid := node.Nid
			name := node.Name

			if _, found := serviceWatcher.nodesMap[name]; !found {
				serviceWatcher.nodesMap[name] = make(map[string]Node)
			}

			if _, found := serviceWatcher.GetNodeByID(nid); !found {
				log.Infof("New %s node UP => [%s]", name, nid)
				callback(ServerUp, node)
				serviceWatcher.etcd.Watch(node.Nid, serviceWatcher.WatchNode, false)
				serviceWatcher.nodesMap[name][nid] = node
			}
		}
		time.Sleep(5 * time.Second)
	}
}

// serviceKey 返回查询节点的前缀
func serviceKey(prefix string) string {
	if prefix == "" {
		return prefix
	}
	return "/" + prefix
}

// GetServiceNodes 返回指定前缀的的活动的服务节点列表
func (serviceWatcher *ServiceWatcher) GetServiceNodes(prefix string) ([]Node, error) {
	rsp, err := serviceWatcher.etcd.GetResponseByPrefix(serviceKey(prefix))
	if err != nil {
		return nil, err
	}
	nodes := make([]Node, 0)
	if len(rsp.Kvs) == 0 {
		return nodes, nil
	}

	for _, val := range rsp.Kvs {
		nodeobj := Decode(val.Value)
		if nodeobj["Nid"] != "" {
			node := Node{}
			node.Ndc = nodeobj["Ndc"]
			node.Nid = nodeobj["Nid"]
			node.Name = nodeobj["Name"]
			node.Nip = nodeobj["Nip"]
			node.Npayload = nodeobj["Npayload"]
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}
