package main

import (
	"net/http"

	_ "net/http/pprof"

	conf "mgkj/pkg/conf/biz"
	"mgkj/pkg/log"
	biz "mgkj/pkg/node/biz"
	"mgkj/pkg/server"
)

func close() {
	biz.Close()
}

func main() {
	defer close()

	log.Init(conf.Log.Level)
	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}

	serviceNode := server.NewServiceNode(conf.Etcd.Addrs, conf.Global.ServerID, conf.Global.Name, "")
	serviceNode.RegisterNode()

	biz.Init(conf.Amqp.URL, serviceNode.GetRPCChannel(), serviceNode.GetEventChannel())
	biz.InitWebSocket(conf.Signal.Host, conf.Signal.Port, conf.Signal.Cert, conf.Signal.Key)
	select {}
}
