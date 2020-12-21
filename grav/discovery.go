package grav

import "github.com/suborbital/vektor/vlog"

// Discovery represents a discovery plugin
type Discovery interface {
	// Start is called to start the Discovery plugin
	Start(*DiscoveryOpts) error
	// UseDiscoveryFunc gives the plugin a function to call when a new peer is discovered
	UseDiscoveryFunc(func(endpoint string, uuid string))
}

// DiscoveryOpts is a set of options for transports
type DiscoveryOpts struct {
	NodeUUID      string
	TransportPort string
	TransportURI  string
	Logger        *vlog.Logger
	Custom        interface{}
}
