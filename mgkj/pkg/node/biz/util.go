package node

import (
	"fmt"

	"mgkj/pkg/util"

	"github.com/cloudwebrtc/go-protoo/peer"
)

func getMID(uid string) string {
	return fmt.Sprintf("%s#%s", uid, util.RandStr(6))
}

func getKeepAliveID(mid string, ssrc uint32) string {
	return fmt.Sprintf("%s#%d", mid, ssrc)
}

func invalid(msg map[string]interface{}, key string, reject peer.RejectFunc) bool {
	val := util.Val(msg, key)
	if val == "" {
		switch key {
		case "uid":
			reject(codeUIDErr, codeStr(codeUIDErr))
			return true
		case "rid":
			reject(codeRoomErr, codeStr(codeRoomErr))
			return true
		case "mid":
			reject(codeMIDErr, codeStr(codeMIDErr))
			return true
		case "jsep":
			reject(codeJsepErr, codeStr(codeJsepErr))
			return true
		case "sdp":
			reject(codeSDPErr, codeStr(codeSDPErr))
			return true
		}
	}
	return false
}
