package main

import (
	"net/http"

	conf "mgkj/pkg/conf/islb"
	"mgkj/pkg/db"
	"mgkj/pkg/discovery"
	"mgkj/pkg/log"
	ilsb "mgkj/pkg/node/islb"
)

func main() {
	log.Init(conf.Log.Level)
	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}
	discovery.Init(conf.Etcd.Addrs)
	config := db.Config{
		Addrs: conf.Redis.Addrs,
		Pwd:   conf.Redis.Pwd,
		DB:    conf.Redis.DB,
	}
	ilsb.Init(conf.Amqp.Url, config)
	select {}
}
