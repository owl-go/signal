package rtc

import (
	"fmt"
	"sync"
	"time"

	"mgkj/pkg/log"
	"mgkj/pkg/rtc/plugins"
	"mgkj/pkg/rtc/transport"

	"github.com/pion/webrtc/v2"
)

const (
	statCycle    = 3 * time.Second
	maxCleanSize = 100
)

var (
	routers    = make(map[string]*Router)
	routerLock sync.RWMutex

	//CleanChannel return the dead pub's mid
	CleanChannel  = make(chan string, maxCleanSize)
	pluginsConfig plugins.Config

	stop bool
)

// InitIce ice urls
func InitIce(iceServers []webrtc.ICEServer, icePortStart, icePortEnd uint16) error {
	//init ice urls and ICE settings
	return transport.InitWebRTC(iceServers, icePortStart, icePortEnd)
}

// InitPlugins plugins config
func InitPlugins(config plugins.Config) {
	pluginsConfig = config
	log.Infof("InitPlugins pluginsConfig=%+v", pluginsConfig)
	go check()
}

// CheckPlugins plugins config
func CheckPlugins(config plugins.Config) error {
	return plugins.CheckPlugins(config)
}

func GetOrNewRouter(id string) *Router {
	log.Infof("rtc.GetOrNewRouter id=%s", id)
	router := GetRouter(id)
	if router == nil {
		return AddRouter(id)
	}
	return router
}

// Get All Routers
func GetRouters() map[string]*Router {
	routerLock.RLock()
	defer routerLock.RUnlock()
	return routers
}

// GetRouter get router from map
func GetRouter(id string) *Router {
	log.Infof("rtc.GetRouter id=%s", id)
	routerLock.RLock()
	defer routerLock.RUnlock()
	return routers[id]
}

// AddRouter add a new router
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

// DelRouter delete pub
func DelRouter(id string) {
	log.Infof("DelRouter id=%s", id)
	router := GetRouter(id)
	if router == nil {
		return
	}
	router.Close()
	routerLock.Lock()
	defer routerLock.Unlock()
	delete(routers, id)
}

// Close close all Router
func Close() {
	if stop {
		return
	}
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

// check show all Routers' stat
func check() {
	t := time.NewTicker(statCycle)
	for {
		select {
		case <-t.C:
			info := "\n----------------rtc-----------------\n"
			print := false
			routerLock.Lock()
			if len(routers) > 0 {
				print = true
			}

			for id, Router := range routers {
				if !Router.Alive() {
					Router.Close()
					delete(routers, id)
					CleanChannel <- id
					log.Infof("Stat delete %v", id)
				}
				info += "pub: " + id + "\n"
				subs := Router.GetSubs()
				if len(subs) < 6 {
					for id := range subs {
						info += fmt.Sprintf("sub: %s\n\n", id)
					}
				} else {
					info += fmt.Sprintf("subs: %d\n\n", len(subs))
				}
			}
			routerLock.Unlock()
			if print {
				log.Infof(info)
			}
		}
	}
}
