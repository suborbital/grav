package grav

import (
	"github.com/pkg/errors"

	"github.com/google/uuid"
	"github.com/suborbital/vektor/vlog"
)

// ErrTransportNotConfigured represent package-level vars
var (
	ErrTransportNotConfigured = errors.New("transport plugin not configured")
	ErrTunnelNotEstablished   = errors.New("tunnel cannot be established")
)

// Grav represents a Grav message bus instance
type Grav struct {
	NodeUUID  string
	BelongsTo string
	Interests []string
	bus       *messageBus
	logger    *vlog.Logger
	hub       *hub
}

// New creates a new Grav with the provided options
func New(opts ...OptionsModifier) *Grav {
	nodeUUID := uuid.New().String()

	options := newOptionsWithModifiers(opts...)

	g := &Grav{
		NodeUUID:  nodeUUID,
		BelongsTo: options.BelongsTo,
		Interests: options.Interests,
		bus:       newMessageBus(),
		logger:    options.Logger,
	}

	// the hub handles coordinating the transport and discovery plugins
	g.hub = initHub(nodeUUID, options, g.Connect)

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
	return g.hub.connectEndpoint(endpoint, "")
}

// ConnectBridgeTopic connects the Grav instance to a particular topic on the connected bridge
func (g *Grav) ConnectBridgeTopic(topic string) error {
	return g.hub.connectBridgeTopic(topic)
}

// Tunnel sends a message to a specific connection that has advertised it has the required capability.
// This bypasses the main Grav bus, which is why it isn't a method on Pod.
// Messages are load balanced between the connections that advertise the capability in question.
func (g *Grav) Tunnel(capability string, msg Message) error {
	return g.hub.sendTunneledMessage(capability, msg)
}

// Withdraw cancels discovery, sends withdraw messages to all peers,
// and returns when all peers have acknowledged the withdraw
func (g *Grav) Withdraw() error {
	return g.hub.withdraw()
}

// Stop stops Grav's meshing entirely, causing all connections to peers to close.
// It is reccomended to call `Withdraw` first to give peers notice and stop recieving messages
func (g *Grav) Stop() error {
	return g.hub.stop()
}

func (g *Grav) connectWithOpts(opts *podOpts) *Pod {
	pod := newPod(g.bus.busChan, opts)

	g.bus.addPod(pod)

	return pod
}
