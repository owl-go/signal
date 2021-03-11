package ws

import (
	"github.com/cloudwebrtc/go-protoo/logger"
	"github.com/cloudwebrtc/go-protoo/transport"
	"sync"
)

type Room struct {
	*sync.Mutex
	peers  map[string]*Peer
	closed bool
	id     string
}

func NewRoom(roomId string) *Room {
	room := &Room{
		peers:  make(map[string]*Peer),
		closed: false,
		id:     roomId,
	}
	room.Mutex = new(sync.Mutex)
	return room
}

func (room *Room) CreatePeer(peerId string, transport *transport.WebSocketTransport) *Peer {
	newPeer := newPeer(peerId, transport)
	newPeer.On("close", func(code int, err string) {
		room.Lock()
		defer room.Unlock()
		delete(room.peers, peerId)
	})
	room.Lock()
	defer room.Unlock()
	room.peers[peerId] = newPeer
	return newPeer
}

func (room *Room) AddPeer(newPeer *Peer) {
	room.Lock()
	defer room.Unlock()
	room.peers[newPeer.ID()] = newPeer
}

func (room *Room) GetPeer(peerId string) *Peer {
	room.Lock()
	defer room.Unlock()
	if peer, ok := room.peers[peerId]; ok {
		return peer
	}
	return nil
}

func (room *Room) Map(fn func(string, *Peer)) {
	room.Lock()
	defer room.Unlock()
	for id, peer := range room.peers {
		fn(id, peer)
	}
}

func (room *Room) GetPeers() map[string]*Peer {
	return room.peers
}

func (room *Room) RemovePeer(peerId string) {
	room.Lock()
	defer room.Unlock()
	delete(room.peers, peerId)
}

func (room *Room) ID() string {
	return room.id
}

func (room *Room) HasPeer(peerId string) bool {
	room.Lock()
	defer room.Unlock()
	_, ok := room.peers[peerId]
	return ok
}

func (room *Room) Notify(from *Peer, method string, data map[string]interface{}) {
	room.Lock()
	defer room.Unlock()
	for id, peer := range room.peers {
		//send to other peers
		if id != from.ID() {
			peer.Notify(method, data)
		}
	}
}

func (room *Room) Close() {
	logger.Warnf("Close all peers !")
	room.Lock()
	defer room.Unlock()
	for id, peer := range room.peers {
		logger.Warnf("Close => peer(%s).", id)
		peer.Close()
	}
	room.closed = true
}
