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
	db "signal/infra/redis"
	conf "signal/pkg/conf/islb"
	"signal/pkg/log"
	islb "signal/pkg/node/islb"
	"signal/util"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func close() {
	islb.Close()
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

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if http.ListenAndServe(":"+strconv.Itoa(conf.Monitor.Port), nil) == nil {
			log.Errorf("start prometheus service fail")
		}
	}()

	serviceNode := dis.NewServiceNode(util.ProcessUrlString(conf.Etcd.Addrs), conf.Global.Ndc, conf.Global.Nid, conf.Global.Name, conf.Global.Nip)
	serviceNode.RegisterNode()
	serviceWatcher := dis.NewServiceWatcher(util.ProcessUrlString(conf.Etcd.Addrs))
	config := db.Config{
		Addrs: conf.Redis.Addrs,
		Pwd:   conf.Redis.Pwd,
		DB:    conf.Redis.DB,
	}

	islb.Init(serviceNode, serviceWatcher, conf.Nats.URL, config, l)

	l.Infof(fmt.Sprintf("islb %s start.", conf.Global.Nid))

	select {}
}

func probe(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}
