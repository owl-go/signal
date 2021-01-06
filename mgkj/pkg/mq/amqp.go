package mq

import (
	"encoding/json"
	"net"
	"time"

	"mgkj/pkg/log"
	"mgkj/pkg/util"

	"github.com/chuckpreslar/emission"
	"github.com/streadway/amqp"
)

const (
	connTimeout         = 3 * time.Second
	broadCastRoutingKey = "broadcast"
	rpcExchange         = "rpcExchange"
	broadCastExchange   = "broadCastExchange"
)

// Amqp Rabbitmq管理对象
type Amqp struct {
	emission.Emitter
	conn             *amqp.Connection
	rpcChannel       *amqp.Channel
	broadCastChannel *amqp.Channel
	rpcQueue         amqp.Queue
	broadCastQueue   amqp.Queue
}

// New 创建一个Amqp对象
func New(rpc, event, url string) *Amqp {
	a := &Amqp{
		Emitter: *emission.NewEmitter(),
	}
	var err error
	a.conn, err = amqp.DialConfig(url, amqp.Config{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, connTimeout)
		},
	})
	if err != nil {
		log.Panicf(err.Error())
		return nil
	}

	err = a.initRPC(rpc)
	if err != nil {
		log.Panicf(err.Error())
		return nil
	}

	err = a.initBroadCast(event)
	if err != nil {
		log.Panicf(err.Error())
		return nil
	}
	return a
}

// Close 关闭对象
func (a *Amqp) Close() {
	if a.conn != nil {
		a.conn.Close()
	}
}

// initRPC 初始化RPC
func (a *Amqp) initRPC(id string) error {
	var err error
	// 创建通道
	a.rpcChannel, err = a.conn.Channel()
	if err != nil {
		return err
	}

	// a direct router 创建路由
	err = a.rpcChannel.ExchangeDeclare(rpcExchange, "direct", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// a receive queue 创建队列
	a.rpcQueue, err = a.rpcChannel.QueueDeclare(id, false, false, true, false, nil)
	if err != nil {
		return err
	}

	// bind queue to it's name 绑定队列路由关系
	err = a.rpcChannel.QueueBind(a.rpcQueue.Name, a.rpcQueue.Name, rpcExchange, false, nil)
	return err
}

// initBroadCast 初始化广播
func (a *Amqp) initBroadCast(id string) error {
	var err error
	// 创建通道
	a.broadCastChannel, err = a.conn.Channel()
	if err != nil {
		return err
	}

	// a receiving broadcast msg queue 创建路由
	err = a.broadCastChannel.ExchangeDeclare(broadCastExchange, "topic", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// a receive queue 创建队列
	a.broadCastQueue, err = a.broadCastChannel.QueueDeclare(id, false, false, true, false, nil)
	if err != nil {
		return err
	}

	// bind to broadCastRoutingKey 绑定队列路由关系
	err = a.broadCastChannel.QueueBind(a.broadCastQueue.Name, broadCastRoutingKey, broadCastExchange, false, nil)
	return err
}

// ConsumeRPC 获取RPC消息
func (a *Amqp) ConsumeRPC() (<-chan amqp.Delivery, error) {
	return a.rpcChannel.Consume(
		a.rpcQueue.Name, // queue
		"",              // consumer
		true,            // auto ack
		false,           // exclusive
		false,           // no local
		false,           // no wait
		nil,             // args
	)
}

// ConsumeBroadcast 获取广播消息
func (a *Amqp) ConsumeBroadcast() (<-chan amqp.Delivery, error) {
	return a.broadCastChannel.Consume(
		a.broadCastQueue.Name, // queue
		"",                    // consumer
		true,                  // auto ack
		false,                 // exclusive
		false,                 // no local
		false,                 // no wait
		nil,                   // args
	)
}

// RPCCall 发布一个RPC消息
func (a *Amqp) RPCCall(id string, msg map[string]interface{}, corrID string) (string, error) {
	str := util.Marshal(msg)
	correlatioID := ""
	if corrID == "" {
		correlatioID = util.RandStr(8)
	} else {
		correlatioID = corrID
	}
	log.Infof("Amqp.RPCCall id=%s msg=%v corrID=%s", id, msg, corrID)

	err := a.rpcChannel.Publish(
		rpcExchange, // exchange
		id,          // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: correlatioID,
			ReplyTo:       a.rpcQueue.Name,
			Body:          []byte(str),
		})
	if err != nil {
		return "", err
	}
	return correlatioID, nil
}

// RPCCallWithResp 发布一个RPC消息，回调返回
func (a *Amqp) RPCCallWithResp(id string, msg map[string]interface{}, callback interface{}) error {
	str := util.Marshal(msg)
	corrID := util.RandStr(8)
	a.Emitter.On(corrID, callback)
	log.Infof("Amqp.RpcCallWithResp id=%s msg=%v corrID=%s", id, msg, corrID)

	err := a.rpcChannel.Publish(
		rpcExchange, // exchange
		id,          // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: corrID,
			ReplyTo:       a.rpcQueue.Name,
			Body:          []byte(str),
		})
	if err != nil {
		return err
	}
	return nil
}

// BroadCast 发布广播消息
func (a *Amqp) BroadCast(msg map[string]interface{}) error {
	str, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("Marshal %v", err)
		return err
	}
	log.Infof("Amqp.BroadCast msg=%v", msg)

	return a.broadCastChannel.Publish(
		broadCastExchange,   // exchange
		broadCastRoutingKey, // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(str),
			ReplyTo:     a.broadCastQueue.Name,
		})
}
