package biz

import (
	"fmt"
	"net/http"

	"mgkj/pkg/util"
	"mgkj/pkg/ws"

	"github.com/gearghost/go-protoo/transport"
)

func in(transport *transport.WebSocketTransport, request *http.Request) {
	vars := request.URL.Query()
	peerID := vars["peer"]
	if peerID == nil || len(peerID) < 1 {
		return
	}

	appID := vars["appid"]
	if appID == nil || len(appID) < 1 {
		return
	}

	id := peerID[0]

	//log.Infof("signal.in, id = %s,appid = %s", id, appID[0])
	logger.Infof(fmt.Sprintf("signal.in,id=%s appid=%s", id, appID[0]), "uid", id, "appid", appID[0])

	peer := ws.NewPeer(id, transport)
	peer.SetAppID(appID[0])

	handleRequest := func(request map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
		defer util.Recover("signal.in handleRequest")
		method := util.Val(request, "method")
		if method == "" {
			//log.Errorf("method = %v", method)
			logger.Errorf(fmt.Sprintf("method=%s", method), "uid", id)
			reject(-1, ws.ErrInvalidMethod)
			return
		}

		data := request["data"]
		if data == nil {
			//log.Errorf("data = %v", data)
			logger.Errorf(fmt.Sprintf("data=%s", data), "uid", id)
			reject(-1, ws.ErrInvalidData)
			return
		}

		msg := data.(map[string]interface{})
		//log.Infof("signal.in handleRequest id=%s method = %s", peer.ID(), method)
		logger.Infof(fmt.Sprintf("signal.in handleRequest id=%s,method=%s", peer.ID(), method), "uid", id)
		wsReq(method, peer, msg, accept, reject)
	}

	handleNotification := func(notification map[string]interface{}) {
		defer util.Recover("signal.in handleNotification")
		method := util.Val(notification, "method")
		if method == "" {
			//log.Errorf("method = %v", method)
			logger.Errorf(fmt.Sprintf("method=%s", method), "uid", id)
			ws.DefaultReject(-1, ws.ErrInvalidMethod)
			return
		}

		data := notification["data"]
		if data == nil {
			//log.Errorf("data = %v", data)
			logger.Errorf(fmt.Sprintf("data=%s", data), "uid", id)
			ws.DefaultReject(-1, ws.ErrInvalidData)
			return
		}

		msg := data.(map[string]interface{})
		//log.Infof("signal.in handleNotification id=%s method = %s", peer.ID(), method)
		logger.Infof(fmt.Sprintf("signal.in handleNotification id=%s, method=%s", peer.ID(), method), "uid", id)
		wsReq(method, peer, msg, ws.DefaultAccept, ws.DefaultReject)
	}

	handleClose := func(code int, err string) {
		//log.Infof("signal.in handleClose = peer (%s) ", peer.ID())
		logger.Infof(fmt.Sprintf("signal.in handleClose = peer %s", peer.ID()), "uid", id)
		peer.Close()
	}

	peer.On("request", handleRequest)
	peer.On("notification", handleNotification)
	peer.On("close", handleClose)
	peer.On("error", handleClose)
}
