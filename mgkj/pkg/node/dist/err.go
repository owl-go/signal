package node

import (
	"mgkj/pkg/util"

	"github.com/cloudwebrtc/go-protoo/peer"
)

const (
	codeOK int = -iota
	codeUIDErr
	codeRIDErr
	codeBizErr
	codeSfuErr
	codeIslbErr
	codeDistErr
	codeUnknownErr
)

var codeErr = map[int]string{
	codeOK:         "OK",
	codeUIDErr:     "uid not found",
	codeRIDErr:     "rid not found",
	codeBizErr:     "biz not found",
	codeSfuErr:     "sfu not found",
	codeIslbErr:    "islb not found",
	codeDistErr:    "dist not found",
	codeUnknownErr: "unknown error",
}

func codeStr(code int) string {
	return codeErr[code]
}

var emptyMap = map[string]interface{}{}

func invalid(msg map[string]interface{}, key string, reject peer.RejectFunc) bool {
	val := util.Val(msg, key)
	if val == "" {
		switch key {
		case "uid":
			reject(codeUIDErr, codeStr(codeUIDErr))
			return true
		case "rid":
			reject(codeRIDErr, codeStr(codeRIDErr))
			return true
		case "biz":
			reject(codeBizErr, codeStr(codeBizErr))
			return true
		case "sfu":
			reject(codeSfuErr, codeStr(codeSfuErr))
			return true
		case "islb":
			reject(codeIslbErr, codeStr(codeIslbErr))
			return true
		case "dist":
			reject(codeDistErr, codeStr(codeDistErr))
			return true
		}
	}
	return false
}
