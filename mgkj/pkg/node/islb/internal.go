package node

import (
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
		for rpcm := range rpcMsgs {
			msg := util.Unmarshal(string(rpcm.Body))
			log.Infof("islb.handleRPCMsgs msg=%v", msg)

			src := rpcm.ReplyTo
			index := rpcm.CorrelationId
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
			case proto.IslbGetSfuInfo:
				getSfuByMid(msg, src, index)
			case proto.IslbGetMediaInfo:
				getSfuByMid(msg, src, index)
			case proto.IslbGetMediaPubs:
				getMediaPubs(msg, src, index)
			}
		}
	}()
}

// clientJoin 有人加入房间
func clientJoin(data map[string]interface{}) {
	log.Infof("islb.rpc.clientJoin data=%v", data)
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
	msg := util.Map("method", proto.IslbClientOnJoin, "rid", rid, "uid", uid, "info", info)
	amqp.BroadCast(msg)
}

// clientLeave 有人退出房间
func clientLeave(data map[string]interface{}) {
	log.Infof("islb.rpc.clientLeave data=%v", data)
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
	msg := util.Map("method", proto.IslbClientOnLeave, "rid", rid, "uid", uid, "info", info)
	time.Sleep(500 * time.Millisecond)
	amqp.BroadCast(msg)
}

// streamAdd 有人发布流
func streamAdd(data map[string]interface{}) {
	log.Infof("islb.rpc.streamAdd data=%v", data)
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")
	nid := util.Val(data, "nid")
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
	msg := util.Map("method", proto.IslbOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo)
	amqp.BroadCast(msg)
}

// streamRemove 有人取消发布流
func streamRemove(data map[string]interface{}) {
	log.Infof("islb.rpc.streamRemove data=%v", data)
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")
	// 获取媒体信息保存的key
	ukey := proto.GetMediaInfoKey(rid, uid, mid)
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
	msg := util.Map("method", proto.IslbOnStreamRemove, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo)
	amqp.BroadCast(msg)
}

// getSfuByMid 获取指定mid对应的sfu节点
func getSfuByMid(data map[string]interface{}, from, index string) {
	log.Infof("islb.rpc.getSfuByMid data=%v", data)
	mid := util.Val(data, "mid")
	// 获取发布流信息保存的key
	ukeys := redis.Keys("/sfu/mid/" + mid + "*")
	if ukeys != nil && len(ukeys) == 1 {
		nid := redis.Get(ukeys[0])
		resp := util.Map("response", proto.IslbGetSfuInfo, "mid", mid, "nid", nid)
		amqp.RPCCall(from, resp, index)
		return
	}
	resp := util.Map("response", proto.IslbGetSfuInfo, "mid", mid)
	amqp.RPCCall(from, resp, index)
}

// getMediaInfo 获取指定mid对应的流的信息
func getMediaInfo(data map[string]interface{}, from, index string) {
	log.Infof("islb.rpc.getMediaInfo data=%v", data)
	mid := util.Val(data, "mid")
	// 获取媒体信息保存的key
	ukeys := redis.Keys("/media/mid/" + mid + "*")
	if ukeys != nil && len(ukeys) == 1 {
		minfo := redis.Get(ukeys[0])
		resp := util.Map("response", proto.IslbGetMediaInfo, "mid", mid, "minfo", minfo)
		amqp.RPCCall(from, resp, index)
		return
	}
	resp := util.Map("response", proto.IslbGetSfuInfo, "mid", mid)
	amqp.RPCCall(from, resp, index)
}

// getMediaPubs 获取房间所有人的发布流
func getMediaPubs(data map[string]interface{}, from, index string) {
	log.Infof("islb.rpc.getAllPubs data=%v", data)
	rid := util.Val(data, "rid")
	// 找到保存用户流信息的key
	ukeys := redis.Keys("/media/mid/*/rid/" + rid)
	nlen := len(ukeys)
	for _, key := range ukeys {
		minfo := redis.Get(key)
		mid, uid, err := parseMediaKey(key)
		if err == nil {
			resp := util.Map("response", proto.IslbGetMediaPubs, "rid", rid, "uid", uid, "mid", mid, "minfo", minfo, "len", nlen)
			amqp.RPCCall(from, resp, index)
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
