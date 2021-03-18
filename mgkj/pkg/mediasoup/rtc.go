package mediasoup

import (
	"mgkj/pkg/log"
	"sync"
	"time"

	mediasoup "github.com/jiyeyuran/mediasoup-go"
)

const (
	statCycle    = 2 * time.Second
	maxCleanSize = 100
)

var (
	worker     *mediasoup.Worker
	CleanPub   = make(chan string, maxCleanSize)
	routers    = make(map[string]*Router)
	routerLock sync.RWMutex
	stop       bool
)

// InitWorker 初始化worker
func InitWorker() {
	stop = false
	worker, err := mediasoup.NewWorker()
	if err != nil {
		log.Errorf(err.Error())
		panic(err)
	}

	worker.On("died", func(err error) {
		log.Errorf(err.Error())
		panic(err)
	})

	go CheckRoute()
}

// FreeWorker 关闭worker
func FreeWorker() {
	stop = true
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

// GetOrNewRouter 获取Router
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

// GetRouter 获取Router
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
