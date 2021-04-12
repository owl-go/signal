package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"strconv"

	h "mgkj/infra/http"
	conf "mgkj/pkg/conf/sfu"
	"mgkj/pkg/log"
	lgr "mgkj/pkg/logger"
	"mgkj/pkg/node/sfu"
	"mgkj/pkg/server"
	"mgkj/pkg/util"
)

func close() {
	sfu.Close()
}

func main() {

	defer close()

	//init logger
	factory := lgr.NewDefaultFactory(conf.Etcd.Addrs, conf.Nats.NatsLog)
	l := lgr.NewLogger(conf.Global.Ndc, conf.Global.Name, conf.Global.Nid, conf.Global.Nip, "info", true, factory)

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			err := http.ListenAndServe(conf.Global.Pprof, nil)
			if err != nil {
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	httpserver := h.Http{}
	httpserver.Init(conf.Probe.Host, strconv.Itoa(conf.Probe.Port))
	g := httpserver.Group("/api/v1", nil, nil)
	g.Post("/probe", probe, nil)

	serviceNode := server.NewServiceNode(util.ProcessUrlString(conf.Etcd.Addrs), conf.Global.Ndc, conf.Global.Nid, conf.Global.Name, conf.Global.Nip)
	serviceNode.RegisterNode()
	serviceWatcher := server.NewServiceWatcher(util.ProcessUrlString(conf.Etcd.Addrs))
	sfu.Init(serviceNode, serviceWatcher, conf.Nats.URL, l)

	l.Infof(fmt.Sprintf("sfu %s start.", conf.Global.Nid))

	select {}
}

func probe(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}
