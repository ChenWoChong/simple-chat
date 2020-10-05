package server

import (
	rabbitMQ "github.com/ChenWoChong/simple-chat/pkg/rabbitmq"
	"log"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestInitFanout(t *testing.T) {

	exchange := "exchangeName"

	ch := InitMq("amqp://test:test@127.0.0.1:5672/my_vhost", exchange, rabbitMQ.Fanout)

	i := 0
	for {
		time.Sleep(1 )
		greetings := []string{"Hello world! test after ", strconv.Itoa(i)}
		ch.Publish(exchange, strings.Join(greetings, " "), "")
		i = i + 1
		if i == 10 {
			break
		}
	}

}

func TestNewSubscriber(t *testing.T) {
	exchange := "exchangeName"

	sub := NewSubscriber("amqp://test:test@127.0.0.1:5672/my_vhost", exchange, "test1", "")

	stop := make(chan bool)
	//4
	//接收消息时，指定
	msgs := sub.Consume()
	go func() {
		for d := range msgs {
			log.Printf("recevie1  Received a message: %s", d.Body)
		}
	}()

	<-stop
}
