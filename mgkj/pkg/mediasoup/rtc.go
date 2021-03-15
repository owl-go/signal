package mediasoup

import (
	"encoding/json"
	"mgkj/pkg/log"
	"sync"

	mediasoup "github.com/jiyeyuran/mediasoup-go"
)

var (
	worker     *mediasoup.Worker
	routers    = make(map[string]*Router)
	routerLock sync.RWMutex
)

// InitWork 初始化work
func InitWork() {
	work, err := mediasoup.NewWorker()
	if err != nil {
		panic(err)
	}

	worker = work
	worker.On("died", func(err error) {
		log.Errorf("%s", err)
	})

	dump, _ := worker.Dump()
	log.Debugf("dump: %+v", dump)

	usage, err := worker.GetResourceUsage()
	if err != nil {
		panic(err)
	}
	data, _ := json.Marshal(usage)
	log.Debugf("usage: %s", data)
}

// GetOrNewRouter 获取Router
func GetOrNewRouter(id string) *Router {
	log.Infof("rtc.GetOrNewRouter id=%s", id)
	router := GetRouter(id)
	if router == nil {
		return AddRouter(id)
	}
	return router
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
	routers[id] = NewRouter(worker)
	return routers[id]
}

// DelRouter delete pub
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

// Close close all Router
func Close() {
	routerLock.Lock()
	defer routerLock.Unlock()
	for id, router := range routers {
		if router != nil {
			router.Close()
			delete(routers, id)
		}
	}
	worker.Close()
}
