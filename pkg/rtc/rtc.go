package rtc

import (
	"sync"
	"time"

	conf "signal/pkg/conf/sfu"
	"signal/pkg/log"
	"signal/pkg/rtc/plugins"
	"signal/pkg/rtc/transport"

	"github.com/pion/webrtc/v2"
)

// InitPlugins plugins config
func InitPlugins(config plugins.Config) {
	pluginsConfig = config
}

const (
	statCycle    = 3 * time.Second
	maxCleanSize = 100
)

var (
	stop          bool
	routers       = make(map[string]*Router)
	routerLock    sync.RWMutex
	CleanPub      = make(chan string, maxCleanSize)
	pluginsConfig plugins.Config
)

// InitSfu 启动sfu
func InitSfu() {
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
	if err := InitIce(iceServers, icePortStart, icePortEnd); err != nil {
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

	if err := CheckPlugins(pluginConfig); err != nil {
		panic(err)
	}

	InitPlugins(pluginConfig)
	go CheckRoute()
}

// FreeSfu 关闭sfu
func FreeSfu() {
	stop = true
	routerLock.Lock()
	defer routerLock.Unlock()
	for id, router := range routers {
		if router != nil {
			router.Close()
			delete(routers, id)
		}
	}
}

// InitIce ice urls
func InitIce(iceServers []webrtc.ICEServer, icePortStart, icePortEnd uint16) error {
	return transport.InitWebRTC(iceServers, icePortStart, icePortEnd)
}

// CheckPlugins plugins config
func CheckPlugins(config plugins.Config) error {
	return plugins.CheckPlugins(config)
}

// GetOrNewRouter 获取router
func GetOrNewRouter(id string) *Router {
	log.Infof("rtc.GetOrNewRouter id=%s", id)
	router := GetRouter(id)
	if router == nil {
		return AddRouter(id)
	}
	return router
}

// GetRouters 获取所有的Router
func GetRouters() map[string]*Router {
	routerLock.RLock()
	defer routerLock.RUnlock()
	return routers
}

// GetRouter 获取指定id的Router
func GetRouter(id string) *Router {
	log.Infof("rtc.GetRouter id=%s", id)
	routerLock.RLock()
	defer routerLock.RUnlock()
	return routers[id]
}

// AddRouter 增加一个指定id的Router
func AddRouter(id string) *Router {
	log.Infof("rtc.AddRouter id=%s", id)
	routerLock.Lock()
	defer routerLock.Unlock()
	routers[id] = NewRouter(id)
	if err := routers[id].InitPlugins(pluginsConfig); err != nil {
		log.Errorf("rtc.AddRouter InitPlugins err=%v", err)
		return nil
	}
	return routers[id]
}

// DelRouter 删除指定id的Router
func DelRouter(id string) {
	log.Infof("rtc.DelRouter id=%s", id)
	router := GetRouter(id)
	if router == nil {
		return
	}
	router.Close()
	routerLock.Lock()
	defer routerLock.Unlock()
	delete(routers, id)
}

// CheckRoute 查询所有router的状态
func CheckRoute() {
	t := time.NewTicker(statCycle)
	defer t.Stop()
	for {
		if stop {
			return
		}

		select {
		case <-t.C:
			routerLock.Lock()
			for id, Router := range routers {
				if !Router.Alive() {
					Router.Close()
					delete(routers, id)
					CleanPub <- id
					log.Infof("Router delete %v", id)
				}
			}
			routerLock.Unlock()
		}
	}
}
