package issr

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"mgkj/infra/kafka"
	"mgkj/pkg/log"
	"mgkj/pkg/server"
)

var (
	rpcs          map[string]*nprotoo.Requestor
	protoo        *nprotoo.NatsProtoo
	kafkaClient   *kafka.KafkaClient
	kafkaProducer *kafka.SyncProducer
	node          *server.ServiceNode
	watch         *server.ServiceWatcher
)

// Init 初始化服务
func Init(serviceNode *server.ServiceNode, ServiceWatcher *server.ServiceWatcher, natsURL, kafkaURL string) {
	// 赋值
	node = serviceNode
	watch = ServiceWatcher
	rpcs = make(map[string]*nprotoo.Requestor)
	//连接kafka
	client, err := kafka.NewKafkaClient(kafkaURL)
	if err != nil {
		panic(err)
	}
	kafkaClient = client
	//创建kafka生产者
	producer, err := kafka.NewSyncProducer(kafkaClient)
	if err != nil {
		panic(err)
	}
	kafkaProducer = producer
	//连接nats-server
	protoo = nprotoo.NewNatsProtoo("nats://" + natsURL)
	// 启动MQ监听
	handleRPCRequest(node.GetRPCChannel())
	//监听islb节点
	go watch.WatchServiceNode("", WatchServiceCallBack)
}

// WatchServiceCallBack 查看所有的Node节点
func WatchServiceCallBack(state server.NodeStateType, node server.Node) {
	if state == server.ServerUp {
		log.Infof("WatchServiceCallBack node up %v", node)
		if node.Name == "islb" {
			id := node.Nid
			_, found := rpcs[id]
			if !found {
				rpcID := server.GetRPCChannel(node)
				rpcs[id] = protoo.NewRequestor(rpcID)
			}
		}
	} else if state == server.ServerDown {
		log.Infof("WatchServiceCallBack node down %v", node.Nid)
		if _, found := rpcs[node.Nid]; found {
			delete(rpcs, node.Nid)
		}
	}
}

// findIslbNode 查询全局的可用的islb节点
func findIslbNode() *server.Node {
	servers, find := watch.GetNodes("islb")
	if find {
		for _, node := range servers {
			return &node
		}
	}
	return nil
}

func getIslbRequestor() *nprotoo.Requestor {
	islb := findIslbNode()

	if islb == nil {
		log.Errorf("find islb node not found")
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

// Close 关闭连接
func Close() {
	if protoo != nil {
		protoo.Close()
	}
	if node != nil {
		node.Close()
	}
	if watch != nil {
		watch.Close()
	}
	if kafkaProducer != nil {
		kafkaProducer.Close()
	}
	if kafkaClient != nil {
		kafkaClient.Close()
	}
}
