package node

import (
	"encoding/json"
	"fmt"
	"strings"

	nprotoo "github.com/gearghost/nats-protoo"

	"signal/pkg/proto"
	"signal/util"
)

// 接收biz消息处理
func handleRPCRequest(rpcID string) {
	logger.Infof(fmt.Sprintf("islb.handleRequest: rpcID=%s", rpcID), "rpcid", rpcID)
	nats.OnRequest(rpcID, handleRpcMsg)
}

// 处理rpc请求
func handleRpcMsg(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	defer util.Recover("islb.handleRPCRequest")
	logger.Infof(fmt.Sprintf("islb.handleRPCRequest recv request=%v", request))

	method := request["method"].(string)
	data := request["data"].(map[string]interface{})
	var result map[string]interface{}
	err := &nprotoo.Error{Code: 400, Reason: fmt.Sprintf("Unkown method [%s]", method)}

	/* 处理和biz服务器通信 */
	switch method {
	case proto.BizToIslbOnJoin:
		result, err = clientJoin(data)
	case proto.BizToIslbOnLeave:
		result, err = clientLeave(data)
	case proto.BizToIslbKeepAlive:
		result, err = keepalive(data)
	case proto.BizToIslbGetBizInfo:
		result, err = getBizByUid(data)

	case proto.BizToIslbOnStreamAdd:
		result, err = streamAdd(data)
	case proto.BizToIslbOnStreamRemove:
		result, err = streamRemove(data)
	case proto.BizToIslbGetSfuInfo:
		result, err = getSfuByMid(data)

	case proto.BizToIslbBroadcast:
		result, err = broadcast(data)
	case proto.BizToIslbGetRoomUsers:
		result, err = getRoomUsers(data)
	case proto.BizToIslbGetMediaPubs:
		result, err = getMediaPubs(data)
	case proto.IssrToIslbStoreFailedStreamState:
		result, err = pushFailedStreamState(data)
	case proto.IssrToIslbGetFailedStreamState:
		result, err = popFailedStreamState(data)
	case proto.BizToIslbGetMcuInfo:
		result, err = getMcuInfo(data)
	case proto.BizToIslbSetMcuInfo:
		result, err = setMcuInfo(data)
	case proto.BizToIslbGetMediaInfo:
		result, err = getMediaInfo(data)
	}
	if err != nil {
		reject(err.Code, err.Reason)
	} else {
		accept(result)
	}
}

/*
	"method", proto.BizToIslbOnJoin, "rid", rid, "uid", uid, "nid", nid, "info", info
*/
// 有人加入房间
func clientJoin(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	nid := util.Val(data, "nid")
	info := util.Val(data, "info")
	// 获取用户的服务器信息
	uKey := proto.GetUserNodeKey(rid, uid)
	err := redis.Set(uKey, nid, redisShort)
	if err != nil {
		logger.Errorf(fmt.Sprintf("islb.clientJoin redis.Set err=%v", err), "rid", rid, "uid", uid)
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("join err=%v", err)}
	}
	// 获取用户的信息
	uKey = proto.GetUserInfoKey(rid, uid)
	err = redis.Set(uKey, info, redisShort)
	if err != nil {
		logger.Errorf(fmt.Sprintf("islb.clientJoin redis.Set err=%v", err), "rid", rid, "uid", uid)
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("join err=%v", err)}
	}
	// 生成resp对象
	broadcaster.Say(proto.IslbToBizOnJoin, util.Map("rid", rid, "uid", uid, "nid", nid, "info", util.Unmarshal(data["info"].(string))))
	return util.Map(), nil
}

/*
	"method", proto.BizToIslbOnLeave, "rid", rid, "uid", uid
*/
// 有人退出房间
func clientLeave(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	// 获取用户的服务器信息
	uKey := proto.GetUserNodeKey(rid, uid)
	ukeys := redis.Keys(uKey)
	if len(ukeys) > 0 {
		err := redis.Del(uKey)
		if err != nil {
			logger.Errorf(fmt.Sprintf("islb.clientLeave redis.Del err=%v", err), "rid", rid, "uid", uid)
		}
	}
	// 获取用户的信息
	uKey = proto.GetUserInfoKey(rid, uid)
	ukeys = redis.Keys(uKey)
	if len(ukeys) > 0 {
		err := redis.Del(uKey)
		if err != nil {
			logger.Errorf(fmt.Sprintf("islb.clientLeave redis.Del err=%v", err), "rid", rid, "uid", uid)
		}
	}
	broadcaster.Say(proto.IslbToBizOnLeave, util.Map("rid", rid, "uid", uid))
	return util.Map(), nil
}

/*
	"method", proto.BizToIslbKeepAlive, "rid", rid, "uid", uid
*/
// 保活处理
func keepalive(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	// 获取用户的服务器信息
	uKey := proto.GetUserNodeKey(rid, uid)
	err := redis.Expire(uKey, redisShort)
	if err != nil {
		logger.Errorf(fmt.Sprintf("islb.keepalive redis.Set err=%v", err), "rid", rid, "uid", uid)
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("keepalive err=%v", err)}
	}
	// 获取用户的信息
	uKey = proto.GetUserInfoKey(rid, uid)
	err = redis.Expire(uKey, redisShort)
	if err != nil {
		logger.Errorf(fmt.Sprintf("islb.keepalive redis.Set err=%v", err), "rid", rid, "uid", uid)
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("keepalive err=%v", err)}
	}
	return util.Map(), nil
}

/*
	"method", proto.BizToIslbGetBizInfo, "rid", rid, "uid", uid
*/
// 获取uid指定的biz节点信息
func getBizByUid(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	// 获取用户的服务器信息
	uKey := proto.GetUserNodeKey(rid, uid)
	ukeys := redis.Keys(uKey)
	if len(ukeys) > 0 {
		nid := redis.Get(uKey)
		return util.Map("rid", rid, "nid", nid), nil
	} else {
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("can't find peer info by key:%s", uKey)}
	}
}

/*
	"method", proto.BizToIslbOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "nid", nid, "minfo", minfo
*/
// 有人发布流
func streamAdd(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")
	nid := util.Val(data, "nid")
	minfo := util.Val(data, "minfo")
	// 获取用户发布的流信息
	ukey := proto.GetMediaInfoKey(rid, uid, mid)
	err := redis.Set(ukey, minfo, redisKeyTTL)
	if err != nil {
		logger.Errorf(fmt.Sprintf("islb.streamAdd redis.Set err=%v", err), "rid", rid, "uid", uid, "mid", mid)
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("streamAdd err=%v", err)}
	}
	// 获取用户发布流对应的sfu信息
	ukey = proto.GetMediaPubKey(rid, uid, mid)
	err = redis.Set(ukey, nid, redisKeyTTL)
	if err != nil {
		logger.Errorf(fmt.Sprintf("islb.streamAdd redis.Set err=%v", err), "rid", rid, "uid", uid, "mid", mid)
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("streamAdd err=%v", err)}
	}
	// 生成resp对象
	broadcaster.Say(proto.IslbToBizOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "nid", nid, "minfo", data["minfo"]))
	return util.Map(), nil
}

/*
	"method", proto.BizToIslbOnStreamRemove, "rid", rid, "uid", uid, "mid", ""
*/
// 有人取消发布流
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
				logger.Errorf(fmt.Sprintf("islb.streamRemove media redis.Del err=%v", err), "rid", rid, "uid", uid)
			}
		}
		ukey = "/pub/rid/" + rid + "/uid/" + uid + "/mid/*"
		ukeys = redis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			arr := strings.Split(key, "/")
			mid := arr[7]
			// 删除key值
			err := redis.Del(ukey)
			if err != nil {
				logger.Errorf(fmt.Sprintf("islb.streamRemove pub redis.Del err=%v", err), "rid", rid, "uid", uid)
			}
			broadcaster.Say(proto.IslbToBizOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
		}
	} else {
		// 获取用户发布的流信息
		ukey = proto.GetMediaInfoKey(rid, uid, mid)
		ukeys := redis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			// 删除key值
			err := redis.Del(ukey)
			if err != nil {
				logger.Errorf(fmt.Sprintf("islb.streamRemove media redis.Del err=%v", err), "rid", rid, "uid", uid, "mid", mid)
			}
		}
		// 获取用户发布流对应的sfu信息
		ukey = proto.GetMediaPubKey(rid, uid, mid)
		ukeys = redis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			arr := strings.Split(key, "/")
			mid := arr[7]
			// 删除key值
			err := redis.Del(ukey)
			if err != nil {
				logger.Errorf(fmt.Sprintf("islb.streamRemove pub redis.Del err=%v", err), "rid", rid, "uid", uid, "mid", mid)
			}
			broadcaster.Say(proto.IslbToBizOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
		}
	}
	return util.Map(), nil
}

/*
	"method", proto.BizToIslbGetSfuInfo, "rid", rid, "mid", mid
*/
// 获取mid指定对应的sfu节点
func getSfuByMid(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	mid := util.Val(data, "mid")
	uid := proto.GetUIDFromMID(mid)
	// 获取用户发布流对应的sfu信息
	uKey := proto.GetMediaPubKey(rid, uid, mid)
	ukeys := redis.Keys(uKey)
	if len(ukeys) > 0 {
		nid := redis.Get(uKey)
		return util.Map("rid", rid, "nid", nid), nil
	} else {
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("can't find sfu node by mid:%s", uKey)}
	}
}

/*
	"method", proto.BizToIslbBroadcast, "rid", rid, "uid", uid, "data", data
*/
// 发送广播
func broadcast(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	broadcaster.Say(proto.IslbToBizBroadcast, util.Map("rid", rid, "uid", uid, "data", data["data"]))
	return util.Map(), nil
}

/*
	"method", proto.BizToIslbGetRoomUsers, "rid", rid, "uid", uid
*/
// 获取房间里所有人
func getRoomUsers(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	id := util.Val(data, "uid")
	// 获取用户的信息
	pubs := getUserMedias(rid, id)
	users := make([]map[string]interface{}, 0)
	uKey := "/user/rid/" + rid + "/uid/*"
	ukeys := redis.Keys(uKey)
	for _, key := range ukeys {
		// 去掉指定的uid
		arr := strings.Split(key, "/")
		uid := arr[5]
		if uid == id {
			continue
		}

		info := redis.Get(key)
		uKey := proto.GetUserNodeKey(rid, uid)
		nid := redis.Get(uKey)

		media := make([]map[string]interface{}, 0)
		for _, pub := range pubs {
			if uid == pub["uid"].(string) {
				if pub["mid"].(string) != "" {
					media = append(media, pub)
				}
			}
		}

		if len(media) > 0 {
			for _, stream := range media {
				user := util.Map("uid", uid, "nid", nid, "info", util.Unmarshal(info), "media", stream)
				users = append(users, user)
			}
		} else {
			user := util.Map("uid", uid, "nid", nid, "info", util.Unmarshal(info), "media", util.Map())
			users = append(users, user)
		}
	}
	// 返回
	resp := util.Map("rid", rid, "users", users)
	logger.Infof(fmt.Sprintf("islb.getRoomUsers resp=%v ", resp), "rid", rid)
	return resp, nil
}

// 获取房间所有人的发布流
func getUserMedias(rid, id string) []map[string]interface{} {
	// 找到所有用户流信息的key
	var pubs []map[string]interface{}
	uKey := "/media/rid/" + rid + "/uid/*"
	ukeys := redis.Keys(uKey)
	for _, key := range ukeys {
		arr := strings.Split(key, "/")
		uid := arr[5]
		mid := arr[7]
		if uid == id {
			continue
		}

		minfo := redis.Get(key)
		uKey := proto.GetMediaPubKey(rid, uid, mid)
		nid := redis.Get(uKey)

		pub := util.Map("uid", uid, "mid", mid, "nid", nid, "minfo", util.Unmarshal(minfo))
		pubs = append(pubs, pub)
	}
	// 返回
	logger.Infof(fmt.Sprintf("islb.getUserMedias resp=%v ", pubs), "rid", rid)
	return pubs
}

/*
	"method", proto.BizToIslbGetMediaPubs, "rid", rid, "uid", uid
*/
// 获取房间所有人的发布流
func getMediaPubs(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	id := util.Val(data, "uid")
	// 找到保存用户流信息的key
	pubs := make([]map[string]interface{}, 0)
	uKey := "/media/rid/" + rid + "/uid/*"
	ukeys := redis.Keys(uKey)
	for _, key := range ukeys {
		arr := strings.Split(key, "/")
		uid := arr[5]
		mid := arr[7]
		if uid == id {
			continue
		}

		minfo := redis.Get(key)
		uKey := proto.GetMediaPubKey(rid, uid, mid)
		nid := redis.Get(uKey)

		pub := util.Map("uid", uid, "mid", mid, "nid", nid, "minfo", util.Unmarshal(minfo))
		pubs = append(pubs, pub)
	}
	// 返回
	resp := util.Map("rid", rid, "pubs", pubs)
	logger.Infof(fmt.Sprintf("islb.getMediaPubs resp=%v ", resp), "rid", rid)
	return resp, nil
}

// 存储失败拉流数据
func pushFailedStreamState(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	//生成key
	sKey := proto.GetFailedStreamStateKey()
	// 写入key值
	state, err := json.Marshal(data)
	if err != nil {
		logger.Errorf(fmt.Sprintf("islb.pushFailedStreamState json marshal err=%v", err), "rid", rid, "uid", uid)
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("json marshal err:%v", err)}
	}

	err = redis1.RPush(sKey, string(state))
	if err != nil {
		logger.Errorf(fmt.Sprintf("biz.pushFailedStreamState redis.Set stream state err=%v", err), "rid", rid, "uid", uid)
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("redis.Set err=%v", err)}
	} else {
		return util.Map(), nil
	}
}

// 获取失败拉流记录
func popFailedStreamState(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	sKey := proto.GetFailedStreamStateKey()
	length := redis1.LLen(sKey)
	if length > 20 {
		length = 20
	}
	failures := make([]string, 0)
	for i := int64(0); i < length; i++ {
		failure := redis1.LPop(sKey)
		if failure != "" {
			failures = append(failures, failure)
		}
	}
	return util.Map("failures", failures), nil
}

// 根据rid查询对应mcu
func getMcuInfo(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	key := proto.GetMcuInfoKey(rid)
	// 获取房间对应的mcu信息
	nid := redis.Get(key)
	if nid == "" {
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("can't get mcu info by rid:%s", rid)}
	}
	return util.Map("rid", rid, "nid", nid), nil
}

// 设置rid跟mcu绑定关系
func setMcuInfo(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	nid := util.Val(data, "nid")
	key := proto.GetMcuInfoKey(rid)
	mcu := redis.Get(key)
	if mcu != "" {
		return util.Map("nid", mcu), nil
	}
	err := redis.Set(key, nid, redisKeyTTL)
	if err != nil {
		return nil, &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("redis.Set err:%v", err)}
	}
	return util.Map("nid", nid), nil
}

func getMediaInfo(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")
	ukey := proto.GetMediaInfoKey(rid, uid, mid)
	minfo := redis.Get(ukey)
	if minfo == "" {
		return util.Map(), &nprotoo.Error{Code: -1, Reason: fmt.Sprintf("minfo doesn't exist:%s", ukey)}
	}
	return util.Map("minfo", util.Unmarshal(minfo)), nil
}
