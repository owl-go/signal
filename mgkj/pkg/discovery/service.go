package discovery

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"mgkj/pkg/log"
	"mgkj/pkg/util"
)

// ServiceRegistry etcd服务注册管理对象
type ServiceRegistry struct {
	Scheme string
	etcd   *Etcd
}

// Node 服务器节点信息对象
type Node struct {
	ID   string
	Info map[string]string
}

// NewServiceRegistry 创建一个etcd服务注册管理对象
func NewServiceRegistry(endpoints []string, scheme string) *ServiceRegistry {
	r := &ServiceRegistry{
		Scheme: scheme,
	}
	etcd, _ = newEtcd(endpoints)
	r.etcd = etcd
	return r
}

// RegisterServiceNode 注册一个服务节点到etcd服务管理上
func (r *ServiceRegistry) RegisterServiceNode(serviceName string, node Node) error {
	if serviceName == "" {
		return fmt.Errorf("Service name must be non empty")
	}
	if node.ID == "" {
		return fmt.Errorf("Node name must be non empty")
	}
	node.Info["ip"] = util.GetIntefaceIP()
	go r.keepRegistered(serviceName, node)
	return nil
}

// keepRegistered 注册一个服务节点到etcd服务管理上
func (r *ServiceRegistry) keepRegistered(serviceName string, node Node) {
	for {
		nodePath := r.Scheme + serviceName + "-" + node.ID
		err := r.etcd.keep(nodePath, encode(node.Info))
		if err != nil {
			log.Warnf("Registration got errors. Restarting. err=%s", err)
			time.Sleep(5 * time.Second)
		} else {
			log.Infof("Node [%s] registration success!", nodePath)
			return
		}
	}
}

// GetServiceNodes 返回活动的服务器列表
func (r *ServiceRegistry) GetServiceNodes(serviceName string) ([]Node, error) {
	rsp, err := r.etcd.GetResponseByPrefix(r.servicePath(serviceName))
	if err != nil {
		return nil, err
	}
	nodes := make([]Node, 0)
	if len(rsp.Kvs) == 0 {
		log.Debugf("No services nodes were found under %s", r.servicePath(serviceName))
		return nodes, nil
	}

	for _, n := range rsp.Kvs {
		node := Node{}
		node.ID = string(n.Key)
		node.Info = decode(n.Value)
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

// servicePath 获取指定服务器名称对应的key
func (r *ServiceRegistry) servicePath(serviceName string) string {
	service := strings.Replace(serviceName, "/", "-", -1)
	return path.Join(r.Scheme, service)
}
