package node

import (
	"sync"
	"time"

	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/util"
)

var (
	medias     = make(map[string]*MediaNode)
	mediasLock sync.RWMutex
	peers      = make(map[string]*PeerNode)
	peerLock   sync.RWMutex
	rooms      = make(map[string]*RoomNode)
	roomLock   sync.RWMutex
)

// MediaNode 媒体结构
type MediaNode struct {
	rid   string
	uid   string
	mid   string
	minfo string
	nid   string
}

// PeerNode peer结构
type PeerNode struct {
	uid  string
	info string
	mids []*MediaNode
}

// RoomNode 房间结构
type RoomNode struct {
	rid   string
	peers []*PeerNode
}

// handleRPCMsgs 接收消息处理
func handleRPCMsgs() {
	rpcMsgs, err := amqp.ConsumeRPC()
	if err != nil {
		log.Errorf(err.Error())
		return
	}

	go func() {
		for rpcm := range rpcMsgs {
			msg := util.Unmarshal(string(rpcm.Body))
			src := rpcm.ReplyTo
			corrID := rpcm.CorrelationId
			log.Infof("islb.handleRPCMsgs msg=%v", msg)
			method := util.Val(msg, "method")
			if method == "" {
				continue
			}

			switch method {
			case proto.IslbClientOnJoin:
				clientJoin(msg)
			case proto.IslbClientOnLeave:
				clientLeave(msg)
			case proto.IslbOnStreamAdd:
				streamAdd(msg)
			case proto.IslbOnStreamRemove:
				streamRemove(msg)
			case proto.IslbOnBroadcast:
				broadcast(msg)
			case proto.IslbGetMediaPubs:
				getAllPubs(msg, src, corrID)
			}
		}
	}()
}

// clientJoin 有人加入房间
func clientJoin(data map[string]interface{}) {
	log.Infof("amqp.rpc.clientJoin data=%v", data)
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	info := util.Val(data, "info")
	// 找到保存用户信息的key
	ukey := proto.GetUserInfoKey(rid)
	// 写入key值
	err := redis.HSetTTL(ukey, uid, info, redisKeyTTL)
	if err != nil {
		log.Errorf("redis.HSetTTL clientJoin err = %v", err)
	}
	// 生成resp对象
	msg := util.Map("method", proto.IslbClientOnJoin, "rid", rid, "uid", uid, "info", info)
	log.Infof("amqp.BroadCast msg=%v", msg)
	amqp.BroadCast(msg)
}

// clientLeave 有人退出
func clientLeave(data map[string]interface{}) {
	log.Infof("amqp.rpc.clientLeave data=%v", data)
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	// 找到保存用户信息的key
	ukey := proto.GetUserInfoKey(rid)
	// 删除key值
	info := redis.HGet(ukey, uid)
	err := redis.HDel(ukey, uid)
	if err != nil {
		log.Errorf("redis.HDel clientLeave err = %v", err)
	}
	// 生成resp对象
	msg := util.Map("method", proto.IslbClientOnLeave, "rid", rid, "uid", uid, "info", info)
	log.Infof("amqp.BroadCast msg=%v", msg)
	// make broadcast leave msg after remove stream msg, for ion block bug
	time.Sleep(500 * time.Millisecond)
	amqp.BroadCast(msg)
}

// streamAdd 有人发布流
func streamAdd(data map[string]interface{}) {
	log.Infof("amqp.rpc.streamAdd data=%v", data)
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")
	minfo := util.Val(data, "minfo")
	// 找到保存用户发布流的key
	ukey := proto.GetPubMediaKey(rid, uid)
	// 写入key值
	err := redis.HSetTTL(ukey, mid, minfo, redisKeyTTL)
	if err != nil {
		log.Errorf("redis.HSetTTL streamAdd err = %v", err)
	}
	// 生成resp对象
	msg := util.Map("method", proto.IslbOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo)
	log.Infof("amqp.BroadCast msg=%v", msg)
	amqp.BroadCast(msg)
}

// streamRemove 有人取消发布流
func streamRemove(data map[string]interface{}) {
	log.Infof("amqp.rpc.streamRemove data=%v", data)
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")
	// 找到保存用户发布流的key
	ukey := proto.GetPubMediaKey(rid, uid)
	// 删除key值
	minfo := redis.HGet(ukey, mid)
	err := redis.HDel(ukey, mid)
	if err != nil {
		log.Errorf("redis.HDel streamRemove err = %v", err)
	}
	// 生成resp对象
	msg := util.Map("method", proto.IslbOnStreamRemove, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo)
	log.Infof("amqp.BroadCast msg=%v", msg)
	amqp.BroadCast(msg)
}

// broadcast 接收到消息，发送广播出去
func broadcast(data map[string]interface{}) {
	log.Infof("amqp.rpc.broadcast data=%v", data)
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mData := util.Val(data, "data")
	// resp对象
	msg := util.Map("method", proto.IslbOnBroadcast, "rid", rid, "uid", uid, "data", mData)
	log.Infof("amqp.BroadCast msg=%v", msg)
	amqp.BroadCast(msg)
}

// getAllPubs 获取房间所有人的发布流
func getAllPubs(data map[string]interface{}, from, corrID string) {
	log.Infof("amqp.rpc.getAllPubs data=%v", data)
	rid := util.Val(data, "rid")
	// 找到保存用户信息的key
	ukey := proto.GetUserInfoKey(rid)
	// 查询key值
	uids := redis.HGetAll(ukey)
	for uid := range uids {
		getOnePubs(rid, uid, from, corrID)
	}
}

// getOnePubs 获取一个用户的发布流
func getOnePubs(rid, uid, from, corrID string) {
	// 找到保存用户发布流的key
	ukey := proto.GetPubMediaKey(rid, uid)
	// 查询key值
	mids := redis.HGetAll(ukey)
	for mid := range mids {
		minfo := redis.HGet(ukey, mid)
		resp := util.Map("response", proto.IslbGetMediaPubs, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo)
		log.Infof("amqp.RpcCall from=%s resp=%v corrID=%s", from, resp, corrID)
		amqp.RPCCall(from, resp, corrID)
	}
}

// removePeerAll 删除一个用户所有的信息
func removePeerAll(data map[string]interface{}) {
	log.Infof("amqp.rpc.removePeerAll data=%v", data)
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	// 找到保存用户发布流的key
	ukey := proto.GetPubMediaKey(rid, uid)
	mids := redis.HGetAll(ukey)
	for mid := range mids {
		// 删除用户发布流
		minfo := redis.HGet(ukey, mid)
		err := redis.HDel(ukey, mid)
		if err != nil {
			log.Errorf("redis.HDel streamRemove err = %v", err)
		}
		// 生成resp对象
		msg := util.Map("method", proto.IslbOnStreamRemove, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo)
		log.Infof("amqp.BroadCast msg=%v", msg)
		amqp.BroadCast(msg)
	}
	// 找到保存用户信息的key
	ukey = proto.GetUserInfoKey(rid)
	// 删除key值
	info := redis.HGet(ukey, uid)
	err := redis.HDel(ukey, uid)
	if err != nil {
		log.Errorf("redis.HDel clientLeave err = %v", err)
	}
	// 生成resp对象
	msg := util.Map("method", proto.IslbClientOnLeave, "rid", rid, "uid", uid, "info", info)
	log.Infof("amqp.BroadCast msg=%v", msg)
	amqp.BroadCast(msg)
}
