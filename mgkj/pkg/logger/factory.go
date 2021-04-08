package logger

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"mgkj/pkg/proto"
	"mgkj/pkg/util"

	nprotoo "github.com/gearghost/nats-protoo"
	"go.etcd.io/etcd/clientv3"
)

type Factory interface {
	OutPut(msg string)
}

var logsvr = "logsvr"

type DefaultFactory struct {
	etcdClients *clientv3.Client
	nats        *nprotoo.NatsProtoo
	logSvrRpcs  map[string]*nprotoo.Requestor
	rpcLock     sync.RWMutex
}

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

func NewDefaultFactory(etcdUrls string, natsURL string) *DefaultFactory {
	s := new(DefaultFactory)
	conf := clientv3.Config{
		Endpoints:   util.ProcessUrlString(etcdUrls),
		DialTimeout: 5 * time.Second,
	}

	client, err := clientv3.New(conf)
	if err != nil {
		log.Fatal(err)
	}
	s.etcdClients = client
	s.logSvrRpcs = make(map[string]*nprotoo.Requestor)
	s.nats = nprotoo.NewNatsProtoo(util.GenerateNatsUrlString(natsURL))

	//获取所有节点
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := client.Get(ctx, "", clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	if err != nil {
		cancel()
		log.Fatal(err)
	}
	cancel()
	//fmt.Println("resp:", resp)

	for _, val := range resp.Kvs {
		nodeobj := Decode(val.Value)
		if nodeobj["Nid"] != "" {
			node := Node{}
			node.Ndc = nodeobj["Ndc"]
			node.Nid = nodeobj["Nid"]
			node.Name = nodeobj["Name"]
			node.Nip = nodeobj["Nip"]
			node.Npayload = nodeobj["Npayload"]
			if node.Name == logsvr {
				rpcID := GetRPCChannel(node)
				s.logSvrRpcs[node.Nid] = s.nats.NewRequestor(rpcID)
			}
		}
	}
	go s.watcher("")
	return s
}

func (s *DefaultFactory) watcher(prefix string) {
	ch := s.etcdClients.Watch(context.Background(), prefix, clientv3.WithPrefix())
	for msg := range ch {
		for _, ev := range msg.Events {
			nid := string(ev.Kv.Key)
			switch ev.Type {
			case clientv3.EventTypeDelete:
				s.delRpcs(nid)
			case clientv3.EventTypePut:
				nodeobj := Decode(ev.Kv.Value)
				if nodeobj["Nid"] != "" {
					node := Node{}
					node.Ndc = nodeobj["Ndc"]
					node.Nid = nodeobj["Nid"]
					node.Name = nodeobj["Name"]
					node.Nip = nodeobj["Nip"]
					node.Npayload = nodeobj["Npayload"]
					if node.Name == logsvr {
						s.addRpcs(node)
					}
				}
			}
		}
	}
}

func (s *DefaultFactory) delRpcs(nid string) {
	s.rpcLock.Lock()
	defer s.rpcLock.Unlock()
	if _, ok := s.logSvrRpcs[nid]; ok {
		delete(s.logSvrRpcs, nid)
	}
}

func (s *DefaultFactory) addRpcs(node Node) {
	s.rpcLock.Lock()
	defer s.rpcLock.Unlock()
	if _, ok := s.logSvrRpcs[node.Nid]; !ok {
		rpcID := GetRPCChannel(node)
		s.logSvrRpcs[node.Nid] = s.nats.NewRequestor(rpcID)
	}
}

func (s *DefaultFactory) getLogsvrRpc(nid string) *nprotoo.Requestor {
	s.rpcLock.RLock()
	defer s.rpcLock.RUnlock()
	if _, ok := s.logSvrRpcs[nid]; !ok {
		return nil
	}
	return s.logSvrRpcs[nid]
}

func (s *DefaultFactory) getLogsvrNode() *nprotoo.Requestor {
	s.rpcLock.RLock()
	defer s.rpcLock.RUnlock()

	for k, _ := range s.logSvrRpcs {
		return s.logSvrRpcs[k]
	}
	return nil
}

func (s *DefaultFactory) OutPut(msg string) {
	nprotoo := s.getLogsvrNode()
	if nprotoo != nil {
		nprotoo.AsyncRequest(proto.ToLogsvr, util.Map("data", msg))
	}
}

// GetRPCChannel 获取RPC对象string
func GetRPCChannel(node Node) string {
	return "rpc-" + node.Nid
}

// Decode 将string格式转换成map
func Decode(str []byte) map[string]string {
	if str != nil && len(str) > 0 {
		var data map[string]string
		json.Unmarshal(str, &data)
		return data
	}
	return nil
}
