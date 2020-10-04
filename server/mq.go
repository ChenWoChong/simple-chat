package server

import (
	"fmt"
	rabbitMQ "github.com/ChenWoChong/simple-chat/pkg/rabbitmq"
	"github.com/golang/glog"
)

var (
	serverMQ *rabbitMQ.RabbitMQ
)

// amqp://test:test@127.0.0.1:5672/my_vhost

func InitFanout(url, exchangeName string) *rabbitMQ.RabbitMQ {

	glog.V(4).Infoln(logTag, fmt.Sprintf("InitFanout: url: %s, exchangeName: %s", url, exchangeName))

	serverMQ = rabbitMQ.Connect(url)
	rabbitMQ.NewExchange(url, exchangeName, rabbitMQ.Fanout)
	return serverMQ
}

func NewSubscriber(url, exchangeName, name string) *rabbitMQ.RabbitMQ {

	glog.V(4).Infoln(logTag, fmt.Sprintf("NewSubscriber: url: %s, exchangeName: %s, name: %s", url, exchangeName, name))

	//第一个参数：rabbitmq服务器的链接，第二个参数：交换机名字，第三个参数：交换机类型
	//3
	// 队列绑定到exchange
	receiveMq := rabbitMQ.New(url, name)

	rabbitMQ.NewExchange(url, exchangeName, rabbitMQ.Fanout)

	receiveMq.Bind(exchangeName, "")

	return receiveMq
}
