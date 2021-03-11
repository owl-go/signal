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

	serviceNode := server.NewServiceNode(conf.Etcd.Addrs, conf.Global.Ndc, conf.Global.Nid, conf.Global.Name, conf.Global.Nip)
	serviceNode.RegisterNode()
	serviceWatcher := server.NewServiceWatcher(conf.Etcd.Addrs)
	biz.Init(serviceNode, serviceWatcher, conf.Nats.URL)
	biz.InitSignalServer(conf.Signal.Host, conf.Signal.Port, conf.Signal.Cert, conf.Signal.Key)

	select {}
}
