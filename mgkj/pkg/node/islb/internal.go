package node

import (
	"fmt"
	"strings"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"

	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/util"
)

// handleRPCRequest 接收消息处理
func handleRPCRequest(rpcID string) {
	log.Infof("handleRPCRequest: rpcID => [%v]", rpcID)

	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		go func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
			defer util.Recover("islb.handleRPCRequest")

			log.Infof("islb.handleRPCRequest recv request=%v", request)
			method := request["method"].(string)
			data := request["data"].(map[string]interface{})
			log.Infof("method => %s, data => %v", method, data)

			var result map[string]interface{}
			err := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))

			switch method {
			/* 处理和dist服务器通信 */
			case proto.DistToIslbLogin:
				result, err = clientlogin(data)
			case proto.DistToIslbLogout:
				result, err = clientlogout(data)
			case proto.DistToIslbPeerHeartbeat:
				result, err = clientPeerHeartbeat(data)
			case proto.DistToIslbPeerInfo:
				result, err = getPeerinfo(data)
			/* 处理和biz服务器通信 */
			case proto.BizToIslbOnJoin:
				result, err = clientJoin(data)
			case proto.BizToIslbOnLeave:
				result, err = clientLeave(data)
			case proto.BizToIslbOnStreamAdd:
				result, err = streamAdd(data)
			case proto.BizToIslbOnStreamRemove:
				result, err = streamRemove(data)
			case proto.BizToIslbKeepLive:
				result, err = keeplive(data)
			case proto.BizToIslbBroadcast:
				result, err = broadcast(data)
			case proto.BizToIslbGetSfuInfo:
				result, err = getSfuByMid(data)
			case proto.BizToIslbGetMediaPubs:
				result, err = getMediaPubs(data)
			case proto.BizToIslbPeerLive:
				result, err = getPeerLive(data)
			}
			if err != nil {
				reject(err.Code, err.Reason)
			} else {
				accept(result)
			}
		}(request, accept, reject)
	})
}

/*
	"method", proto.DistToIslbLogin, "uid", uid, "nid", nid
*/
// clientlogin 有人登录到dist服务器
func clientlogin(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	uid := util.Val(data, "uid")
	dist := util.Val(data, "nid")
	// 获取用户信息保存的key
	uKey := proto.GetUserDistKey(uid)
	// 写入key值
	err := redis.Set(uKey, dist, redisShort)
	if err != nil {
		log.Errorf("redis.Set clientlogin err = %v", err)
	}
	return util.Map(), nil
}

/*
	"method", proto.DistToIslbLogout, "uid", uid, "nid", nid
*/
// clientlogout 有人退录到dist服务器
func clientlogout(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	uid := util.Val(data, "uid")
	// 获取用户信息保存的key
	uKey := proto.GetUserDistKey(uid)
	// 写入key值
	err := redis.Del(uKey)
	if err != nil {
		log.Errorf("redis.Del clientloginout err = %v", err)
	}
	return util.Map(), nil
}

/*
	"method", proto.DistToIslbPeerHeartbeat, "uid", uid, "nid", nid
*/
// clientPeerHeartbeat 有人发送心跳到dist服务器,更新key的时间
func clientPeerHeartbeat(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	uid := util.Val(data, "uid")
	dist := util.Val(data, "nid")
	// 获取用户信息保存的key
	uKey := proto.GetUserDistKey(uid)
	// 写入key值
	err := redis.Set(uKey, dist, redisShort)
	if err != nil {
		log.Errorf("redis.Set clientPeerHeart err = %v", err)
	}
	return util.Map(), nil
}

/*
	"method", proto.DistToIslbPeerInfo, "uid", uid
*/
// getPeerinfo 获取Peer在哪个Dist服务器
func getPeerinfo(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	// 获取参数
	uid := util.Val(data, "uid")
	// 获取用户信息保存的key
	uKey := proto.GetUserDistKey(uid)
	dist := redis.Get(uKey)
	resp := make(map[string]interface{})
	if dist == "" {
		resp = util.Map("errorCode", 1)
	} else {
		resp = util.Map("errorCode", 0, "nid", dist)
	}
	return resp, nil
}

/*
	"method", proto.BizToIslbOnJoin, "rid", rid, "uid", uid, "info", info
*/
// clientJoin 有人加入房间
func clientJoin(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	info := util.Val(data, "info")
	// 获取用户的信息
	uKey := proto.GetUserInfoKey(rid, uid)
	// 写入key值
	err := redis.Set(uKey, info, redisShort)
	if err != nil {
		log.Errorf("redis.Set clientJoin err = %v", err)
	}
	// 生成resp对象
	broadcaster.Say(proto.IslbToBizOnJoin, util.Map("rid", rid, "uid", uid, "info", data["info"]))
	return util.Map(), nil
}

/*
	"method", proto.BizToIslbOnLeave, "rid", rid, "uid", uid
*/
// clientLeave 有人退出房间
func clientLeave(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	// 获取用户的信息
	uKey := proto.GetUserInfoKey(rid, uid)
	// 删除key值
	err := redis.Del(uKey)
	if err != nil {
		log.Errorf("redis.Del clientLeave err = %v", err)
	} else {
		time.Sleep(200 * time.Millisecond)
		broadcaster.Say(proto.IslbToBizOnLeave, util.Map("rid", rid, "uid", uid))
	}
	return util.Map(), nil
}

/*
	"method", proto.BizToIslbOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "nid", nid, "minfo", minfo
*/
// streamAdd 有人发布流
func streamAdd(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")
	nid := util.Val(data, "nid")
	minfo := util.Val(data, "minfo")

	// 获取用户发布的流信息
	ukey := proto.GetMediaInfoKey(rid, uid, mid)
	// 写入key值
	err := redis.Set(ukey, minfo, redisKeyTTL)
	if err != nil {
		log.Errorf("redis.Set streamAdd err = %v", err)
	}

	// 获取用户发布流对应的sfu信息
	ukey = proto.GetMediaPubKey(rid, uid, mid)
	// 写入key值
	err = redis.Set(ukey, nid, redisKeyTTL)
	if err != nil {
		log.Errorf("redis.Set streamAdd err = %v", err)
	}
	// 生成resp对象
	broadcaster.Say(proto.IslbToBizOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "nid", nid, "minfo", data["minfo"]))
	return util.Map(), nil
}

/*
	"method", proto.BizToIslbOnStreamRemove, "rid", rid, "uid", uid, "mid", ""
*/
// streamRemove 有人取消发布流
func streamRemove(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")
	// 判断mid是否为空
	var ukey string
	if mid == "" {
		ukey = "/media/rid/" + rid + "/uid/" + uid + "/mid/*"
		ukeys := redis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			// 删除key值
			err := redis.Del(ukey)
			if err != nil {
				log.Errorf("redis.Del streamRemove err = %v", err)
			}
		}
		ukey = "/pub/rid/" + rid + "/uid/" + uid + "/mid/*"
		ukeys = redis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			mid, _, _ := parseMediaKey(ukey)
			// 删除key值
			err := redis.Del(ukey)
			if err != nil {
				log.Errorf("redis.Del streamRemove err = %v", err)
			} else {
				// 生成resp对象
				broadcaster.Say(proto.IslbToBizOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
			}
		}
	} else {
		// 获取用户发布的流信息
		ukey = proto.GetMediaInfoKey(rid, uid, mid)
		// 删除key值
		err := redis.Del(ukey)
		if err != nil {
			log.Errorf("redis.Del streamRemove err = %v", err)
		}
		// 获取用户发布流对应的sfu信息
		ukey = proto.GetMediaPubKey(rid, uid, mid)
		// 删除key值
		err = redis.Del(ukey)
		if err != nil {
			log.Errorf("redis.Del streamRemove err = %v", err)
		} else {
			// 生成resp对象
			broadcaster.Say(proto.IslbToBizOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
		}
	}
	return util.Map(), nil
}

/*
	"method", proto.BizToIslbKeepLive, "rid", rid, "uid", uid, "info", info
*/
// keeplive 保活
func keeplive(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	info := util.Val(data, "info")
	// 获取用户的信息
	uKey := proto.GetUserInfoKey(rid, uid)
	// 写入key值
	err := redis.Set(uKey, info, redisShort)
	if err != nil {
		log.Errorf("redis.Set keeplive err = %v", err)
	}
	return util.Map(), nil
}

/*
	"method", proto.BizToIslbBroadcast, "rid", rid, "uid", uid, "data", data
*/
// broadcast 发送广播
func broadcast(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	broadcaster.Say(proto.IslbToBizBroadcast, util.Map("rid", rid, "uid", uid, "data", data["data"]))
	return util.Map(), nil
}

/*
	"method", proto.BizToIslbGetSfuInfo, "rid", rid, "mid", mid
*/
// getSfuByMid 获取指定mid对应的sfu节点
func getSfuByMid(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	mid := util.Val(data, "mid")
	uid := proto.GetUIDFromMID(mid)
	// 获取用户发布流对应的sfu信息
	uKey := proto.GetMediaPubKey(rid, uid, mid)
	nid := redis.Get(uKey)
	log.Infof("getSfuByMid ==> %v", nid)
	resp := make(map[string]interface{})
	if nid != "" {
		resp = util.Map("errorCode", 0, "rid", rid, "mid", mid, "nid", nid)
	} else {
		resp = util.Map("errorCode", 1, "rid", rid, "mid", mid)
	}
	return resp, nil
}

/*
	"method", proto.BizToIslbGetMediaPubs, "rid", rid, "uid", uid
*/
// getMediaPubs 获取房间所有人的发布流
func getMediaPubs(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uidTmp := util.Val(data, "uid")
	// 找到保存用户流信息的key
	var pubs []map[string]interface{}
	for _, key := range redis.Keys("/media/rid/" + rid + "/uid/*") {
		mid, uid, err := parseMediaKey(key)
		if err != nil {
			resp := util.Map("errorCode", 1, "rid", rid)
			return resp, nil
		}

		if uidTmp == uid {
			continue
		}

		minfo := redis.Get(key)
		uKey := proto.GetMediaPubKey(rid, uid, mid)
		nid := redis.Get(uKey)

		pub := util.Map("rid", rid, "uid", uid, "mid", mid, "nid", nid, "minfo", util.Unmarshal(minfo))
		pubs = append(pubs, pub)
	}

	resp := util.Map("errorCode", 0, "rid", rid, "pubs", pubs)
	log.Infof("getMediaPubs: resp=%v", resp)
	return resp, nil
}

// parseMediaKey 分析key "/media/rid/" + rid + "/uid/" + uid + "/mid/" + mid
func parseMediaKey(key string) (string, string, error) {
	arr := strings.Split(key, "/")
	if len(arr) < 7 {
		return "", "", fmt.Errorf("Can‘t parse mediainfo; [%s]", key)
	}

	mid := arr[7]
	uid := arr[5]
	return mid, uid, nil
}

/*
	"method", proto.BizToIslbPeerLive, "rid", rid, "uid", uid
*/
// getPeerLive 获取peer存活状态
func getPeerLive(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	// 获取用户的信息
	uKey := proto.GetUserInfoKey(rid, uid)
	info := redis.Get(uKey)
	resp := make(map[string]interface{})
	if info != "" {
		resp = util.Map("errorCode", 0)
	} else {
		resp = util.Map("errorCode", 1)
	}
	return resp, nil
}
