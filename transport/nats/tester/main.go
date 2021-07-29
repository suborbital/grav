package main

import (
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/nats"
	"github.com/suborbital/vektor/vlog"
)

func main() {
	logger := vlog.Default(vlog.Level(vlog.LogLevelDebug))

	gnats, err := nats.New("nats://localhost:4222")
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to nats.New"))
	}

	g := grav.New(
		grav.UseLogger(logger),
		grav.UseTransport(gnats),
	)

	if err := g.ConnectBridgeTopic(grav.MsgTypeDefault); err != nil {
		log.Fatal(errors.Wrap(err, "failed to ConnectTopic"))
	}

	pod := g.Connect()
	pod.On(func(msg grav.Message) error {
		fmt.Println("received something:", string(msg.Data()))
		return nil
	})

	go func() {
		<-time.After(time.Second * time.Duration(5))
		fmt.Println("sending 1")
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, world")))

		<-time.After(time.Second * time.Duration(5))
		fmt.Println("sending 2")
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, again")))
	}()

	<-time.After(time.Minute)
}
