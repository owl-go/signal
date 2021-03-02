package node

import (
	"fmt"
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"strings"
	"time"

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
			case proto.DistToIslbLoginin:
				result, err = clientloginin(data)
			case proto.DistToIslbLoginOut:
				result, err = clientloginout(data)
			case proto.DistToIslbPeerHeart:
				result, err = clientPeerHeart(data)
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
			case proto.BizToIslbGetMediaInfo:
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
	"method", proto.DistToIslbLoginin, "uid", uid, "nid", nid
*/
// clientloginin 有人登录到dist服务器
func clientloginin(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	uid := util.Val(data, "uid")
	dist := util.Val(data, "nid")
	// 获取用户信息保存的key
	uKey := proto.GetUserDistKey(uid)
	// 写入key值
	err := redis.Set(uKey, dist, redisShort)
	if err != nil {
		log.Errorf("redis.Set clientloginin err = %v", err)
	}
	return util.Map(), nil
}

/*
	"method", proto.DistToIslbLoginOut, "uid", uid, "nid", nid
*/
// clientloginin 有人退录到dist服务器
func clientloginout(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
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
	"method", proto.DistToIslbPeerHeart, "uid", uid, "nid", nid
*/
// clientPeerHeart 有人发送心跳到dist服务器,更新key的时间
func clientPeerHeart(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
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
	broadcaster.Say(proto.IslbToBizOnJoin, util.Map("rid", rid, "uid", uid, "info", info))
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
	"method", proto.BizToIslbOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "tracks", tracks, "nid", nid
*/
// streamAdd 有人发布流
func streamAdd(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")
	nid := util.Val(data, "nid")

	// 获取用户发布的流信息
	ukey := proto.GetMediaInfoKey(rid, uid, mid)
	// 写入key值
	tracks := data["tracks"].(map[string]interface{})
	for msid, track := range tracks {
		var infos []proto.TrackInfo
		for _, tinfo := range track.([]interface{}) {
			tmp := tinfo.(map[string]interface{})
			infos = append(infos, proto.TrackInfo{
				ID:      tmp["id"].(string),
				Type:    tmp["type"].(string),
				Ssrc:    int(tmp["ssrc"].(float64)),
				Payload: int(tmp["pt"].(float64)),
				Codec:   tmp["codec"].(string),
				Fmtp:    tmp["fmtp"].(string),
			})
		}
		field, value, err := proto.MarshalTrackField(msid, infos)
		if err != nil {
			log.Errorf("MarshalTrackField: %v ", err)
			continue
		}
		log.Infof("SetTrackField: mkey, field, value = %s, %s, %s", ukey, field, value)
		err = redis.HSet(ukey, field, value)
		if err != nil {
			log.Errorf("redis.HSet streamAdd err = %v", err)
		}
		redis.Expire(ukey, redisKeyTTL)
	}
	// 获取用户发布流对应的sfu信息
	ukey = proto.GetMediaPubKey(rid, uid, mid)
	// 写入key值
	err := redis.Set(ukey, nid, redisKeyTTL)
	if err != nil {
		log.Errorf("redis.Set streamAdd err = %v", err)
	}
	// 生成resp对象
	broadcaster.Say(proto.IslbToBizOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "tracks", tracks, "nid", nid))
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
	dataTmp := util.Val(data, "data")

	broadcaster.Say(proto.IslbToBizBroadcast, util.Map("rid", rid, "uid", uid, "data", dataTmp))
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
	resp := make(map[string]interface{})
	if nid != "" {
		resp = util.Map("errorCode", 0, "rid", rid, "mid", mid, "nid", nid)
	} else {
		resp = util.Map("errorCode", 1, "rid", rid, "mid", mid)
	}
	return resp, nil
}

/*
	"method", proto.BizToIslbGetMediaInfo, "rid", rid, "mid", mid
*/
// getMediaInfo 获取指定mid对应的流的信息
func getMediaInfo(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	mid := util.Val(data, "mid")
	uid := proto.GetUIDFromMID(mid)
	// 获取用户发布的流信息
	uKey := proto.GetMediaInfoKey(rid, uid, mid)
	fields := redis.HGetAll(uKey)
	tracks := make(map[string][]proto.TrackInfo)
	for key, value := range fields {
		if strings.HasPrefix(key, "track/") {
			msid, infos, err := proto.UnmarshalTrackField(key, value)
			if err != nil {
				log.Errorf("getMediaInfo err = %s", err.Error())
			}
			log.Infof("msid => %s, tracks => %v\n", msid, infos)
			tracks[msid] = *infos
		}
	}
	resp := make(map[string]interface{})
	if len(tracks) != 0 {
		resp = util.Map("errorCode", 0, "rid", rid, "mid", mid, "tracks", tracks)
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

		sfu := redis.Get(proto.GetMediaPubKey(rid, uid, mid))
		trackFields := redis.HGetAll(key)

		tracks := make(map[string][]proto.TrackInfo)
		for key, value := range trackFields {
			if strings.HasPrefix(key, "track/") {
				msid, infos, err := proto.UnmarshalTrackField(key, value)
				if err != nil {
					log.Errorf("getMediaPubs err = %s", err.Error())
				}
				log.Infof("msid => %s, tracks => %v\n", msid, infos)
				tracks[msid] = *infos
			}
		}
		pub := util.Map("rid", rid, "uid", uid, "mid", mid, "nid", sfu, "tracks", tracks)
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
