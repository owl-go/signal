package proto

import (
	"fmt"
	"testing"
)

func TestKeyBuildAndParse(t *testing.T) {
	key := BuildMediaInfoKey("dc1", "room1", "sfu1", "mid1")

	if key != "dc1/room1/sfu1/media/pub/mid1" {
		t.Error("MediaInfo key not match")
	}
	fmt.Println(key)
	// dc1/room1/sfu1/media/pub/mid1

	minfo, err := ParseMediaInfo(key)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(minfo)
	// &{dc1 room1 sfu1 mid1 }

	key = BuildUserInfoKey("dc1", "room1", "user1")
	if key != "dc1/room1/user/info/user1" {
		t.Error("UserInfo key not match")
	}
	fmt.Println(key)
	// dc1/room1/user/info/user1

	uinfo, _ := ParseUserInfo(key)
	fmt.Println(uinfo)
	// &{dc1 room1 user1}
}

func TestMarshal(t *testing.T) {
	var tracks []TrackInfo
	tracks = append(tracks, TrackInfo{Ssrc: 3694449886, Payload: 96, Type: "audio", ID: "aid"})
	key, value, err := MarshalTrackField("msidxxxxxx", tracks)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("TrackField: key = %s => %s\n", key, value)
	// TrackField: key = track/msidxxxxxx => [{"id":"aid","ssrc":3694449886,"pt":96,"type":"audio","codec":"","fmtp":""}]

	key, value, err = MarshalNodeField(NodeInfo{Name: "sfu001", ID: "uuid-xxxxx-xxxx", Type: "origin"})
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("NodeField: key = %s => %s\n", key, value)
	// NodeField: key = node/uuid-xxxxx-xxxx => {"name":"sfu001","id":"uuid-xxxxx-xxxx","type":"origin"}
}

func TestUnMarshal(t *testing.T) {
	node, err := UnmarshalNodeField("node/uuid-xxxxx-xxxx", `{"name": "sfu001", "id": "uuid-xxxxx-xxxx", "type": "origin"}`)
	if err == nil {
		fmt.Printf("node => %v\n", node)
		// node => &{sfu001 uuid-xxxxx-xxxx origin}
	}
	msid, tracks, err := UnmarshalTrackField("track/pion audio", `[{"ssrc": 3694449886, "pt": 111, "type": "audio", "id": "aid"}]`)
	if err != nil {
		t.Errorf("err => %v", err)
	}
	fmt.Printf("msid => %s, tracks => %v\n", msid, tracks)
	// msid => pion audio, tracks => &[{aid 3694449886 111 audio  }]
}
