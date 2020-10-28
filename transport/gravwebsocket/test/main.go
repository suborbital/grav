package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/gravwebsocket"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

func main() {
	logger := vlog.Default(vlog.Level(vlog.LogLevelDebug))

	gwss := gravwebsocket.New(&grav.TransportOpts{
		Logger: logger,
	})

	g := grav.NewWithTransport(gwss)

	pod := g.Connect()
	pod.On(func(msg grav.Message) error {
		fmt.Println("got here!", string(msg.Data()))
		return nil
	})

	if err := g.ConnectEndpoint("ws://localhost:8080/meta/message"); err != nil {
		logger.Error(err)
	} else {
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, world")))
	}

	vk := vk.New()
	vk.HandleHTTP(http.MethodGet, "/meta/message", gwss.HTTPHandlerFunc())

	go func() {
		<-time.After(time.Second * time.Duration(10))
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, again")))
	}()

	vk.Start()
}
