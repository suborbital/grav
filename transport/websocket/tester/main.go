package main

import (
	"fmt"
	"log"
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
	logger := vlog.Default(vlog.Level(vlog.LogLevelInfo))
	gwss := websocket.New()
	locald := local.New()

	port := os.Getenv("VK_HTTP_PORT")

	var b2 string
	var cap string
	if len(os.Args) > 2 {
		b2 = os.Args[1]
		cap = os.Args[2]
	}

	g := grav.New(
		grav.UseLogger(logger),
		grav.UseEndpoint(port, ""),
		grav.UseTransport(gwss),
		grav.UseDiscovery(locald),
		grav.UseBelongsTo(b2),
		grav.UseInterests(cap),
	)

	pod := g.Connect()
	pod.On(func(msg grav.Message) error {
		fmt.Println("received something:", string(msg.Data()))
		return nil
	})

	vk := vk.New(vk.UseAppName("websocket tester"))
	vk.HandleHTTP(http.MethodGet, "/meta/message", gwss.HTTPHandlerFunc())

	go func() {
		<-time.After(time.Second * time.Duration(5))
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, world")))

		<-time.After(time.Second * time.Duration(5))
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, again")))

		if len(os.Args) > 3 {
			for _, cap := range os.Args[3:] {
				<-time.After(time.Second * time.Duration(5))

				msg := grav.NewMsg(cap, []byte("hello, "+cap))
				if err := g.Tunnel(cap, msg); err != nil {
					log.Fatal(err)
				}
			}
		}

	}()

	vk.Start()
}
