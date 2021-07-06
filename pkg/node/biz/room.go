package biz

import (
	"fmt"
	"signal/pkg/ws"
)

// RoomNode 房间对象
type RoomNode struct {
	room *ws.Room
}

// GetID 获取房间rid
func (r *RoomNode) GetID() string {
	return r.room.ID()
}

// NewRoom 创建一个房间
func NewRoom(rid string) *RoomNode {
	node := new(RoomNode)
	node.room = ws.NewRoom(rid)
	roomLock.Lock()
	rooms[rid] = node
	roomLock.Unlock()
	return node
}

// GetRoom 获取指定房间
func GetRoom(rid string) *RoomNode {
	roomLock.RLock()
	node := rooms[rid]
	roomLock.RUnlock()
	return node
}

// DelRoom 删除指定房间
func DelRoom(rid string) {
	roomLock.Lock()
	if rooms[rid] != nil {
		rooms[rid].room.Close()
	}
	delete(rooms, rid)
	roomLock.Unlock()
}

// AddPeer 房间增加指定的人
func AddPeer(rid string, peer *ws.Peer) {
	uid := peer.ID()
	node := GetRoom(rid)
	// 房间空，创建一个
	if node == nil {
		node = NewRoom(rid)
	}
	if node.room.GetPeer(uid) != nil {
		node.room.RemovePeer(uid)
	}
	node.room.AddPeer(peer)
}

// DelPeer 删除房间里的人
func DelPeer(rid, uid string) {
	node := GetRoom(rid)
	if node != nil {
		node.room.RemovePeer(uid)
		// 房间空，删除房间
		peers := node.room.GetPeers()
		nCount := len(peers)
		if nCount == 0 {
			DelRoom(rid)
		}
	}
}

// GetPeer 获取房间里面的人
func GetPeer(rid, uid string) *ws.Peer {
	node := GetRoom(rid)
	if node != nil {
		return node.room.GetPeer(uid)
	}
	return nil
}

// NotifyAll 通知房间所有人
func NotifyAll(rid string, method string, msg map[string]interface{}) {
	//log.Infof("biz.NotifyAll rid=%s method=%s msg=%v", rid, method, msg)
	logger.Infof(fmt.Sprintf("biz.NotifyAll rid=%s, method=%s, msg=%v", rid, method, msg), "rid", rid)
	node := GetRoom(rid)
	if node != nil {
		for _, peer := range node.room.GetPeers() {
			if peer != nil {
				peer.Notify(method, msg)
			}
		}
	}
}

// NotifyAllWithoutPeer 通知房间所有人除去peer
func NotifyAllWithoutPeer(rid string, peer *ws.Peer, method string, msg map[string]interface{}) {
	//log.Infof("biz.NotifyAllWithoutPeer rid=%s uid=%s method=%s msg=%v", rid, peer.ID(), method, msg)
	logger.Infof(fmt.Sprintf("biz.NotifyAllWithoutPeer rid=%s,uid=%s, method=%s, msg=%v", rid, peer.ID(), method, msg),
		"rid", rid)
	node := GetRoom(rid)
	if node != nil {
		node.room.Notify(peer, method, msg)
	}
}

// NotifyAllWithoutID 通知房间所有人除去skipID
func NotifyAllWithoutID(rid string, skipID string, method string, msg map[string]interface{}) {
	//log.Infof("biz.NotifyAllWithoutID rid=%s uid=%s method=%s msg=%v", rid, skipID, method, msg)
	logger.Infof(fmt.Sprintf("biz.NotifyAllWithoutID, rid=%s, skipID=%s, method=%s, msg=%v", rid, skipID, method, msg),
		"rid", rid)
	node := GetRoom(rid)
	if node != nil {
		node.room.Lock()
		for _, peer := range node.room.GetPeers() {
			if peer != nil && peer.ID() != skipID {
				peer.Notify(method, msg)
			}
		}
		node.room.Unlock()
	}
}
