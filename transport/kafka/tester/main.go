package main

import (
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/kafka"
	"github.com/suborbital/vektor/vlog"
)

func main() {
	logger := vlog.Default(vlog.Level(vlog.LogLevelDebug))

	knats, err := kafka.New("127.0.0.1:9092")
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to nats.New"))
	}

	g := grav.New(
		grav.UseLogger(logger),
		grav.UseBridgeTransport(knats),
	)

	if err := g.ConnectBridgeTopic("something.in"); err != nil {
		log.Fatal(errors.Wrap(err, "failed to ConnectTopic"))
	}

	if err := g.ConnectBridgeTopic("something.out"); err != nil {
		log.Fatal(errors.Wrap(err, "failed to ConnectTopic"))
	}

	pod := g.Connect()
	pod.On(func(msg grav.Message) error {
		fmt.Println("received something:", string(msg.Data()), msg.Type())
		return nil
	})

	go func() {
		<-time.After(time.Second * time.Duration(5))
		fmt.Println("sending 1")
		pod.Send(grav.NewMsg("something.in", []byte("world")))

		<-time.After(time.Second * time.Duration(5))
		fmt.Println("sending 2")
		pod.Send(grav.NewMsg("something.in", []byte("again")))
	}()

	<-time.After(time.Minute)
}
