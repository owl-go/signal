package issr

import (
	dis "mgkj/infra/discovery"
	"mgkj/infra/kafka"
	logger2 "mgkj/infra/logger"
	"mgkj/pkg/log"
	"mgkj/util"

	nprotoo "github.com/gearghost/nats-protoo"
)

var (
	logger        *logger2.Logger
	rpcs          map[string]*nprotoo.Requestor
	protoo        *nprotoo.NatsProtoo
	kafkaClient   *kafka.KafkaClient
	kafkaProducer *kafka.SyncProducer
	node          *dis.ServiceNode
	watch         *dis.ServiceWatcher
)

// Init 初始化服务
func Init(serviceNode *dis.ServiceNode, ServiceWatcher *dis.ServiceWatcher, natsURL, kafkaURL string, l *logger2.Logger) {
	// 赋值
	logger = l
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
	//连接nats
	protoo = nprotoo.NewNatsProtoo(util.GenerateNatsUrlString(natsURL))
	// 启动MQ监听
	handleRPCRequest(node.GetRPCChannel())
	//监听islb节点
	go watch.WatchServiceNode("", WatchServiceCallBack)
}

// WatchServiceCallBack 查看所有的Node节点
func WatchServiceCallBack(state dis.NodeStateType, node dis.Node) {
	if state == dis.ServerUp {
		log.Infof("WatchServiceCallBack node up %v", node)
		if node.Name == "islb" {
			id := node.Nid
			_, found := rpcs[id]
			if !found {
				rpcID := dis.GetRPCChannel(node)
				rpcs[id] = protoo.NewRequestor(rpcID)
			}
		}
	} else if state == dis.ServerDown {
		log.Infof("WatchServiceCallBack node down %v", node.Nid)
		if _, found := rpcs[node.Nid]; found {
			delete(rpcs, node.Nid)
		}
	}
}

// findIslbNode 查询全局的可用的islb节点
func findIslbNode() *dis.Node {
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
