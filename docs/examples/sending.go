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
