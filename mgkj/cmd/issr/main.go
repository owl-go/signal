package main

import (
	"context"
	"fmt"
	"mgkj/infra/logger"
	"net/http"
	_ "net/http/pprof"
	"strconv"

	dis "mgkj/infra/discovery"
	h "mgkj/infra/http"
	conf "mgkj/pkg/conf/issr"
	"mgkj/pkg/log"
	issr "mgkj/pkg/node/issr"
	"mgkj/util"
)

func main() {

	log.Init(conf.Log.Level)

	//init logger
	factory := logger.NewDefaultFactory(conf.Etcd.Addrs, conf.Nats.NatsLog)
	l := logger.NewLogger(conf.Global.Ndc, conf.Global.Name, conf.Global.Nid, conf.Global.Nip, "info", true, factory)

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

	issr.Init(serviceNode, serviceWatcher, conf.Nats.URL, conf.Kafka.URL, l)

	l.Infof(fmt.Sprintf("issr %s start.", conf.Global.Nid))

	select {}
}

func probe(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}
