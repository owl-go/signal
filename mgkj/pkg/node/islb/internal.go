package node

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/util"
)

// handleRPCMsgs 接收消息处理
func handleRPCMsgs() {
	rpcMsgs, err := amqp.ConsumeRPC()
	if err != nil {
		log.Errorf(err.Error())
		return
	}

	go func() {
		defer util.Recover("islb.handleRPCMsgs")
		for rpcm := range rpcMsgs {
			var msg map[string]interface{}
			err := json.Unmarshal(rpcm.Body, &msg)
			if err != nil {
				log.Errorf("islb handleRPCMsgs Unmarshal err = %s", err.Error())
			}

			from := rpcm.ReplyTo
			corrID := rpcm.CorrelationId
			log.Infof("islb.handleRPCMsgs recv msg=%v, from=%s, corrID=%s", msg, from, corrID)

			method := util.Val(msg, "method")
			if method == "" {
				continue
			}

			switch method {
			case proto.DistToIslbLoginin:
				clientloginin(msg)
			case proto.DistToIslbLoginOut:
				clientloginout(msg)
			case proto.DistToIslbPeerHeart:
				clientPeerHeart(msg)
			case proto.DistToIslbPeerInfo:
				getPeerinfo(msg, from, corrID)

			case proto.BizToIslbOnJoin:
				clientJoin(msg)
			case proto.BizToIslbOnLeave:
				clientLeave(msg)
			case proto.BizToIslbOnStreamAdd:
				streamAdd(msg)
			case proto.BizToIslbOnStreamRemove:
				streamRemove(msg)
			case proto.BizToIslbBroadcast:
				broadcast(msg)
			case proto.BizToIslbGetSfuInfo:
				getSfuByMid(msg, from, corrID)
			case proto.BizToIslbGetMediaInfo:
				getSfuByMid(msg, from, corrID)
			case proto.BizToIslbGetMediaPubs:
				getMediaPubs(msg, from, corrID)
			}
		}
	}()
}

/*
	"method", proto.DistToIslbLoginin, "uid", uid, "nid", nid
*/
// clientloginin 有人登录到dist服务器
func clientloginin(data map[string]interface{}) {
	uid := util.Val(data, "uid")
	dist := util.Val(data, "nid")
	// 获取用户信息保存的key
	uKey := proto.GetUserDistKey(uid)
	// 写入key值
	err := redis.Set(uKey, dist, redisShort)
	if err != nil {
		log.Errorf("redis.Set clientloginin err = %v", err)
	}
}

/*
	"method", proto.DistToIslbLoginOut, "uid", uid, "nid", nid
*/
// clientloginin 有人退录到dist服务器
func clientloginout(data map[string]interface{}) {
	uid := util.Val(data, "uid")
	// 获取用户信息保存的key
	uKey := proto.GetUserDistKey(uid)
	// 写入key值
	err := redis.Del(uKey)
	if err != nil {
		log.Errorf("redis.Del clientloginout err = %v", err)
	}
}

/*
	"method", proto.DistToIslbPeerHeart, "uid", uid, "nid", nid
*/
// clientPeerHeart 有人发送心跳到dist服务器,更新key的时间
func clientPeerHeart(data map[string]interface{}) {
	uid := util.Val(data, "uid")
	dist := util.Val(data, "nid")
	// 获取用户信息保存的key
	uKey := proto.GetUserDistKey(uid)
	// 写入key值
	err := redis.Set(uKey, dist, redisShort)
	if err != nil {
		log.Errorf("redis.Set clientPeerHeart err = %v", err)
	}
}

/*
	"method", proto.DistToIslbPeerInfo, "uid", callee
*/
// getPeerinfo 获取Peer在哪个Dist服务器
func getPeerinfo(data map[string]interface{}, from, corrID string) {
	// 获取参数
	uid := util.Val(data, "uid")
	// 获取用户信息保存的key
	uKey := proto.GetUserDistKey(uid)
	dist := redis.Get(uKey)
	if dist == "" {
		resp := util.Map("method", proto.IslbToDistPeerInfo, "errorCode", 1, "errorReason", "uid is not live")
		amqp.RPCCall(from, resp, corrID)
	} else {
		resp := util.Map("method", proto.IslbToDistPeerInfo, "errorCode", 0, "nid", dist)
		amqp.RPCCall(from, resp, corrID)
	}
}

/*
	"method", proto.BizToIslbOnJoin, "rid", rid, "uid", uid, "info", info
*/
// clientJoin 有人加入房间
func clientJoin(data map[string]interface{}) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	info := util.Val(data, "info")
	// 获取用户信息保存的key
	uKey := proto.GetUserInfoKey(rid, uid)
	// 写入key值
	err := redis.Set(uKey, info, redisKeyTTL)
	if err != nil {
		log.Errorf("redis.Set clientJoin err = %v", err)
	}
	// 生成resp对象
	msg := util.Map("method", proto.IslbToBizOnJoin, "rid", rid, "uid", uid, "info", info)
	amqp.BroadCast(msg)
}

/*
	"method", proto.BizToIslbOnLeave, "rid", ridTmp, "uid", uid), ""
*/
// clientLeave 有人退出房间
func clientLeave(data map[string]interface{}) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	// 获取用户信息保存的key
	ukey := proto.GetUserInfoKey(rid, uid)
	// 删除key值
	info := redis.Get(ukey)
	err := redis.Del(ukey)
	if err != nil {
		log.Errorf("redis.Del clientLeave err = %v", err)
	}
	// 生成resp对象
	msg := util.Map("method", proto.IslbToBizOnLeave, "rid", rid, "uid", uid, "info", info)
	time.Sleep(200 * time.Millisecond)
	amqp.BroadCast(msg)
}

/*
	"method", proto.BizToIslbOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo, "nid", nid
*/
// streamAdd 有人发布流
func streamAdd(data map[string]interface{}) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	nid := util.Val(data, "nid")
	mid := util.Val(data, "mid")
	minfo := util.Val(data, "minfo")
	// 获取媒体信息保存的key
	ukey := proto.GetMediaInfoKey(rid, uid, mid)
	// 写入key值
	err := redis.Set(ukey, minfo, redisKeyTTL)
	if err != nil {
		log.Errorf("redis.Set streamAdd err = %v", err)
	}
	// 获取发布流信息保存的key
	ukey = proto.GetMediaPubKey(rid, uid, mid)
	// 写入key值
	err = redis.Set(ukey, nid, redisKeyTTL)
	if err != nil {
		log.Errorf("redis.Set streamAdd err = %v", err)
	}
	// 生成resp对象
	msg := util.Map("method", proto.IslbToBizOnStreamAdd, "rid", rid, "uid", uid, "nid", nid, "mid", mid, "minfo", minfo)
	amqp.BroadCast(msg)
}

/*
	"method", proto.BizToIslbOnStreamRemove, "rid", ridTmp, "uid", uid, "mid", ""
*/
// streamRemove 有人取消发布流
func streamRemove(data map[string]interface{}) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")
	// 获取媒体信息保存的key
	var ukey string
	if mid == "" {
		ukey = "/media/mid/*/uid/" + uid + "/rid/" + rid
		ukeys := redis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			// 删除key值
			minfo := redis.Get(ukey)
			err := redis.Del(ukey)
			if err != nil {
				log.Errorf("redis.Del streamRemove err = %v", err)
			}
			// 生成resp对象
			msg := util.Map("method", proto.IslbToBizOnStreamRemove, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo)
			amqp.BroadCast(msg)
		}
		ukey = "/sfu/mid/*/uid/" + uid + "/rid/" + rid
		ukeys = redis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			// 删除key值
			err := redis.Del(ukey)
			if err != nil {
				log.Errorf("redis.Del streamRemove err = %v", err)
			}
		}
	} else {
		ukey = proto.GetMediaInfoKey(rid, uid, mid)
		// 删除key值
		minfo := redis.Get(ukey)
		err := redis.Del(ukey)
		if err != nil {
			log.Errorf("redis.Del streamRemove err = %v", err)
		}
		// 获取发布流信息保存的key
		ukey = proto.GetMediaPubKey(rid, uid, mid)
		// 删除key值
		err = redis.Del(ukey)
		if err != nil {
			log.Errorf("redis.Del streamRemove err = %v", err)
		}
		// 生成resp对象
		msg := util.Map("method", proto.IslbToBizOnStreamRemove, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo)
		amqp.BroadCast(msg)
	}
}

/*
	"method", proto.BizToIslbBroadcast, "rid", rid, "uid", uid, "data", data
*/
// broadcast 发送广播
func broadcast(data map[string]interface{}) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	dataTmp := util.Val(data, "data")

	msg := util.Map("method", proto.IslbToBizBroadcast, "rid", rid, "uid", uid, "data", dataTmp)
	amqp.BroadCast(msg)
}

/*
	"method", proto.BizToIslbGetSfuInfo, "mid", mid
*/
// getSfuByMid 获取指定mid对应的sfu节点
func getSfuByMid(data map[string]interface{}, from, corrID string) {
	mid := util.Val(data, "mid")
	// 获取发布流信息保存的key
	ukeys := redis.Keys("/sfu/mid/" + mid + "*")
	if ukeys != nil && len(ukeys) == 1 {
		nid := redis.Get(ukeys[0])
		resp := util.Map("method", proto.IslbToBizGetSfuInfo, "mid", mid, "nid", nid)
		amqp.RPCCall(from, resp, corrID)
		return
	}
	resp := util.Map("method", proto.IslbToBizGetSfuInfo, "mid", mid)
	amqp.RPCCall(from, resp, corrID)
}

/*
	"method", proto.BizToIslbGetMediaInfo, "mid", mid
*/
// getMediaInfo 获取指定mid对应的流的信息
func getMediaInfo(data map[string]interface{}, from, corrID string) {
	mid := util.Val(data, "mid")
	// 获取媒体信息保存的key
	ukeys := redis.Keys("/media/mid/" + mid + "*")
	if ukeys != nil && len(ukeys) == 1 {
		minfo := redis.Get(ukeys[0])
		resp := util.Map("method", proto.IslbToBizGetMediaInfo, "mid", mid, "minfo", minfo)
		amqp.RPCCall(from, resp, corrID)
		return
	}
	resp := util.Map("method", proto.IslbToBizGetMediaInfo, "mid", mid)
	amqp.RPCCall(from, resp, corrID)
}

/*
	"method", proto.BizToIslbGetMediaPubs, "rid", rid, "uid", uid
*/
// getMediaPubs 获取房间所有人的发布流
func getMediaPubs(data map[string]interface{}, from, corrID string) {
	rid := util.Val(data, "rid")
	// 找到保存用户流信息的key
	ukeys := redis.Keys("/media/mid/*/rid/" + rid)
	nLen := len(ukeys)
	if nLen == 0 {
		resp := util.Map("method", proto.IslbToBizGetMediaPubs, "rid", rid, "len", nLen)
		amqp.RPCCall(from, resp, corrID)
		return
	}
	for _, key := range ukeys {
		minfo := redis.Get(key)
		mid, uid, err := parseMediaKey(key)
		if err == nil {
			ukey := proto.GetMediaPubKey(rid, uid, mid)
			nid := redis.Get(ukey)
			resp := util.Map("method", proto.IslbToBizGetMediaPubs, "rid", rid, "uid", uid, "nid", nid, "mid", mid, "minfo", minfo, "len", nLen)
			amqp.RPCCall(from, resp, corrID)
		}
	}
}

// parseMediaKey 分析key
func parseMediaKey(key string) (string, string, error) {
	arr := strings.Split(key, "/")
	if len(arr) < 6 {
		return "", "", fmt.Errorf("Can‘t parse mediainfo; [%s]", key)
	}

	mid := arr[3]
	uid := arr[5]
	return mid, uid, nil
}
