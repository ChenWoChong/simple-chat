package rabbitMQ

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/streadway/amqp"
)

const (
	Fanout string = `fanout`
	Direct string = `direct`
	Topic  string = `topic`
	logTag string = `[Rabbitmq]`
)

//声明队列类型
type RabbitMQ struct {
	channel  *amqp.Channel
	Name     string
	exchange string
}

//连接服务器
func Connect(s string) *RabbitMQ {
	//连接rabbitmq
	conn, e := amqp.Dial(s)
	if e != nil {
		glog.Fatalf("%s: %s", e, "连接Rabbitmq服务器失败！")
	}
	ch, e := conn.Channel()
	if e != nil {
		glog.Fatalf("%s: %s", e, "无法打开频道！")
	}

	mq := &RabbitMQ{
		channel: ch,
	}

	return mq
}

//初始化单个消息队列
//第一个参数：rabbitmq服务器的链接，第二个参数：队列名字
func New(s string, name string) *RabbitMQ {
	//连接rabbitmq
	conn, e := amqp.Dial(s)
	if e != nil {
		glog.Fatalf("%s: %s", e, "连接Rabbitmq服务器失败！")
	}

	ch, e := conn.Channel()
	if e != nil {
		glog.Fatalf("%s: %s", e, "无法打开频道！")
	}

	q, e := ch.QueueDeclare(
		name,  //队列名
		false, //是否开启持久化
		true,  //不使用时删除
		false, //排他
		false, //不等待
		nil,   //参数
	)
	if e != nil {
		glog.Fatalf("%s: %s", e, "初始化队列失败！")
	}

	mq := &RabbitMQ{
		channel: ch,
		Name:    q.Name,
	}
	return mq
}

//批量初始化消息队列
//第一个参数：rabbitmq服务器的链接，第二个参数：队列名字列表

//声明交换机
func (q *RabbitMQ) QueueDeclare(queue string) {
	// 1. 申请队列，如果队列不存在会自动创建，如何存在则跳过创建
	// 保证队列存在，消息能发送到队列中
	_, e := q.channel.QueueDeclare(
		queue,
		// 是否持久化
		false,
		// 是否为自动删除
		true,
		// 是否具有排他性
		false,
		// 是否阻塞
		false,
		// 额外属性
		nil,
	)
	if e != nil {
		glog.Fatalf("%s: %s", e, "声明交换机！")
	}
}

//删除交换机
func (q *RabbitMQ) QueueDelete(queue string) {
	_, e := q.channel.QueueDelete(queue, false, true, false)
	if e != nil {
		glog.Fatalf("%s: %s", e, "删除队列失败！")
	}
}

//配置队列参数
func (q *RabbitMQ) Qos() {
	e := q.channel.Qos(1, 0, false)
	if e != nil {
		glog.Fatalf("%s: %s", e, "无法设置QoS")
	}
}

//配置交换机参数

//初始化交换机
//第一个参数：rabbitmq服务器的链接，第二个参数：交换机名字，第三个参数：交换机类型
func NewExchange(url string, name string, typename string) {
	//连接rabbitmq
	conn, e := amqp.Dial(url)
	if e != nil {
		glog.Fatalf("%s: %s", e, "连接Rabbitmq服务器失败！")
	}
	ch, e := conn.Channel()
	if e != nil {
		glog.Fatalf("%s: %s", e, "无法打开频道！")
	}
	e = ch.ExchangeDeclare(
		name,     // name
		typename, // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if e != nil {
		glog.Fatalf("%s: %s", e, "初始化交换机失败！")
	}

}

//删除交换机
func (q *RabbitMQ) ExchangeDelete(exchange string) {
	e := q.channel.ExchangeDelete(exchange, false, true)
	if e != nil {
		glog.Fatalf("%s: %s", e, "绑定队列失败！")
	}
}

//绑定消息队列到哪个exchange
func (q *RabbitMQ) Bind(exchange string, key string) {
	e := q.channel.QueueBind(
		q.Name,
		key,
		exchange,
		false,
		nil,
	)
	if e != nil {
		glog.Fatalf("%s: %s", e, "绑定队列失败！")
	}
	q.exchange = exchange
}

//向消息队列发送消息
//Send方法可以往某个消息队列发送消息
func (q *RabbitMQ) Send(queue string, body interface{}) {
	str, e := json.Marshal(body)
	if e != nil {
		glog.Fatalf("%s: %s", e, "消息序列化失败！")
	}
	e = q.channel.Publish(
		"",    //交换
		queue, //路由键
		// 如果为true, 会根据exchange类型和routkey规则，如果无法找到符合条件的队列那么会把发送的消息返回给发送者
		false, //必填
		// 如果为true, 当exchange发送消息到队列后发现队列上没有绑定消费者，则会把消息发还给发送者
		false, //立即
		amqp.Publishing{
			ReplyTo: q.Name,
			Body:    []byte(str),
		})
	msg := "向队列:" + q.Name + "发送消息失败！"
	glog.Fatalf("%s: %s", e, msg)
}

//向exchange发送消息
//Publish方法可以往某个exchange发送消息
func (q *RabbitMQ) Publish(exchange string, body interface{}, key string) {

	glog.V(4).Infoln(logTag, `publish`, exchange, body, "key: ", key)

	str, e := json.Marshal(body)
	if e != nil {
		glog.Fatalf("%s: %s", e, "消息序列化失败！")
	}
	e = q.channel.Publish(
		exchange,
		key,
		// 如果为true, 会根据exchange类型和routkey规则，如果无法找到符合条件的队列那么会把发送的消息返回给发送者
		false,
		// 如果为true, 当exchange发送消息到队列后发现队列上没有绑定消费者，则会把消息发还给发送者
		false,
		amqp.Publishing{ReplyTo: q.Name,
			Body: []byte(str)},
	)
	if e != nil {
		glog.Fatalf("%s: %s", e, "向路由发送消息失败！")
	}
}

//接收某个消息队列的消息
func (q *RabbitMQ) Consume() <-chan amqp.Delivery {
	c, e := q.channel.Consume(
		q.Name, //指定从哪个队列中接收消息
		// 用来区分多个消费者
		"",
		// 是否自动应答
		true,
		// 是否具有排他性
		false,
		// 如果设置为true，表示不能将同一个connection中发送的消息传递给这个connection中的消费者
		false,
		// 队列消费是否阻塞
		false,
		nil,
	)
	if e != nil {
		glog.Fatalf("%s: %s", e, "接收消息失败！")
	}
	return c
}

//关闭队列连接
func (q *RabbitMQ) Close() {
	q.channel.Close()
}

//错误处理函数
//func failOnError(err error, msg string) {
//	if err != nil {
//		glog.Fatalf("%s: %s", msg, err)
//	}
//}
