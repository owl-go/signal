package node

import (
	"mgkj/pkg/log"

	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/room"
)

// RoomObj 房间对象
type RoomObj struct {
	room *room.Room
}

// GetID 获取房间rid
func (r *RoomObj) GetID() string {
	return r.room.ID()
}

// NewRoom 创建一个房间
func NewRoom(rid string) *RoomObj {
	roomobj := new(RoomObj)
	roomobj.room = room.NewRoom(rid)
	roomLock.Lock()
	rooms[rid] = roomobj
	roomLock.Unlock()
	return roomobj
}

// GetRoom 根据rid获取指定房间
func GetRoom(rid string) *RoomObj {
	roomLock.RLock()
	roomobj := rooms[rid]
	roomLock.RUnlock()
	return roomobj
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

// GetRoomByPeer 根据peer查询房间信息
func GetRoomByPeer(id string) *RoomObj {
	roomLock.RLock()
	defer roomLock.RUnlock()
	for _, roomobj := range rooms {
		if roomobj == nil {
			continue
		}
		if peer := roomobj.room.GetPeer(id); peer != nil {
			return roomobj
		}
	}
	return nil
}

// AddPeer 房间增加指定的人
func AddPeer(rid string, peer *peer.Peer) {
	roomobj := GetRoom(rid)
	if roomobj == nil {
		roomobj = NewRoom(rid)
	}
	if roomobj.room.GetPeer(peer.ID()) != nil {
		roomobj.room.RemovePeer(peer.ID())
	}
	roomobj.room.AddPeer(peer)
}

// DelPeer 删除房间里的人
func DelPeer(rid, uid string) {
	roomobj := GetRoom(rid)
	if roomobj != nil {
		roomobj.room.RemovePeer(uid)
		// 判断房间里面剩余人个数
		peers := roomobj.room.GetPeers()
		nCount := len(peers)
		if nCount == 0 {
			DelRoom(roomobj.room.ID())
		}
	}
}

// HasPeer 判断房间是否有指定的人
func HasPeer(rid string, peer *peer.Peer) bool {
	roomobj := GetRoom(rid)
	if roomobj == nil {
		return false
	}
	return (roomobj.room.GetPeer(peer.ID()) != nil)
}

// HasPeer2 判断房间是否有指定的人
func HasPeer2(rid, uid string) bool {
	roomobj := GetRoom(rid)
	if roomobj == nil {
		return false
	}
	return (roomobj.room.GetPeer(uid) != nil)
}

// NotifyAll 通知房间所有人
func NotifyAll(rid string, method string, msg map[string]interface{}) {
	log.Infof("biz.NotifyAll rid=%s method=%s msg=%v", rid, method, msg)
	roomobj := GetRoom(rid)
	if roomobj != nil {
		for _, peer := range roomobj.room.GetPeers() {
			if peer != nil {
				peer.Notify(method, msg)
			}
		}
	}
}

// NotifyAllWithoutPeer 通知房间所有人除去peer
func NotifyAllWithoutPeer(rid string, peer *peer.Peer, method string, msg map[string]interface{}) {
	log.Infof("biz.NotifyAllWithoutPeer rid=%s peer=%s method=%s msg=%v", rid, peer.ID(), method, msg)
	roomobj := GetRoom(rid)
	if roomobj != nil {
		roomobj.room.Notify(peer, method, msg)
	}
}

// NotifyAllWithoutID 通知房间所有人除去skipID
func NotifyAllWithoutID(rid string, skipID string, method string, msg map[string]interface{}) {
	log.Infof("biz.NotifyAllWithoutID rid=%s skipID=%s method=%s msg=%v", rid, skipID, method, msg)
	roomobj := GetRoom(rid)
	if roomobj != nil {
		for _, peer := range roomobj.room.GetPeers() {
			if peer != nil && peer.ID() != skipID {
				peer.Notify(method, msg)
			}
		}
	}
}
