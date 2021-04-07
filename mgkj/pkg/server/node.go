package server

import (
	"encoding/json"
	"mgkj/pkg/util"
)

// NodeStateType 节点状态
type NodeStateType int32

const (
	// ServerUp 服务存活
	ServerUp NodeStateType = 0
	// ServerDown 服务死亡
	ServerDown NodeStateType = 1
)

// Node 服务节点对象
type Node struct {
	// Ndc 节点区域
	Ndc string
	// Nid 节点id
	Nid string
	// Name 节点名
	Name string
	// Nip 节点Ip
	Nip string
	// 节点负载
	Npayload string
}

// GetPrefixByDc 返回保存节点的key前缀
func (node *Node) GetPrefixByDc() string {
	return "/" + node.Ndc
}

// GetPrefixByName 返回保存节点的key前缀
func (node *Node) GetPrefixByName() string {
	return "/" + node.Ndc + "/Name/" + node.Name
}

// GetPrefixByNid 返回保存节点的key前缀
func (node *Node) GetPrefixByNid() string {
	return "/" + node.Ndc + "/Name/" + node.Name + "/Node/" + node.Nid
}

// GetNodeValue 获取节点保存的值
func (node *Node) GetNodeValue() string {
	return Encode(util.Map2("Ndc", node.Ndc, "Nid", node.Nid, "Name", node.Name, "Nip", node.Nip, "Npayload", node.Npayload))
}

// Encode 将map格式转换成string
func Encode(data map[string]string) string {
	if data != nil {
		str, _ := json.Marshal(data)
		return string(str)
	}
	return ""
}

// Decode 将string格式转换成map
func Decode(str []byte) map[string]string {
	if len(str) > 0 {
		var data map[string]string
		json.Unmarshal(str, &data)
		return data
	}
	return nil
}

// GetEventChannel 获取广播对象string
func GetEventChannel(node Node) string {
	return "event-" + node.Nid
}

// GetRPCChannel 获取RPC对象string
func GetRPCChannel(node Node) string {
	return "rpc-" + node.Nid
}
