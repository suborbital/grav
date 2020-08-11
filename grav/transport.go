package grav

// ConnectFunc represents a function that returns a pod conntected to Grav
type ConnectFunc func() *Pod

// Transport represents a Grav transport plugin
type Transport interface {
	// Serve is a transport-specific function that exposes a connection point
	Serve() error
	// ConnectEndpoint indicates to the Transport that a connection to a remote endpoint is needed
	ConnectEndpoint(string) error
	// UseConnectFunc allows Grav to provide a function for the Transport to create a new pod connected to Grav
	UseConnectFunc(ConnectFunc)
}
