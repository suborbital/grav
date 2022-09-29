# Request / Reply

Beyond just sending messages, Grav can be used to make requests and get replies. This can be useful for service-to-service communication, RPC-style message, and more.

```go
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

```

This is an example of [Receipts](Suborbital/grav/docs/concepts/receipts.md) in action. When a message is sent, the `MsgReceipt` is returned, and then the `WaitOn` method is used to retrieve a reply from the message stream.

`WaitOn` will block forever until a reply is received. There is an alternate method `WaitUntil` that accepts a timeout. Since a receipt is an extension of the Pod that created it, calling any of its methods will replace the receive function for the Pod. 

Receipts also have an async method, `OnReply`. This will run the provided function in the background when a reply is received, and will set the Pod's receive function to `nil` afterwards.

