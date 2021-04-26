package proto

import (
	"strings"
)

const (

	//IssrToIslbReportStreamState Issr -> Islb 记录拉流数据
	IssrToIslbReportStreamState = "report-stream-state"
	//BizToIssrReportStreamState Biz -> Issr 发送拉流信息到Issr
	BizToIssrReportStreamState = IssrToIslbReportStreamState

	/*
		客户端与biz服务器通信
	*/

	// ClientToBizJoin C->Biz 加入会议
	ClientToBizJoin = "join"
	// ClientToBizLeave C->Biz 离开会议
	ClientToBizLeave = "leave"
	// ClientToBizKeepAlive C->Biz 保活
	ClientToBizKeepAlive = "keepalive"
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
	//ClientToBizListusers C->Biz 列出其他所有用户信息
	ClientToBizListusers = "listusers"

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
	BizToIslbKeepLive = ClientToBizKeepAlive
	// BizToIslbGetSfuInfo biz->islb 根据mid查询对应的sfu
	BizToIslbGetSfuInfo = "getSfuInfo"
	//BizToIslbListusers biz->islb 列出其他所有用户信息
	BizToIslbListusers = ClientToBizListusers
	// IslbToBizGetSfuInfo islb->biz 返回mid对应的sfu
	IslbToBizGetSfuInfo = BizToIslbGetSfuInfo
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
	//日志输出
	ToLogsvr = "toLogsvr"
)

// GetUIDFromMID 从mid中获取uid
func GetUIDFromMID(mid string) string {
	return strings.Split(mid, "#")[0]
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

// GetStreamStateKey 获取拉流状态信息key
func GetStreamStateKey(rid, uid, mid string) string {
	return "/ss/rid/" + rid + "/uid/" + uid + "/mid/" + mid
}

// TrackInfo track信息
type TrackInfo struct {
	ID      string `json:"id"`
	Ssrc    uint   `json:"ssrc"`
	Payload int    `json:"pt"`
	Type    string `json:"type"`
	Codec   string `json:"codec"`
	Fmtp    string `json:"fmtp"`
}
