package logsvr

import (
	"fmt"

	dis "mgkj/infra/discovery"
	es "mgkj/infra/es"
	db "mgkj/infra/mysql"
	"mgkj/pkg/log"
	lgr "mgkj/pkg/logger"

	nprotoo "github.com/gearghost/nats-protoo"
	"github.com/gin-gonic/gin"
)

var (
	nats     *nprotoo.NatsProtoo
	rpcs     = make(map[string]*nprotoo.Requestor)
	node     *dis.ServiceNode
	watch    *dis.ServiceWatcher
	mysql    *db.MysqlDriver
	logger   *lgr.Logger
	esClient *es.EsClient
)

// Init 初始化服务
func Init(serviceNode *dis.ServiceNode, ServiceWatcher *dis.ServiceWatcher, natsURL string, config db.MysqlConfig, esUrl string, index string) {
	node = serviceNode
	watch = ServiceWatcher
	nats = nprotoo.NewNatsProtoo(natsURL)
	rpcs = make(map[string]*nprotoo.Requestor)
	mysql = db.NewMysqlDriver(config)
	esClient, _ = es.NewEsClient(esUrl)
	go watch.WatchServiceNode("", WatchServiceCallBack)
	handleRPCRequest(node.GetRPCChannel(), index)

}

func InitLogger(dc, name, nid, nip, level string, etcdUrls string, natsUrl string, addCaller bool) {
	factory := lgr.NewDefaultFactory(etcdUrls, natsUrl)
	logger = lgr.NewLogger(dc, name, nid, nip, level, addCaller, factory)
}

func SetLoggerOutput(filename string, maxsize int, maxage int, maxBackup int) {
	logger.SetOutPut(filename, maxsize, maxage, maxBackup)
}

//启动http服务器
func InitHttpServer(host string, port int, cert, key string) {
	r := gin.New()
	gin.SetMode(gin.ReleaseMode)
	Entry(r)
	r.Run(fmt.Sprintf("%s:%d", host, port))
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
	if esClient != nil {
		esClient.Stop()
	}
}

// WatchServiceCallBack 查看所有的Node节点
func WatchServiceCallBack(state dis.NodeStateType, node dis.Node) {
	if state == dis.ServerUp {
		log.Infof("WatchServiceCallBack node up %v", node)
		id := node.Nid
		_, found := rpcs[id]
		if !found {
			rpcID := dis.GetRPCChannel(node)
			rpcs[id] = nats.NewRequestor(rpcID)
		}
	} else if state == dis.ServerDown {
		log.Infof("WatchServiceCallBack node down %v", node.Nid)
		if _, found := rpcs[node.Nid]; found {
			delete(rpcs, node.Nid)
		}
	}
}

// FindIslbNode 查询全局的可用的islb节点
func FindIslbNode() *dis.Node {
	servers, find := watch.GetNodes("islb")
	if find {
		for _, node := range servers {
			return &node
		}
	}
	return nil
}
