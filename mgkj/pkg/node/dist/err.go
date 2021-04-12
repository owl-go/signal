package dist

import (
	"mgkj/pkg/ws"
	"mgkj/util"
)

const (
	codeOK int = -iota
	codeUIDErr
	codeRIDErr
	codeIslbErr
	codeDistErr
	codeUnknownErr
)

var codeErr = map[int]string{
	codeOK:         "OK",
	codeUIDErr:     "uid not found",
	codeRIDErr:     "rid not found",
	codeIslbErr:    "islb not found",
	codeDistErr:    "dist not found",
	codeUnknownErr: "unknown error",
}

func codeStr(code int) string {
	return codeErr[code]
}

var emptyMap = map[string]interface{}{}

func invalid(msg map[string]interface{}, key string, reject ws.RejectFunc) bool {
	val := util.Val(msg, key)
	if val == "" {
		switch key {
		case "uid":
			reject(codeUIDErr, codeStr(codeUIDErr))
			return true
		case "rid":
			reject(codeRIDErr, codeStr(codeRIDErr))
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
