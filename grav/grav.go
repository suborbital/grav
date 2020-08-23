package grav

import "errors"

// ErrTransportNotConfigured represent package-level vars
var (
	ErrTransportNotConfigured = errors.New("transport plugin not configured")
)

// Grav represents a Grav message bus instance
type Grav struct {
	bus       *messageBus
	transport Transport
}

// New creates a new Grav instance
func New() *Grav {
	return NewWithTransport(nil, nil)
}

// NewWithTransport creates a new Grav with a transport plugin configured
func NewWithTransport(tspt Transport, opts *TransportOpts) *Grav {
	g := &Grav{
		bus:       newMessageBus(),
		transport: tspt,
	}

	if tspt != nil {
		go func() {
			if err := tspt.Serve(opts, g.Connect()); err != nil {
				// not sure what to do here, yet
			}
		}()
	}

	return g
}

// Connect creates a new connection (pod) to the bus
func (g *Grav) Connect() *Pod {
	pod := newPod("", g.bus.busChan)

	g.bus.addPod(pod)

	return pod
}

// ConnectEndpoint uses the configured transport to connect the bus to an external endpoint
func (g *Grav) ConnectEndpoint(endpoint string) error {
	if g.transport == nil {
		return ErrTransportNotConfigured
	}

	return g.transport.ConnectEndpoint(endpoint, g.Connect)
}
