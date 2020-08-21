package grav

import "github.com/suborbital/vektor/vlog"

// ConnectFunc represents a function that returns a pod conntected to Grav
type ConnectFunc func() *Pod

// CreateTransportFunc is a function that returns a transport
type CreateTransportFunc func(*TransportOpts, ConnectFunc) Transport

// TransportOpts is a set of options for transports
type TransportOpts struct {
	Logger vlog.Logger
	Port   int
	Custom interface{}
}

// Transport represents a Grav transport plugin
type Transport interface {
	// Serve is a transport-specific function that exposes a connection point
	Serve() error
	// ConnectEndpoint indicates to the Transport that a connection to a remote endpoint is needed
	ConnectEndpoint(string) error
}

// DefaultTransportOpts returns the default Grav Transport options
func DefaultTransportOpts() *TransportOpts {
	to := &TransportOpts{
		Logger: vlog.DefaultLogger(),
		Port:   8080,
	}

	return to
}
