package grav

import "github.com/suborbital/vektor/vlog"

// Discovery represents a discovery plugin
type Discovery interface {
	Start(Transport, ConnectFunc) error
}

// DiscoveryOpts is a set of options for transports
type DiscoveryOpts struct {
	Logger *vlog.Logger
	Custom interface{}
}

// DefaultDiscoveryOpts returns the default Grav Transport options
func DefaultDiscoveryOpts() *DiscoveryOpts {
	do := &DiscoveryOpts{
		Logger: vlog.Default(),
	}

	return do
}
