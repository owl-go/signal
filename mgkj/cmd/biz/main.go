package main

import (
	"fmt"
	"net/http"

	_ "net/http/pprof"

	conf "mgkj/pkg/conf/biz"
	"mgkj/pkg/discovery"
	"mgkj/pkg/log"
	biz "mgkj/pkg/node/biz"
)

var (
	ionID = fmt.Sprintf("%s:%d", conf.Global.Addr, conf.Rtp.Port)
)

func init() {
	log.Init(conf.Log.Level)
	biz.Init(ionID, conf.Amqp.URL)
	biz.InitWebSocket(conf.Signal.Host, conf.Signal.Port, conf.Signal.Cert, conf.Signal.Key)
	discovery.Init(conf.Etcd.Addrs)
	discovery.UpdateLoad(conf.Global.Addr, conf.Rtp.Port)
}

func close() {
	biz.Close()
}

func main() {
	defer close()
	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}

	select {}
}
