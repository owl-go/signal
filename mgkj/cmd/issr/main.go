package main

import (
	"net/http"
	_ "net/http/pprof"

	conf "mgkj/pkg/conf/issr"
	"mgkj/pkg/log"
	issr "mgkj/pkg/node/issr"
	"mgkj/pkg/server"
)

func main() {

	log.Init(conf.Log.Level)
	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}

	serviceNode := server.NewServiceNode(conf.Etcd.Addrs, conf.Global.Ndc, conf.Global.Nid, conf.Global.Name, conf.Global.Nip)
	serviceNode.RegisterNode()
	serviceWatcher := server.NewServiceWatcher(conf.Etcd.Addrs)

	issr.Init(serviceNode, serviceWatcher, conf.Nats.URL, conf.Kafka.URL)

	select {}
}
