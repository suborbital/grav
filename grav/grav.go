package grav

// Grav represents a Grav message bus instance
type Grav struct {
	GroupName string

	bus *messageBus
}

// New creates a new Grav instance
func New() *Grav {
	g := &Grav{
		GroupName: "",
		bus:       newMessageBus(),
	}

	return g
}

// Connect creates a new connection (pod) to the bus
func (g *Grav) Connect() *Pod {
	pod := newPod("", g.bus.busChan)

	g.bus.addPod(pod)

	return pod
}
