package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"strconv"

	h "mgkj/infra/http"
	conf "mgkj/pkg/conf/issr"
	"mgkj/pkg/log"
	issr "mgkj/pkg/node/issr"
	"mgkj/pkg/server"
	"mgkj/pkg/util"
)

func main() {

	log.Init(conf.Log.Level)
	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}

	httpserver := h.Http{}
	httpserver.Init(conf.Probe.Host, strconv.Itoa(conf.Probe.Port))
	g := httpserver.Group("/api/v1", nil, nil)
	g.Post("/probe", probe, nil)

	serviceNode := server.NewServiceNode(util.ProcessUrlString(conf.Etcd.Addrs), conf.Global.Ndc, conf.Global.Nid, conf.Global.Name, conf.Global.Nip)
	serviceNode.RegisterNode()
	serviceWatcher := server.NewServiceWatcher(util.ProcessUrlString(conf.Etcd.Addrs))

	issr.Init(serviceNode, serviceWatcher, conf.Nats.URL, conf.Kafka.URL)

	select {}
}

func probe(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}
