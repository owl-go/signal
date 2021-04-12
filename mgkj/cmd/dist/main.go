package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"strconv"

	dis "mgkj/infra/discovery"
	h "mgkj/infra/http"
	conf "mgkj/pkg/conf/dist"
	"mgkj/pkg/log"
	"mgkj/pkg/node/dist"
	"mgkj/util"
)

func close() {
	dist.Close()
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

	httpserver := h.Http{}
	httpserver.Init(conf.Probe.Host, strconv.Itoa(conf.Probe.Port))
	g := httpserver.Group("/api/v1", nil, nil)
	g.Post("/probe", probe, nil)

	serviceNode := dis.NewServiceNode(util.ProcessUrlString(conf.Etcd.Addrs), conf.Global.Ndc, conf.Global.Nid, conf.Global.Name, conf.Global.Nip)
	serviceNode.RegisterNode()
	serviceWatcher := dis.NewServiceWatcher(util.ProcessUrlString(conf.Etcd.Addrs))
	dist.Init(serviceNode, serviceWatcher, conf.Nats.URL)
	dist.InitCallServer(conf.Signal.Host, conf.Signal.Port, conf.Signal.Cert, conf.Signal.Key)

	select {}
}

func probe(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}
