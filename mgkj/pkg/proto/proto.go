package proto

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	/*
		客户端和dist服务器之间的通信
	*/

	// ClientToDistLoginin C->dist 登录
	ClientToDistLoginin = "loginin"
	// ClientToDistLoginOut C->dist 退录
	ClientToDistLoginOut = "loginout"
	// ClientToDistHeart C->dist 心跳
	ClientToDistHeart = "heart"
	// ClientToDistCall C->dist 呼叫
	ClientToDistCall = "call"
	// ClientToDistAnswer C->dist 应答
	ClientToDistAnswer = "answer"
	// ClientToDistReject C->dist 拒绝
	ClientToDistReject = "reject"

	// DistToClientCall dist->C 通知端来电
	DistToClientCall = ClientToDistCall
	// DistToDistCall dist->dist 通知端来电
	DistToDistCall = DistToClientCall
	// DistToClientAnswer dist-> 通知端应答
	DistToClientAnswer = ClientToDistAnswer
	// DistToDistAnswer dist->dist 通知端应答
	DistToDistAnswer = DistToClientAnswer
	// DistToClientReject dist-> 通知端拒绝
	DistToClientReject = ClientToDistReject
	// DistToDistReject dist->dist 通知端拒绝
	DistToDistReject = DistToClientReject

	// DistToIslbLoginin dist->islb 上线
	DistToIslbLoginin = ClientToDistLoginin
	// DistToIslbLoginOut dist->islb 下线
	DistToIslbLoginOut = ClientToDistLoginOut
	// DistToIslbPeerHeart dist->islb 心跳
	DistToIslbPeerHeart = ClientToDistHeart
	// DistToIslbPeerInfo dist->islb 获取Peer在哪个Dist服务器
	DistToIslbPeerInfo = "getPeerDist"
	// IslbToDistPeerInfo islb->dist islb返回peer在哪个dist服务器
	IslbToDistPeerInfo = DistToIslbPeerInfo

	/*
		客户端与biz服务器通信
	*/

	// ClientToBizJoin C->Biz 加入会议
	ClientToBizJoin = "join"
	// ClientToBizLeave C->Biz 离开会议
	ClientToBizLeave = "leave"
	// ClientToBizKeepLive C->Biz 保活
	ClientToBizKeepLive = "keeplive"
	// ClientToBizPublish C->Biz 发布流
	ClientToBizPublish = "publish"
	// ClientToBizUnPublish C->Biz 取消发布流
	ClientToBizUnPublish = "unpublish"
	// ClientToBizSubscribe C->Biz 订阅流
	ClientToBizSubscribe = "subscribe"
	// ClientToBizUnSubscribe C->Biz 取消订阅流
	ClientToBizUnSubscribe = "unsubscribe"
	// ClientToBizTrickleICE C->Biz 发送ice数据
	ClientToBizTrickleICE = "trickle"
	// ClientToBizBroadcast C->Biz 发送广播
	ClientToBizBroadcast = "broadcast"

	// BizToClientOnJoin biz->C 有人加入房间
	BizToClientOnJoin = "peer-join"
	// BizToClientOnLeave biz->C 有人离开房间
	BizToClientOnLeave = "peer-leave"
	// BizToClientOnStreamAdd biz->C 有人发布流
	BizToClientOnStreamAdd = "stream-add"
	// BizToClientOnStreamRemove biz->C 有人取消发布流
	BizToClientOnStreamRemove = "stream-remove"
	// BizToClientBroadcast biz->C 有人发送广播
	BizToClientBroadcast = ClientToBizBroadcast

	// BizToSfuPublish Biz->Sfu 发布流
	BizToSfuPublish = "publish"
	// BizToSfuUnPublish Biz->Sfu 取消发布流
	BizToSfuUnPublish = "unpublish"
	// BizToSfuSubscribe Biz->Sfu 订阅流
	BizToSfuSubscribe = "subscribe"
	// BizToSfuUnSubscribe Biz->Sfu 取消订阅流
	BizToSfuUnSubscribe = "unsubscribe"
	// BizToSfuTrickleICE Biz->Sfu 发送ice数据
	BizToSfuTrickleICE = "trickle"

	// SfuToBizPublish Sfu->Biz 发布流返回
	SfuToBizPublish = BizToSfuPublish
	// SfuToBizSubscribe Sfu->Biz 订阅流返回
	SfuToBizSubscribe = BizToSfuSubscribe
	// SfuToBizOnStreamRemove Sfu->Biz Sfu流被移除
	SfuToBizOnStreamRemove = "sfu-stream-remove"

	// BizToIslbOnJoin biz->islb 有人加入房间
	BizToIslbOnJoin = BizToClientOnJoin
	// IslbToBizOnJoin islb->biz 有人加入房间
	IslbToBizOnJoin = BizToClientOnJoin
	// BizToIslbOnLeave biz->islb 有人离开房间
	BizToIslbOnLeave = BizToClientOnLeave
	// IslbToBizOnLeave islb->biz 有人离开房间
	IslbToBizOnLeave = BizToClientOnLeave
	// BizToIslbOnStreamAdd biz->islb 有人发布流
	BizToIslbOnStreamAdd = BizToClientOnStreamAdd
	// IslbToBizOnStreamAdd islb->biz 有人发布流
	IslbToBizOnStreamAdd = BizToClientOnStreamAdd
	// BizToIslbOnStreamRemove biz->islb 有人取消发布流
	BizToIslbOnStreamRemove = BizToClientOnStreamRemove
	// IslbToBizOnStreamRemove islb->biz 有人取消发布流
	IslbToBizOnStreamRemove = BizToClientOnStreamRemove
	// BizToIslbKeepLive biz->islb 保活
	BizToIslbKeepLive = ClientToBizKeepLive
	// BizToIslbGetSfuInfo biz->islb 根据mid查询对应的sfu
	BizToIslbGetSfuInfo = "getSfuInfo"
	// IslbToBizGetSfuInfo islb->biz 返回mid对应的sfu
	IslbToBizGetSfuInfo = BizToIslbGetSfuInfo
	// BizToIslbGetMediaInfo biz->islb 获取mid的流信息
	BizToIslbGetMediaInfo = "getMediaInfo"
	// IslbToBizGetMediaInfo islb->biz 返回mid对应的流信息
	IslbToBizGetMediaInfo = BizToIslbGetMediaInfo
	// BizToIslbGetMediaPubs biz->islb 获取房间内所有的发布流
	BizToIslbGetMediaPubs = "getMediaPubs"
	// IslbToBizGetMediaPubs islb->biz 返回房间内的发布流信息
	IslbToBizGetMediaPubs = BizToIslbGetMediaPubs
	// BizToIslbPeerLive biz->islb 获取Peer是否还存活
	BizToIslbPeerLive = "getPeerLive"
	// IslbToBizPeerLive islb->biz islb返回peer存活状态
	IslbToBizPeerLive = BizToIslbPeerLive
	// BizToIslbBroadcast biz->islb 有人发送广播
	BizToIslbBroadcast = ClientToBizBroadcast
	// IslbToBizBroadcast islb->biz 有人发送广播
	IslbToBizBroadcast = ClientToBizBroadcast
)

// GetUIDFromMID 从mid中获取uid
func GetUIDFromMID(mid string) string {
	return strings.Split(mid, "#")[0]
}

// GetUserDistKey 获取用户连接信息
func GetUserDistKey(uid string) string {
	return "/dist/uid/" + uid
}

// GetUserInfoKey 获取用户的信息
func GetUserInfoKey(rid, uid string) string {
	return "/user/rid/" + rid + "/uid/" + uid
}

// GetMediaInfoKey 获取用户发布的流信息
func GetMediaInfoKey(rid, uid, mid string) string {
	return "/media/rid/" + rid + "/uid/" + uid + "/mid/" + mid
}

// GetMediaPubKey 获取用户发布流对应的sfu信息
func GetMediaPubKey(rid, uid, mid string) string {
	return "/pub/rid/" + rid + "/uid/" + uid + "/mid/" + mid
}

// TrackInfo track信息
type TrackInfo struct {
	ID      string `json:"id"`
	Ssrc    int    `json:"ssrc"`
	Payload int    `json:"pt"`
	Type    string `json:"type"`
	Codec   string `json:"codec"`
	Fmtp    string `json:"fmtp"`
}

// MarshalTrackField 分析
func MarshalTrackField(id string, infos []TrackInfo) (string, string, error) {
	str, err := json.Marshal(infos)
	if err != nil {
		return "track/" + id, "", fmt.Errorf("Marshal: %v", err)
	}
	return "track/" + id, string(str), nil
}

// UnmarshalTrackField 分析
func UnmarshalTrackField(key string, value string) (string, *[]TrackInfo, error) {
	var tracks []TrackInfo
	if err := json.Unmarshal([]byte(value), &tracks); err != nil {
		return "", nil, fmt.Errorf("Unmarshal: %v", err)
	}
	if !strings.Contains(key, "track/") {
		return "", nil, fmt.Errorf("Invalid track failed => %s", key)
	}
	msid := strings.Split(key, "/")[1]
	return msid, &tracks, nil
}
