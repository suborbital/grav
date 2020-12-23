package main

import (
	"fmt"

	"github.com/suborbital/grav/grav"
)

func requestReply() {
	g := grav.New()

	// connect a pod and set it to reply to any message that it receives
	p1 := g.Connect()
	p1.On(func(msg grav.Message) error {
		data := string(msg.Data())

		reply := grav.NewMsg(grav.MsgTypeDefault, []byte(fmt.Sprintf("hey %s", data)))
		p1.ReplyTo(msg, reply)

		return nil
	})

	p2 := g.Connect()
	msg := grav.NewMsg(grav.MsgTypeDefault, []byte("joey"))

	// send a message and wait for a response
	p2.Send(msg).WaitOn(func(msg grav.Message) error {
		fmt.Println("got a reply:", string(msg.Data()))

		return nil
	})
}
