package proto

import (
	"strings"
)

const (

	//IssrToIslbStoreFailedStreamState Issr -> Islb 存储失败拉流数据
	IssrToIslbStoreFailedStreamState = "storefailedstreamstate"
	//IssrToIslbGetFailedStreamState Issr -> Islb 获取失败拉流记录
	IssrToIslbGetFailedStreamState = "getfailedstreamstate"
	//BizToIssrReportStreamState Biz -> Issr 发送拉流信息到Issr
	BizToIssrReportStreamState = "reportstreamstate"

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

	// ClientToBizBroadcast C->Biz 发送广播给房间里所有用户
	ClientToBizBroadcast = "broadcast"
	// ClientToBizGetRoomUsers C->Biz 获取房间所有用户信息
	ClientToBizGetRoomUsers = "listusers"
	// ClientToBizStartLivestream C->Biz 开始直播
	ClientToBizStartLivestream = "startlivestream"
	// ClientToBizStopLivestream C->Biz 取消直播
	ClientToBizStopLivestream = "stoplivestream"

	// BizToClientOnJoin biz->C 有人加入房间
	BizToClientOnJoin = "peer-join"
	// BizToClientOnLeave biz->C 有人离开房间
	BizToClientOnLeave = "peer-leave"
	// BizToClientOnStreamAdd biz->C 有人发布流
	BizToClientOnStreamAdd = "stream-add"
	// BizToClientOnStreamRemove biz->C 有人取消发布流
	BizToClientOnStreamRemove = "stream-remove"
	// BizToClientBroadcast biz->C 有人发送广播
	BizToClientBroadcast = "broadcast"
	// BizToBizOnKick biz->biz 有人被服务器踢下线
	BizToBizOnKick    = "peer-kick"
	BizToClientOnKick = "peer-kick"
	//BizToIslbClearMcuBinding biz->islb 清理mcu binding
	BizToIslbClearMcuBinding = "clearmcubinding"

	/*
		biz与sfu服务器通信
	*/

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
	// SfuToBizOnStreamRemove Sfu->Biz Sfu通知biz流被移除
	SfuToBizOnStreamRemove = "sfu-stream-remove"
	//BizToSfuSubscribeRTP Biz->Sfu 请求sfu创建offer
	BizToSfuSubscribeRTP = "subscribertp"

	/*
		biz与mcu服务器通信
	*/

	//BizToMcuPublishRTP Biz->Mcu 转发sfu offer到Mcu
	BizToMcuPublishRTP = "publishrtp"
	//BizToMcuUnpublish Biz->Mcu 停止mcu推流
	BizToMcuUnpublish = "unpublish"

	/*
		biz与islb服务器通信
	*/

	// BizToIslbOnJoin biz->islb 有人加入房间
	BizToIslbOnJoin = "peer-join"
	// BizToIslbOnLeave biz->islb 有人离开房间
	BizToIslbOnLeave = "peer-leave"
	// BizToIslbOnStreamAdd biz->islb 有人发布流
	BizToIslbOnStreamAdd = "stream-add"
	// BizToIslbOnStreamRemove biz->islb 有人取消发布流
	BizToIslbOnStreamRemove = "stream-remove"

	// BizToIslbPeerLive biz->islb 获取Peer是否在线
	BizToIslbPeerLive = "getPeerLive"
	// BizToIslbKeepAlive biz->islb 保活
	BizToIslbKeepAlive = "keepalive"
	// BizToIslbBroadcast biz->islb 发送广播
	BizToIslbBroadcast = "broadcast"
	// BizToIslbGetBizInfo biz->islb 根据uid查询对应的biz
	BizToIslbGetBizInfo = "getBizInfo"
	// BizToIslbGetSfuInfo biz->islb 根据mid查询对应的sfu
	BizToIslbGetSfuInfo = "getSfuInfo"
	// BizToIslbGetRoomUsers biz->islb 获取所有用户信息
	BizToIslbGetRoomUsers = "getRoomUsers"
	// BizToIslbGetMediaPubs biz->islb 获取所有的发布流信息
	BizToIslbGetMediaPubs = "getMediaPubs"
	//BizToIslbGetMcuInfo biz->islb 根据rid查询对应mcu
	BizToIslbGetMcuInfo = "getMcuInfo"
	//BizToIslbSetMcuInfo biz->islb 设置rid跟mcu绑定关系
	BizToIslbSetMcuInfo = "setMcuInfo"
	//BizToIslbGetMediaInfo biz->islb 根据rid,uid,mid获取media info
	BizToIslbGetMediaInfo = "getMediaInfo"

	// IslbToBizOnJoin islb->biz 有人加入房间
	IslbToBizOnJoin = BizToClientOnJoin
	// IslbToBizOnLeave islb->biz 有人离开房间
	IslbToBizOnLeave = BizToClientOnLeave
	// IslbToBizOnStreamAdd islb->biz 有人发布流
	IslbToBizOnStreamAdd = BizToClientOnStreamAdd
	// IslbToBizOnStreamRemove islb->biz 有人取消发布流
	IslbToBizOnStreamRemove = BizToClientOnStreamRemove
	// IslbToBizBroadcast islb->biz 有人发送广播
	IslbToBizBroadcast = ClientToBizBroadcast
	//McuToBizOnRoomRemove mcu->biz mcu房间移除通知
	McuToBizOnRoomRemove = "mcu-room-remove"
)

// GetUIDFromMID 从mid中获取uid
func GetUIDFromMID(mid string) string {
	return strings.Split(mid, "#")[0]
}

// GetUserInfoKey 获取用户的信息
func GetUserInfoKey(rid, uid string) string {
	return "/user/rid/" + rid + "/uid/" + uid
}

// GetUserNodeKey 获取用户的服务器信息
func GetUserNodeKey(rid, uid string) string {
	return "/node/rid/" + rid + "/uid/" + uid
}

// GetMediaInfoKey 获取用户发布的流信息
func GetMediaInfoKey(rid, uid, mid string) string {
	return "/media/rid/" + rid + "/uid/" + uid + "/mid/" + mid
}

// GetMediaPubKey 获取用户发布流对应的sfu信息
func GetMediaPubKey(rid, uid, mid string) string {
	return "/pub/rid/" + rid + "/uid/" + uid + "/mid/" + mid
}

// GetFailedStreamStateKey 获取报告失败拉流状态信息key
func GetFailedStreamStateKey() string {
	return "/zx/report/failure"
}

// GetMcuInfoKey 获取MCU节点 key
func GetMcuInfoKey(rid string) string {
	return "/mcu/rid/" + rid
}

// GetUserLockKey 获取UserLock key
func GetUserLockKey(rid, uid string) string {
	return "/lock/" + rid + "/" + uid
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
