package main

import (
	conf "mgkj/pkg/conf/logsvr"
	"mgkj/pkg/db"
	"mgkj/pkg/log"
	logsvr "mgkj/pkg/node/logsvr"
	"mgkj/pkg/server"
	"mgkj/pkg/util"
	"net/http"
	_ "net/http/pprof"
)

func close() {
	logsvr.Close()
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
	serviceNode := server.NewServiceNode(util.ProcessUrlString(conf.Etcd.Addrs), conf.Global.Ndc, conf.Global.Nid, conf.Global.Name, conf.Global.Nip)
	serviceNode.RegisterNode()
	serviceWatcher := server.NewServiceWatcher(util.ProcessUrlString(conf.Etcd.Addrs))
	config := db.MysqlConfig{
		Host:     conf.Mysql.Host,
		Port:     conf.Mysql.Port,
		Username: conf.Mysql.Username,
		Password: conf.Mysql.Password,
		Database: conf.Mysql.Database,
	}
	logsvr.InitLogger(conf.Global.Ndc, conf.Global.Name, conf.Global.Nid, conf.Global.Nip, conf.Log.Level,
		conf.Etcd.Addrs, conf.Nats.URL, true) //初始化logger

	logsvr.SetLoggerOutput(conf.Log.Filename, conf.Log.MaxSize, conf.Log.MaxAge, conf.Log.Maxbackups) //设置日志输出

	logsvr.Init(serviceNode, serviceWatcher, conf.Nats.URL, config)

	go logsvr.InitHttpServer(conf.LogSvr.Host, conf.LogSvr.Port, conf.LogSvr.Key, conf.LogSvr.Cert)
	select {}

}
