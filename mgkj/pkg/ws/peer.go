package ws

import (
	"mgkj/pkg/log"
	"mgkj/pkg/timing"

	"github.com/gearghost/go-protoo/peer"
	"github.com/gearghost/go-protoo/transport"
)

// Peer peer对象
type Peer struct {
	peer.Peer
	streamtimer *timing.StreamTimer
	appid       string
}

// NewPeer 初始化peer对象
func NewPeer(id string, t *transport.WebSocketTransport) *Peer {
	return newPeer(id, t)
}

func newPeer(id string, t *transport.WebSocketTransport) *Peer {
	return &Peer{
		Peer: *peer.NewPeer(id, t),
	}
}

func (p *Peer) SetAppID(appid string) {
	p.appid = appid
}

func (p *Peer) GetAppID() string {
	return p.appid
}

func (p *Peer) SetStreamTimer(timer *timing.StreamTimer) {
	p.streamtimer = timer
}

func (p *Peer) GetStreamTimer() *timing.StreamTimer {
	return p.streamtimer
}

// On 事件处理
func (p *Peer) On(event, listener interface{}) {
	p.Peer.On(event, listener)
}

// Request 发请求
func (p *Peer) Request(method string, data map[string]interface{}) {
	p.Peer.Request(method, data, accept, reject)
}

// Close peer关闭
func (p *Peer) Close() {
	p.Peer.Close()
}

func accept(data map[string]interface{}) {
	log.Infof("peer accept data=%v", data)
}

func reject(errorCode int, errorReason string) {
	log.Infof("reject errorCode=%v errorReason=%v", errorCode, errorReason)
}
