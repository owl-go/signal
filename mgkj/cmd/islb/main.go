package main

import (
	"net/http"
	_ "net/http/pprof"

	conf "mgkj/pkg/conf/islb"
	"mgkj/pkg/db"
	"mgkj/pkg/log"
	islb "mgkj/pkg/node/islb"
	"mgkj/pkg/server"
)

func close() {
	islb.Close()
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

	islb.Init(serviceNode, serviceWatcher, conf.Nats.URL, config, config1)

	select {}
}
