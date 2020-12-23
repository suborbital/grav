# Sending Messages

Sending a message using a Pod is as simple as calling `Send(msg)`. The only required field is the message type, which is analogous to a topic or subject in other messaging systems. Messages have a few other properties that can be taken advantage of, which will be discussed in a future section.

```go
package main

import (
	"encoding/json"

	"github.com/suborbital/grav/grav"
)

type heartbeat struct {
	Healthy bool `json:"healthy"`
}

func sending() {
	g := grav.New()

	heart := heartbeat{Healthy: true}
	msg := grav.NewMsg(grav.MsgTypeDefault, heart.JSON())

	p := g.Connect()
	p.Send(msg)
}

func (h *heartbeat) JSON() []byte {
	bytes, _ := json.Marshal(h)
	return bytes
}

```

