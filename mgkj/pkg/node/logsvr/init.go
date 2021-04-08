package logsvr

import (
	"fmt"
	"mgkj/pkg/db"
	"mgkj/pkg/log"
	lgr "mgkj/pkg/logger"
	"mgkj/pkg/server"

	nprotoo "github.com/gearghost/nats-protoo"
	"github.com/gin-gonic/gin"
)

var (
	nats   *nprotoo.NatsProtoo
	rpcs   = make(map[string]*nprotoo.Requestor)
	node   *server.ServiceNode
	watch  *server.ServiceWatcher
	mysql  *db.MysqlDriver
	logger *lgr.Logger
)

// Init 初始化服务
func Init(serviceNode *server.ServiceNode, ServiceWatcher *server.ServiceWatcher, natsURL string, config db.MysqlConfig) {
	node = serviceNode
	watch = ServiceWatcher
	nats = nprotoo.NewNatsProtoo(natsURL)
	rpcs = make(map[string]*nprotoo.Requestor)
	mysql = db.NewMysqlDriver(config)
	go watch.WatchServiceNode("", WatchServiceCallBack)
	handleRPCRequest(node.GetRPCChannel())

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
}

// WatchServiceCallBack 查看所有的Node节点
func WatchServiceCallBack(state server.NodeStateType, node server.Node) {
	if state == server.ServerUp {
		log.Infof("WatchServiceCallBack node up %v", node)
		id := node.Nid
		_, found := rpcs[id]
		if !found {
			rpcID := server.GetRPCChannel(node)
			rpcs[id] = nats.NewRequestor(rpcID)
		}
	} else if state == server.ServerDown {
		log.Infof("WatchServiceCallBack node down %v", node.Nid)
		if _, found := rpcs[node.Nid]; found {
			delete(rpcs, node.Nid)
		}
	}
}

// FindIslbNode 查询全局的可用的islb节点
func FindIslbNode() *server.Node {
	servers, find := watch.GetNodes("islb")
	if find {
		for _, node := range servers {
			return &node
		}
	}
	return nil
}
