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

	// ClientJoin C->Biz join
	ClientJoin = "join"
	// ClientLeave C->Biz leave
	ClientLeave = "leave"
	// ClientPublish C->Biz publish
	ClientPublish = "publish"
	// ClientUnPublish C->Biz unpublish
	ClientUnPublish = "unpublish"
	// ClientSubscribe C->Biz subscribe
	ClientSubscribe = "subscribe"
	// ClientUnSubscribe C->Biz unsubscribe
	ClientUnSubscribe = "unsubscribe"
	// ClientTrickleICE C->Biz 发送ice
	ClientTrickleICE = "trickle"
	// ClientBroadcast C->Biz broadcast
	ClientBroadcast = "broadcast"

	// ClientOnJoin biz->C peer-join
	ClientOnJoin = "peer-join"
	// ClientOnLeave biz->C peer-leave
	ClientOnLeave = "peer-leave"
	// ClientOnStreamAdd biz->C stream-add
	ClientOnStreamAdd = "stream-add"
	// ClientOnStreamRemove biz->C stream-remove
	ClientOnStreamRemove = "stream-remove"

	// IslbClientOnJoin biz->islb peer-join
	IslbClientOnJoin = ClientOnJoin
	// IslbClientOnLeave biz->islb peer-leave
	IslbClientOnLeave = ClientOnLeave
	// IslbOnStreamAdd biz->islb stream-add
	IslbOnStreamAdd = ClientOnStreamAdd
	// IslbOnStreamRemove biz->islb stream-remove
	IslbOnStreamRemove = ClientOnStreamRemove
	// IslbGetSfuInfo biz->islb get sfu by mid
	IslbGetSfuInfo = "getSfuInfo"
	// IslbGetMediaInfo biz->islb get media info
	IslbGetMediaInfo = "getMediaInfo"
	// IslbGetMediaPubs biz->islb get all publish
	IslbGetMediaPubs = "getMediaPubs"
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
