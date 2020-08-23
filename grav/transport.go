package grav

// ConnectFunc represents a function that returns a pod conntected to Grav
type ConnectFunc func() *Pod

// TransportServeOpts is a set of options for transports
type TransportServeOpts struct {
	Port   int
	Custom interface{}
}

// Transport represents a Grav transport plugin
type Transport interface {
	// Serve is a transport-specific function that exposes a connection point
	Serve(*TransportServeOpts) error
	// ConnectEndpoint indicates to the Transport that a connection to a remote endpoint is needed
	ConnectEndpoint(string, ConnectFunc) error
}

// DefaultTransportServeOpts returns the default Grav Transport options
func DefaultTransportServeOpts() *TransportServeOpts {
	to := &TransportServeOpts{
		Port: 8080,
	}

	return to
}
