package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"strconv"

	h "mgkj/infra/http"
	conf "mgkj/pkg/conf/islb"
	"mgkj/pkg/db"
	"mgkj/pkg/log"
	lgr "mgkj/pkg/logger"
	islb "mgkj/pkg/node/islb"
	"mgkj/pkg/server"
	"mgkj/pkg/util"
)

func close() {
	islb.Close()
}

func main() {
	defer close()

	log.Init(conf.Log.Level)

	//init logger
	factory := lgr.NewDefaultFactory(conf.Etcd.Addrs, conf.Nats.NatsLog)
	l := lgr.NewLogger(conf.Global.Ndc, conf.Global.Name, conf.Global.Nid, conf.Global.Nip, "info", true, factory)

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
	config := db.Config{
		Addrs: conf.Redis.Addrs,
		Pwd:   conf.Redis.Pwd,
		DB:    conf.Redis.DB,
	}
	config1 := db.Config{
		Addrs: conf.Redis.Addrs,
		Pwd:   conf.Redis.Pwd,
		DB:    conf.Redis.TDB,
	}

	islb.Init(serviceNode, serviceWatcher, conf.Nats.URL, config, config1, l)

	l.Infof(fmt.Sprintf("islb %s start.", conf.Global.Nid))

	select {}
}

func probe(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}
