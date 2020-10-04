package server

import (
	"log"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestInitFanout(t *testing.T) {

	exchange := "exchangeName"

	ch := InitFanout("amqp://test:test@127.0.0.1:5672/my_vhost", exchange)

	i := 0
	for {
		time.Sleep(1 * time.Minute)
		greetings := []string{"Hello world! test after ", strconv.Itoa(i)}
		ch.Publish(exchange, strings.Join(greetings, " "), "")
		i = i + 1
	}

}

func TestNewSubscriber(t *testing.T) {
	exchange := "exchangeName"

	sub := NewSubscriber("amqp://test:test@127.0.0.1:5672/my_vhost", exchange, "test1")

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
