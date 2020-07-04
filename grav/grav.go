package grav

import (
	"github.com/suborbital/grav/grav/bus"
	"github.com/suborbital/grav/grav/pod"
)

// Grav represents a Grav message bus instance
type Grav struct {
	GroupName string

	bus *bus.Bus
}

// New creates a new Grav instance
func New() *Grav {
	g := &Grav{
		GroupName: "",
		bus:       bus.New(),
	}

	return g
}

// Connect creates a new connection (pod) to the bus
func (g *Grav) Connect() *pod.Pod {
	pod := pod.New("", g.bus.RecieveChan())

	g.bus.AddPod(pod)

	return pod
}
