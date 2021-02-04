package main

import (
	"net/http"
	"os"
	"os/signal"

	conf "mgkj/pkg/conf/islb"
	"mgkj/pkg/db"
	"mgkj/pkg/log"
	ilsb "mgkj/pkg/node/islb"
	"mgkj/pkg/server"
)

func close() {
	ilsb.Close()
}

func main() {
	defer close()

	ch := make(chan os.Signal)
	signal.Notify(ch)

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
	config := db.Config{
		Addrs: conf.Redis.Addrs,
		Pwd:   conf.Redis.Pwd,
		DB:    conf.Redis.DB,
	}
	ilsb.Init(serviceNode, serviceWatcher, conf.Amqp.URL, config)

	sig := <-ch
	log.Infof("islb exit %v", sig)
}
