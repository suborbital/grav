package grav

import (
	"errors"
	"fmt"
)

// ErrTransportNotConfigured represent package-level vars
var (
	ErrTransportNotConfigured = errors.New("transport plugin not configured")
)

// Grav represents a Grav message bus instance
type Grav struct {
	bus       *messageBus
	transport Transport
	discovery Discovery
}

// New creates a new Grav instance
func New() *Grav {
	return NewWithTransport(nil, nil)
}

// NewWithTransport creates a new Grav with a transport plugin configured
func NewWithTransport(tspt Transport, disc Discovery) *Grav {
	g := &Grav{
		bus:       newMessageBus(),
		transport: tspt,
		discovery: disc,
	}

	if g.transport != nil {
		go func() {
			if err := g.transport.Serve(g.Connect()); err != nil {
				fmt.Println("transport failed: " + err.Error())
			}

			if g.discovery != nil {
				if err := g.discovery.Start(g.transport, g.Connect); err != nil {
					fmt.Println("discovery failed: " + err.Error())
				}
			}
		}()
	}

	return g
}

// Connect creates a new connection (pod) to the bus
func (g *Grav) Connect() *Pod {
	opts := &podOpts{WantsReplay: false}

	return g.connectWithOpts(opts)
}

// ConnectWithReplay creates a new connection (pod) to the bus
// and replays recent messages when the pod sets its onFunc
func (g *Grav) ConnectWithReplay() *Pod {
	opts := &podOpts{WantsReplay: true}

	return g.connectWithOpts(opts)
}

// ConnectEndpoint uses the configured transport to connect the bus to an external endpoint
func (g *Grav) ConnectEndpoint(endpoint string) error {
	if g.transport == nil {
		return ErrTransportNotConfigured
	}

	return g.transport.ConnectEndpoint(endpoint, g.Connect)
}

// ConnectEndpointWithReplay uses the configured transport to connect the bus to an external endpoint
// and replays recent messages to the endpoint when the pod registers its onFunc
func (g *Grav) ConnectEndpointWithReplay(endpoint string) error {
	if g.transport == nil {
		return ErrTransportNotConfigured
	}

	return g.transport.ConnectEndpoint(endpoint, g.ConnectWithReplay)
}

func (g *Grav) connectWithOpts(opts *podOpts) *Pod {
	pod := newPod(g.bus.busChan, opts)

	g.bus.addPod(pod)

	return pod
}
