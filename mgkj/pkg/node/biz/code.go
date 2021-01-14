package node

const (
	codeOK int = -iota
	codeUIDErr
	codeRIDErr
	codeMIDErr
	codeJsepErr
	codeSdpErr
	codePubErr
	codeSubErr
	codeIslbErr
	codeSfuErr
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
	codeIslbErr:    "islb not found",
	codeSfuErr:     "sfu not found",
	codeUnknownErr: "unknown error",
}

func codeStr(code int) string {
	return codeErr[code]
}

var emptyMap = map[string]interface{}{}
