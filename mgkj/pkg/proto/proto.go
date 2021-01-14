package proto

import (
	"strings"
)

const (
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
