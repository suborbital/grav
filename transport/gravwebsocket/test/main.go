package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/suborbital/grav/discovery/gravlocal"
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

	locald := gravlocal.New(&grav.DiscoveryOpts{
		Logger: logger,
	})

	g := grav.NewWithTransport(gwss, locald)

	pod := g.Connect()
	pod.On(func(msg grav.Message) error {
		fmt.Println("got here!", string(msg.Data()))
		return nil
	})

	vk := vk.New()
	vk.HandleHTTP(http.MethodGet, "/meta/message", gwss.HTTPHandlerFunc())

	go func() {
		<-time.After(time.Second * time.Duration(20))
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, again")))
	}()

	vk.Start()
}
