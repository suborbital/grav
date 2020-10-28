package grav

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/suborbital/vektor/vlog"
)

// ErrTransportNotConfigured represent package-level vars
var (
	ErrTransportNotConfigured = errors.New("transport plugin not configured")
)

// Grav represents a Grav message bus instance
type Grav struct {
	NodeUUID  string
	bus       *messageBus
	logger    *vlog.Logger
	transport Transport
	discovery Discovery
}

// Opts represent Grav options
type Opts struct {
	Logger    *vlog.Logger
	Port      int
	Transport Transport
	Discovery Discovery
}

// New creates a new Grav instance with no transport or discovery enabled
func New() *Grav {
	return NewWithOptions(&Opts{vlog.Default(), 8080, nil, nil})
}

// NewWithOptions creates a new Grav with a logger, transport, and discovery configured
func NewWithOptions(opts *Opts) *Grav {
	nodeUUID := uuid.New().String()

	g := &Grav{
		NodeUUID:  nodeUUID,
		bus:       newMessageBus(),
		logger:    opts.Logger,
		transport: opts.Transport,
		discovery: opts.Discovery,
	}

	// start transport, then discovery if each have been configured (can have transport but no discovery)
	if g.transport != nil {
		transportOpts := &TransportOpts{
			NodeUUID: nodeUUID,
			Port:     opts.Port,
			Logger:   opts.Logger,
		}

		go func() {
			if err := g.transport.Serve(transportOpts, g.Connect); err != nil {
				fmt.Println("transport failed: " + err.Error())
			}

			if g.discovery != nil {
				discoveryOpts := &DiscoveryOpts{
					NodeUUID:      nodeUUID,
					TransportPort: transportOpts.Port,
					Logger:        opts.Logger,
				}

				if err := g.discovery.Start(discoveryOpts, g.transport, g.Connect); err != nil {
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
