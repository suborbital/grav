package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/suborbital/grav/discovery/local"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/websocket"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

func main() {
	logger := vlog.Default(vlog.Level(vlog.LogLevelDebug))
	gwss := websocket.New()
	locald := local.New()

	port := os.Getenv("VK_HTTP_PORT")

	g := grav.New(
		grav.UseLogger(logger),
		grav.UseEndpoint(port, ""),
		grav.UseTransport(gwss),
		grav.UseDiscovery(locald),
	)

	pod := g.Connect()
	pod.On(func(msg grav.Message) error {
		fmt.Println("received something:", string(msg.Data()))
		return nil
	})

	vk := vk.New(vk.UseAppName("websocket tester"))
	vk.HandleHTTP(http.MethodGet, "/meta/message", gwss.HTTPHandlerFunc())

	go func() {
		<-time.After(time.Second * time.Duration(10))
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, world")))

		<-time.After(time.Second * time.Duration(10))
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, again")))
	}()

	vk.Start()
}
