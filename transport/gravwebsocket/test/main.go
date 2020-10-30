package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/suborbital/grav/discovery/gravlocal"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/gravwebsocket"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

func main() {
	logger := vlog.Default(vlog.Level(vlog.LogLevelInfo))
	gwss := gravwebsocket.New()
	locald := gravlocal.New()

	port, _ := strconv.Atoi(os.Getenv("VK_HTTP_PORT"))

	g := grav.NewWithOptions(&grav.Opts{
		Logger:    logger,
		Port:      port,
		Transport: gwss,
		Discovery: locald,
	})

	pod := g.Connect()
	pod.On(func(msg grav.Message) error {
		fmt.Println("got here!", string(msg.Data()))
		return nil
	})

	vk := vk.New()
	vk.HandleHTTP(http.MethodGet, "/meta/message", gwss.HTTPHandlerFunc())

	go func() {
		<-time.After(time.Second * time.Duration(10))
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, world")))

		<-time.After(time.Second * time.Duration(10))
		pod.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, again")))
	}()

	vk.Start()
}
