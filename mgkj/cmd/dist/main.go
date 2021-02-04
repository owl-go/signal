package main

import (
	"net/http"
	"os"
	"os/signal"

	_ "net/http/pprof"

	conf "mgkj/pkg/conf/dist"
	"mgkj/pkg/log"
	dist "mgkj/pkg/node/dist"
	"mgkj/pkg/server"
)

func close() {
	dist.Close()
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
	dist.Init(serviceNode, serviceWatcher, conf.Amqp.URL)
	dist.InitWebSocket(conf.Signal.Host, conf.Signal.Port, conf.Signal.Cert, conf.Signal.Key)

	sig := <-ch
	log.Infof("dist exit %v", sig)
}
