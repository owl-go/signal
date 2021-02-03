package main

import (
	"net/http"
	_ "net/http/pprof"

	conf "mgkj/pkg/conf/sfu"
	"mgkj/pkg/log"
	"mgkj/pkg/node/sfu"
	"mgkj/pkg/rtc"
	"mgkj/pkg/rtc/plugins"
	"mgkj/pkg/server"

	"github.com/pion/webrtc/v2"
)

func init() {
	var icePortStart, icePortEnd uint16

	if len(conf.WebRTC.ICEPortRange) == 2 {
		icePortStart = conf.WebRTC.ICEPortRange[0]
		icePortEnd = conf.WebRTC.ICEPortRange[1]
	}

	log.Init(conf.Log.Level)
	var iceServers []webrtc.ICEServer
	for _, iceServer := range conf.WebRTC.ICEServers {
		s := webrtc.ICEServer{
			URLs:       iceServer.URLs,
			Username:   iceServer.Username,
			Credential: iceServer.Credential,
		}
		iceServers = append(iceServers, s)
	}
	if err := rtc.InitIce(iceServers, icePortStart, icePortEnd); err != nil {
		panic(err)
	}

	if err := rtc.InitRTP(conf.Rtp.Port, conf.Rtp.KcpKey, conf.Rtp.KcpSalt); err != nil {
		panic(err)
	}

	pluginConfig := plugins.Config{
		On: conf.Plugins.On,
		JitterBuffer: plugins.JitterBufferConfig{
			On:            conf.Plugins.JitterBuffer.On,
			REMBCycle:     conf.Plugins.JitterBuffer.REMBCycle,
			PLICycle:      conf.Plugins.JitterBuffer.PLICycle,
			MaxBandwidth:  conf.Plugins.JitterBuffer.MaxBandwidth,
			MaxBufferTime: conf.Plugins.JitterBuffer.MaxBufferTime,
		},
	}

	if err := rtc.CheckPlugins(pluginConfig); err != nil {
		panic(err)
	}
	rtc.InitPlugins(pluginConfig)
}

func close() {
	sfu.Close()
}

func main() {
	defer close()

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			err := http.ListenAndServe(conf.Global.Pprof, nil)
			if err != nil {
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	serviceNode := server.NewServiceNode(conf.Etcd.Addrs, conf.Global.Ndc, conf.Global.Nid, conf.Global.Name, conf.Global.Nip)
	serviceNode.RegisterNode()

	serviceWatcher := server.NewServiceWatcher(conf.Etcd.Addrs)

	sfu.Init(serviceNode, serviceWatcher, conf.Amqp.URL)
	select {}
}
