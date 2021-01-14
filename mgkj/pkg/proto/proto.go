package proto

import (
	"encoding/json"
	"fmt"
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

	// IslbGetSfuInfo biz->islb get sfu by mid
	IslbGetSfuInfo = "getSfuInfo"
	// IslbGetMediaInfo biz->islb get media info
	IslbGetMediaInfo = "getMediaInfo"
	// IslbGetMediaPubs biz->islb get all publish
	IslbGetMediaPubs = "getMediaPubs"

	// IslbClientOnJoin biz->islb peer-join
	IslbClientOnJoin = ClientOnJoin
	// IslbClientOnLeave biz->islb peer-leave
	IslbClientOnLeave = ClientOnLeave
	// IslbOnStreamAdd biz->islb stream-add
	IslbOnStreamAdd = ClientOnStreamAdd
	// IslbOnStreamRemove biz->islb stream-remove
	IslbOnStreamRemove = ClientOnStreamRemove
	// IslbOnBroadcast biz->islb broadcast
	IslbOnBroadcast = ClientBroadcast
)

/*
media
dc/room1/media/pub/${mid}

node1 origin
node2 shadow
msid  [{ssrc: 1234, pt: 111, type:audio}]
msid  [{ssrc: 5678, pt: 96, type:video}]
*/

func BuildMediaInfoKey(dc string, rid string, nid string, mid string) string {
	strs := []string{dc, rid, nid, "media", "pub", mid}
	return strings.Join(strs, "/")
}

type MediaInfo struct {
	DC  string //Data Center ID
	RID string //Room ID
	NID string //Node ID
	MID string //Media ID
	UID string //User ID
}

// dc1/room1/sfu-tU2GInE5Lfuc/media/pub/7e97c1e8-c80a-4c69-81b0-27efc83e6120#NYYQLV
func ParseMediaInfo(key string) (*MediaInfo, error) {
	var info MediaInfo
	arr := strings.Split(key, "/")
	if len(arr) != 6 {
		return nil, fmt.Errorf("Can‘t parse mediainfo; [%s]", key)
	}
	info.DC = arr[0]
	info.RID = arr[1]
	info.NID = arr[2]
	info.MID = arr[5]
	arr = strings.Split(info.MID, "#")
	if len(arr) == 2 {
		info.UID = arr[0]
	}
	return &info, nil
}

/*
user
/dc/room1/user/info/${uid}
info {name: "Guest"}
*/

func BuildUserInfoKey(dc string, rid string, uid string) string {
	strs := []string{dc, rid, "user", "info", uid}
	return strings.Join(strs, "/")
}

type UserInfo struct {
	DC  string
	RID string
	UID string
}

func ParseUserInfo(key string) (*UserInfo, error) {
	var info UserInfo
	arr := strings.Split(key, "/")
	if len(arr) != 5 {
		return nil, fmt.Errorf("Can‘t parse userinfo; [%s]", key)
	}
	info.DC = arr[0]
	info.RID = arr[1]
	info.UID = arr[4]
	return &info, nil
}

type NodeInfo struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Type string `json:"type"` // origin | shadow
}

func MarshalNodeField(node NodeInfo) (string, string, error) {
	value, err := json.Marshal(node)
	if err != nil {
		return "node/" + node.ID, "", fmt.Errorf("Marshal: %v", err)
	}
	return "node/" + node.ID, string(value), nil
}

func UnmarshalNodeField(key string, value string) (*NodeInfo, error) {
	var node NodeInfo
	if err := json.Unmarshal([]byte(value), &node); err != nil {
		return nil, fmt.Errorf("Unmarshal: %v", err)
	}
	return &node, nil
}

type TrackInfo struct {
	ID      string `json:"id"`
	Ssrc    int    `json:"ssrc"`
	Payload int    `json:"pt"`
	Type    string `json:"type"`
	Codec   string `json:"codec"`
	Fmtp    string `json:"fmtp"`
}

func MarshalTrackField(id string, infos []TrackInfo) (string, string, error) {
	str, err := json.Marshal(infos)
	if err != nil {
		return "track/" + id, "", fmt.Errorf("Marshal: %v", err)
	}
	return "track/" + id, string(str), nil
}

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

// GetUIDFromMID 从mid中获取uid
func GetUIDFromMID(mid string) string {
	return strings.Split(mid, "#")[0]
}

// GetUserInfoKey 获取用户信息保存的key
func GetUserInfoKey(rid string) string {
	return rid + "/user/info"
}

// GetPubMediaKey 获取用户发布流信息保存的key
func GetPubMediaKey(rid, uid string) string {
	return rid + "/media/pub/" + uid
}
