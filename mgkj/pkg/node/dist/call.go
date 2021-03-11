package dist

import (
	"mgkj/pkg/ws"
	"sync"
	"time"
)

var (
	peers    map[string]*ws.Peer
	peerLock sync.RWMutex
	wsReq    func(method string, peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc)
)

const (
	statCycle = 5 * time.Second
)

func InitCallServer(host string, port int, cert, key string) {
	peers = make(map[string]*ws.Peer)
	initWebSocket(host, port, cert, key, Entry)
}

func initWebSocket(host string, port int, cert, key string, handler interface{}) {
	wsServer := ws.NewWebSocketServer(in)
	config := ws.DefaultConfig()
	config.Host = host
	config.Port = port
	config.CertFile = cert
	config.KeyFile = key
	wsReq = handler.(func(method string, peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc))
	go wsServer.Bind(config)
}
