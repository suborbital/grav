package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
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
		grav.UseEndpoint(port, "/meta/message"),
		grav.UseMeshTransport(gwss),
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
		<-time.After(time.Second * time.Duration(5))
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, world")))

		<-time.After(time.Second * time.Duration(5))
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, again")))

		<-time.After(time.Second * time.Duration(5))
		if err := g.Withdraw(); err != nil {
			logger.Error(errors.Wrap(err, "failed to Withdraw"))
			os.Exit(1)
		}

		if err := g.Stop(); err != nil {
			logger.Error(errors.Wrap(err, "failed to Stop"))
			os.Exit(1)
		}

		logger.Info("withdrew and stopped!")
		os.Exit(0)
	}()

	if err := vk.Start(); err != nil {
		logger.Error(errors.Wrap(err, "failed to vk.Start"))
	}
}
