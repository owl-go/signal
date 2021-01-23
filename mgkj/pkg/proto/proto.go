package proto

import (
	"strings"
)

const (
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

	// ClientToBizJoin C->Biz 加入会议
	ClientToBizJoin = "join"
	// ClientToBizLeave C->Biz 离开会议
	ClientToBizLeave = "leave"
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
	SfuToBizOnStreamRemove = BizToClientOnStreamRemove

	// BizToIslbOnJoin biz->islb 有人加入房间
	BizToIslbOnJoin = BizToClientOnJoin
	// BizToIslbOnLeave biz->islb 有人离开房间
	BizToIslbOnLeave = BizToClientOnLeave
	// BizToIslbOnStreamAdd biz->islb 有人发布流
	BizToIslbOnStreamAdd = BizToClientOnStreamAdd
	// BizToIslbOnStreamRemove biz->islb 有人取消发布流
	BizToIslbOnStreamRemove = BizToClientOnStreamRemove
	// BizToIslbGetSfuInfo biz->islb 根据mid查询对应的sfu
	BizToIslbGetSfuInfo = "getSfuInfo"
	// BizToIslbGetMediaInfo biz->islb 获取mid的流信息
	BizToIslbGetMediaInfo = "getMediaInfo"
	// BizToIslbGetMediaPubs biz->islb 获取房间内所有的发布流
	BizToIslbGetMediaPubs = "getMediaPubs"
	// BizToIslbBroadcast biz->islb 有人发送广播
	BizToIslbBroadcast = ClientToBizBroadcast

	// IslbToBizOnJoin islb->biz 有人加入房间
	IslbToBizOnJoin = BizToClientOnJoin
	// IslbToBizOnLeave islb->biz 有人离开房间
	IslbToBizOnLeave = BizToClientOnLeave
	// IslbToBizOnStreamAdd islb->biz 有人发布流
	IslbToBizOnStreamAdd = BizToClientOnStreamAdd
	// IslbToBizOnStreamRemove islb->biz 有人取消发布流
	IslbToBizOnStreamRemove = BizToClientOnStreamRemove
	// IslbToBizGetSfuInfo islb->biz 返回mid对应的sfu
	IslbToBizGetSfuInfo = BizToIslbGetSfuInfo
	// IslbToBizGetMediaInfo islb->biz 返回mid对应的流信息
	IslbToBizGetMediaInfo = BizToIslbGetMediaInfo
	// IslbToBizGetMediaPubs islb->biz 返回房间内的发布流信息
	IslbToBizGetMediaPubs = BizToIslbGetMediaPubs
	// IslbToBizBroadcast islb->biz 有人发送广播
	IslbToBizBroadcast = ClientToBizBroadcast
)

// GetUIDFromMID 从mid中获取uid
func GetUIDFromMID(mid string) string {
	return strings.Split(mid, "#")[0]
}

// GetUserDistKey 获取用户连接信息保存的key
func GetUserDistKey(uid string) string {
	return "/dist/" + uid
}

// GetUserInfoKey 获取用户信息保存的key
func GetUserInfoKey(rid, uid string) string {
	return "/user/uid/" + uid + "/rid/" + rid
}

// GetMediaInfoKey 获取媒体信息保存的key
func GetMediaInfoKey(rid, uid, mid string) string {
	return "/media/mid/" + mid + "/uid/" + uid + "/rid/" + rid
}

// GetMediaPubKey 获取发布流服务器信息保存的key
func GetMediaPubKey(rid, uid, mid string) string {
	return "/sfu/mid/" + mid + "/uid/" + uid + "/rid/" + rid
}
