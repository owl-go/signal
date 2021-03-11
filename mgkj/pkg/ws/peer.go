package ws

import (
	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/transport"
	"mgkj/pkg/log"
)

func NewPeer(id string, t *transport.WebSocketTransport) *Peer {
	return newPeer(id, t)
}

func newPeer(id string, t *transport.WebSocketTransport) *Peer {
	return &Peer{
		Peer: *peer.NewPeer(id, t),
	}
}

type Peer struct {
	peer.Peer
}

func (p *Peer) On(event, listener interface{}) {
	p.Peer.On(event, listener)
}

func (c *Peer) Request(method string, data map[string]interface{}) {
	c.Peer.Request(method, data, accept, reject)
}

func (c *Peer) Close() {
	c.Peer.Close()
}

func accept(data map[string]interface{}) {
	log.Infof("peer accept data=%v", data)
}

func reject(errorCode int, errorReason string) {
	log.Infof("reject errorCode=%v errorReason=%v", errorCode, errorReason)
}
