package grav

import "github.com/suborbital/vektor/vlog"

// TransportMsgTypeUser and others represent internal Transport message types used for handshakes and metadata transfer
const (
	TransportMsgTypeUser = 1
)

// ConnectFunc represents a function that returns a pod conntected to Grav
type ConnectFunc func() *Pod

// TransportOpts is a set of options for transports
type TransportOpts struct {
	Port   int
	Logger *vlog.Logger
	Custom interface{}
}

// Transport represents a Grav transport plugin
type Transport interface {
	// Serve is a transport-specific function that exposes a connection point
	Serve(*Pod) error
	// ConnectEndpoint indicates to the Transport that a connection to a remote endpoint is needed
	ConnectEndpoint(string, ConnectFunc) error
	// ConnectEndpointWithUUID connects to an endpoint with a known identifier
	ConnectEndpointWithUUID(string, string, ConnectFunc) error
}

// DefaultTransportOpts returns the default Grav Transport options
func DefaultTransportOpts() *TransportOpts {
	to := &TransportOpts{
		Port:   8080,
		Logger: vlog.Default(),
	}

	return to
}
