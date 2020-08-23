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
	return NewWithTransport(nil)
}

// NewWithTransport creates a new Grav with a transport plugin configured
func NewWithTransport(tspt Transport) *Grav {
	g := &Grav{
		bus:       newMessageBus(),
		transport: tspt,
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

// Serve directs the configured transport plugin to serve an endpoint that other nodes can connect to
// Calling Serve will block, so if the Grav endpoint is meant to run in the background, it should be called on
// a goroutine. Calling Serve is optional if incoming connections are not needed (or if no transport is being used)
func (g *Grav) Serve(opts *TransportServeOpts) error {
	if g.transport == nil {
		return ErrTransportNotConfigured
	}

	return g.transport.Serve(opts)
}
