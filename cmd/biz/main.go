package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"signal/infra/logger"
	"strconv"

	dis "signal/infra/discovery"
	h "signal/infra/http"
	conf "signal/pkg/conf/biz"
	"signal/pkg/log"
	biz "signal/pkg/node/biz"
	"signal/util"
)

func close() {
	biz.Close()
}

func main() {
	defer close()

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
	biz.Init(serviceNode, serviceWatcher, conf.Nats.URL, l)
	biz.InitSignalServer(conf.Signal.Host, conf.Signal.Port, conf.Signal.Cert, conf.Signal.Key)

	l.Infof(fmt.Sprintf("biz %s start.", conf.Global.Nid))

	select {}
}

func probe(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}
