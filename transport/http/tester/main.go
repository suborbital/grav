package main

import (
	"fmt"
	"os"
	"time"

	"github.com/suborbital/grav/discovery/local"
	"github.com/suborbital/grav/grav"
	ghttp "github.com/suborbital/grav/transport/http"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

func main() {
	logger := vlog.Default(vlog.Level(vlog.LogLevelDebug))
	gravhttp := ghttp.New()
	locald := local.New()

	port := os.Getenv("VK_HTTP_PORT")

	g := grav.New(
		grav.UseLogger(logger),
		grav.UseEndpoint(port, ""),
		grav.UseTransport(gravhttp),
		grav.UseDiscovery(locald),
	)

	pod := g.Connect()
	pod.On(func(msg grav.Message) error {
		fmt.Println("received something:", string(msg.Data()))
		return nil
	})

	vk := vk.New(vk.UseAppName("http tester"))
	vk.POST("/meta/message", gravhttp.HandlerFunc())

	go func() {
		<-time.After(time.Second * time.Duration(10))
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, world")))

		<-time.After(time.Second * time.Duration(10))
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, again")))
	}()

	vk.Start()
}
