package main

import (
	"fmt"

	"github.com/suborbital/grav/grav"
)

func receiveMessagesAsync() {
	g := grav.New()

	// create a pod and set a function to be called on each message
	// if the msgFunc returns an error, it indicates to the Gav instance that
	// the message could not be handled.
	p := g.Connect()
	p.On(func(msg grav.Message) error {
		fmt.Println("message received:", string(msg.Data()))

		return nil
	})

	// create a second pod and set it to receive only messages of a specific type
	// errors indicate that a message could not be handled.
	p2 := g.Connect()
	p2.OnType("grav.specialmessage", func(msg grav.Message) error {
		fmt.Println("special message received:", string(msg.Data()))

		return nil
	})
}
