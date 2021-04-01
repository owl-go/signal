package peer

import (
	"encoding/json"

	"github.com/chuckpreslar/emission"
	"github.com/cloudwebrtc/go-protoo/logger"
	"github.com/cloudwebrtc/go-protoo/transport"
)

type Transcation struct {
	id     int
	accept AcceptFunc
	reject RejectFunc
}

type Peer struct {
	emission.Emitter
	id           string
	transport    *transport.WebSocketTransport
	transcations map[int]*Transcation
}

func NewPeer(id string, transport *transport.WebSocketTransport) *Peer {
	var peer Peer
	peer.Emitter = *emission.NewEmitter()
	peer.id = id
	peer.transport = transport
	peer.transport.On("message", peer.handleMessage)
	peer.transport.On("close", func(code int, err string) {
		logger.Infof("Transport closed [%d] %s", code, err)
		peer.Emit("close", code, err)
	})
	peer.transport.On("error", func(code int, err string) {
		logger.Warnf("Transport got error (%d, %s)", code, err)
		peer.Emit("error", code, err)
	})
	peer.transcations = make(map[int]*Transcation)
	return &peer
}

func (peer *Peer) Close() {
	peer.transport.Close()
}

func (peer *Peer) ID() string {
	return peer.id
}

func (peer *Peer) Request(method string, data map[string]interface{}, success AcceptFunc, reject RejectFunc) {
	id := GenerateRandomNumber()
	request := &Request{
		Request: true,
		Id:      id,
		Method:  method,
		Data:    data,
	}
	str, err := json.Marshal(request)
	if err != nil {
		logger.Errorf("Marshal %v", err)
		return
	}

	transcation := &Transcation{
		id:     id,
		accept: success,
		reject: reject,
	}

	peer.transcations[id] = transcation
	peer.transport.Send(string(str))
}

func (peer *Peer) Notify(method string, data map[string]interface{}) {
	notification := &Notification{
		Notification: true,
		Method:       method,
		Data:         data,
	}
	str, err := json.Marshal(notification)
	if err != nil {
		logger.Errorf("Marshal %v", err)
		return
	}
	peer.transport.Send(string(str))
}

func (peer *Peer) handleMessage(message []byte) {
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		logger.Errorf("handleMessage Unmarshal err => %v", err)
		return
	}
	if data["request"] != nil {
		peer.handleRequest(data)
	} else if data["response"] != nil {
		peer.handleResponse(data)
	} else if data["notification"] != nil {
		peer.handleNotification(data)
	}
	return
}

func (peer *Peer) handleRequest(request map[string]interface{}) {
	accept := func(data map[string]interface{}) {
		response := &Response{
			Response: true,
			Ok:       true,
			Id:       int(request["id"].(float64)),
			Data:     data,
		}
		str, err := json.Marshal(response)
		if err != nil {
			logger.Errorf("Marshal %v", err)
			return
		}
		peer.transport.Send(string(str))
	}

	reject := func(errorCode int, errorReason string) {
		response := &ResponseError{
			Response:    true,
			Ok:          false,
			Id:          int(request["id"].(float64)),
			ErrorCode:   errorCode,
			ErrorReason: errorReason,
		}
		str, err := json.Marshal(response)
		if err != nil {
			logger.Errorf("Marshal %v", err)
			return
		}
		peer.transport.Send(string(str))
	}

	peer.Emit("request", request, accept, reject)
}

func (peer *Peer) handleResponse(response map[string]interface{}) {
	id := int(response["id"].(float64))
	transcation := peer.transcations[id]
	if transcation == nil {
		logger.Errorf("received response does not match any sent request [id:%d]", id)
		return
	}

	if response["ok"] != nil && response["ok"] == true {
		transcation.accept(response["data"].(map[string]interface{}))
	} else {
		transcation.reject(int(response["errorCode"].(float64)), response["errorReason"].(string))
	}

	delete(peer.transcations, id)
}

func (peer *Peer) handleNotification(notification map[string]interface{}) {
	peer.Emit("notification", notification)
}
