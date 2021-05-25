package ws

import (
	"net/http"
	"strconv"

	"signal/pkg/log"

	"github.com/gearghost/go-protoo/peer"
	"github.com/gearghost/go-protoo/transport"
	"github.com/gorilla/websocket"
)

// AcceptFunc 接受函数定义
type AcceptFunc peer.AcceptFunc

// RejectFunc 拒绝函数定义
type RejectFunc peer.RejectFunc

// 错误码
const (
	// ErrInvalidMethod ...
	ErrInvalidMethod = "method not found"
	// ErrInvalidData ...
	ErrInvalidData = "data not found"
)

// DefaultAccept 默认接受处理
func DefaultAccept(data map[string]interface{}) {
	log.Infof("peer accept data => %v", data)
}

// DefaultReject 默认拒绝处理
func DefaultReject(errorCode int, errorReason string) {
	log.Infof("reject errorCode => %v errorReason => %v", errorCode, errorReason)
}

// WebSocketServerConfig 配置对象
type WebSocketServerConfig struct {
	Host          string
	Port          int
	CertFile      string
	KeyFile       string
	HTMLRoot      string
	WebSocketPath string
}

// DefaultConfig 获取默认参数配置
func DefaultConfig() WebSocketServerConfig {
	return WebSocketServerConfig{
		Host:          "0.0.0.0",
		Port:          8443,
		HTMLRoot:      ".",
		WebSocketPath: "/ws",
	}
}

// WebSocketServer websocket对象
type WebSocketServer struct {
	handleWebSocket func(ws *transport.WebSocketTransport, request *http.Request)
	// Websocket upgrader
	upgrader websocket.Upgrader
}

// NewWebSocketServer 新建一个websocket对象
func NewWebSocketServer(handler func(ws *transport.WebSocketTransport, request *http.Request)) *WebSocketServer {
	var server = &WebSocketServer{
		handleWebSocket: handler,
	}
	server.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return server
}

func (server *WebSocketServer) handleWebSocketRequest(writer http.ResponseWriter, request *http.Request) {
	responseHeader := http.Header{}
	//responseHeader.Add("Sec-WebSocket-Protocol", "protoo")
	socket, err := server.upgrader.Upgrade(writer, request, responseHeader)
	if err != nil {
		panic(err)
	}
	wsTransport := transport.NewWebSocketTransport(socket)
	server.handleWebSocket(wsTransport, request)
	wsTransport.ReadMessage()
}

// Bind 绑定处理函数
func (server *WebSocketServer) Bind(cfg WebSocketServerConfig) {
	// Websocket handle func
	http.HandleFunc(cfg.WebSocketPath, server.handleWebSocketRequest)
	//http.Handle("/", http.FileServer(http.Dir(cfg.HTMLRoot)))

	//if cfg.CertFile == "" || cfg.KeyFile == "" {
	log.Infof("non-TLS WebSocketServer listening on: %s:%d", cfg.Host, cfg.Port)
	panic(http.ListenAndServe(cfg.Host+":"+strconv.Itoa(cfg.Port), nil))
	/*} else {
		logger.Infof("TLS WebSocketServer listening on: %s:%d", cfg.Host, cfg.Port)
		panic(http.ListenAndServeTLS(cfg.Host+":"+strconv.Itoa(cfg.Port), cfg.CertFile, cfg.KeyFile, nil))
	}*/
}
