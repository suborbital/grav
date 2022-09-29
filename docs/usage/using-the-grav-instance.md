# Using the Grav Instance

Once you have a Grav instance, you can start connecting Pods and sending messages!

```go
package main

import (
	"fmt"
	"time"

	"github.com/suborbital/grav/grav"
)

func useGravInstance() {
	g := grav.New()

	// create a pod and set a function to be called on each message
	p := g.Connect()
	p.On(func(msg grav.Message) error {
		fmt.Println("message received:", string(msg.Data()))

		return nil
	})

	// create a second pod and send a message
	p2 := g.Connect()
	p2.Send(grav.NewMsg(grav.MsgTypeDefault, []byte("hello, world")))

	// messages are asyncronous, so pause for a second to allow the message to send
	<-time.After(time.Second)
}

```

Here we can see Pods being created, and the two main Pod methods being used. [Read about Pods](Suborbital/grav/docs/concepts/pods.md) if you haven't already.

Each time `On` is called, the function used to receive messages gets **replaced,** therefore if multiple receiving functions are needed, multiple Pods should be created.

