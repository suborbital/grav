# Receiving Messages

Pods have several different methods beyond `On` to help with receiving messages in different scenarios. There are two types of receive methods: asynchronous and synchronous. Synchronous receive methods block until the desired message is received, whereas asynchronous receive methods run "in the background" and do not block at the callsite. 

`On` and `OnType` are async, and they are shown here:

```go
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

```

The sync receive methods `WaitOn` and `WaitUntil` are shown here:

```go
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

```

{% hint style="info" %}
The sync receive methods set the Pod's receive function to `nil` after returning, so the Pod stops accepting new messages. The async methods however will cause the Pod to continue receiving until the Pod is disconnected or the receive function is set to `nil`. This can be done with `pod.On(nil)`
{% endhint %}

For sync receive methods, the `grav.ErrMsgNotWanted` error is used to indicate that the desired message has not yet been received. Returning `nil` or another error will let the Pod know that the operation has completed, and the return value will be propagated to the caller if desired.

