package ws

import (
	"sync"

	"github.com/cloudwebrtc/go-protoo/logger"
	"github.com/cloudwebrtc/go-protoo/transport"
)

// Room room对象
type Room struct {
	*sync.Mutex
	peers  map[string]*Peer
	closed bool
	id     string
}

// NewRoom 新建对象
func NewRoom(roomID string) *Room {
	room := &Room{
		peers:  make(map[string]*Peer),
		closed: false,
		id:     roomID,
	}
	room.Mutex = new(sync.Mutex)
	return room
}

// CreatePeer 新建peer
func (room *Room) CreatePeer(peerID string, transport *transport.WebSocketTransport) *Peer {
	newPeer := newPeer(peerID, transport)
	/*
		newPeer.On("close", func(code int, err string) {
			room.Lock()
			defer room.Unlock()
			delete(room.peers, peerID)
		})*/
	room.Lock()
	defer room.Unlock()
	room.peers[peerID] = newPeer
	return newPeer
}

// AddPeer 新增peer
func (room *Room) AddPeer(newPeer *Peer) {
	room.Lock()
	defer room.Unlock()
	room.peers[newPeer.ID()] = newPeer
}

// GetPeer 获取peer
func (room *Room) GetPeer(peerID string) *Peer {
	room.Lock()
	defer room.Unlock()
	if peer, ok := room.peers[peerID]; ok {
		return peer
	}
	return nil
}

// Map 遍历处理
func (room *Room) Map(fn func(string, *Peer)) {
	room.Lock()
	defer room.Unlock()
	for id, peer := range room.peers {
		fn(id, peer)
	}
}

// GetPeers 获取所有的peer
func (room *Room) GetPeers() map[string]*Peer {
	return room.peers
}

// RemovePeer 删除peer
func (room *Room) RemovePeer(peerID string) {
	room.Lock()
	defer room.Unlock()
	delete(room.peers, peerID)
}

// ID 返回id
func (room *Room) ID() string {
	return room.id
}

// HasPeer 查询peer
func (room *Room) HasPeer(peerID string) bool {
	room.Lock()
	defer room.Unlock()
	_, ok := room.peers[peerID]
	return ok
}

// Notify 通知peers
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

// Close 关闭room
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
