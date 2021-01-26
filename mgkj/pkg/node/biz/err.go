package node

import (
	"fmt"
	"mgkj/pkg/util"

	"github.com/cloudwebrtc/go-protoo/peer"
)

const (
	codeOK int = -iota
	codeUIDErr
	codeRIDErr
	codeMIDErr
	codeJsepErr
	codeSdpErr
	codePubErr
	codeSubErr
	codeSfuErr
	codeIslbErr
	codeUnknownErr
)

var codeErr = map[int]string{
	codeOK:         "OK",
	codeUIDErr:     "uid not found",
	codeRIDErr:     "rid not found",
	codeMIDErr:     "mid not found",
	codeJsepErr:    "jsep not found",
	codeSdpErr:     "sdp not found",
	codePubErr:     "pub not found",
	codeSubErr:     "sub not found",
	codeSfuErr:     "sfu not found",
	codeIslbErr:    "islb not found",
	codeUnknownErr: "unknown error",
}

func codeStr(code int) string {
	return codeErr[code]
}

var emptyMap = map[string]interface{}{}

func getMID(uid string) string {
	return fmt.Sprintf("%s#%s", uid, util.RandStr(6))
}

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
		case "mid":
			reject(codeMIDErr, codeStr(codeMIDErr))
			return true
		case "jsep":
			reject(codeJsepErr, codeStr(codeJsepErr))
			return true
		case "sdp":
			reject(codeSdpErr, codeStr(codeSdpErr))
			return true
		case "sfu":
			reject(codeSfuErr, codeStr(codeSfuErr))
			return true
		case "islb":
			reject(codeIslbErr, codeStr(codeIslbErr))
			return true
		}
	}
	return false
}
