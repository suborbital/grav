package main

import (
	"fmt"
	"time"

	"github.com/suborbital/grav/grav"
)

func receiveMessagesSync() {
	g := grav.New()

	go sendMessages(g.Connect())

	// create a pod and set it to wait for a specific message.
	// WaitOn will block forever if the message is never received.
	p3 := g.Connect()
	p3.WaitOn(func(msg grav.Message) error {
		if msg.Type() != "grav.desiredmessage" {
			return grav.ErrMsgNotWanted
		}

		fmt.Println("desired message received:", string(msg.Data()))

		return nil
	})

	// create a pod that waits up to 3 seconds for a specific message
	// ErrWaitTimeout is returned if the timeout elapses
	p4 := g.Connect()
	err := p4.WaitUntil(grav.Timeout(3), func(msg grav.Message) error {
		if msg.Type() != "grav.importantmessage" {
			return grav.ErrMsgNotWanted
		}

		fmt.Println("important message received:", string(msg.Data()))

		return nil
	})

	// timeout will occur because message is never received
	fmt.Println(err)
}

func sendMessages(p *grav.Pod) {
	<-time.After(time.Millisecond * 500)
	p.Send(grav.NewMsg("grav.desiredmessage", []byte("desired!")))
}
