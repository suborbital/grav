package grav

import (
	"sync"

	"github.com/pkg/errors"
)

// hub is responsible for coordinating the transport and discovery plugins
type hub struct {
	nodeUUID  string
	transport Transport
	discovery Discovery

	connections map[string]connection

	lock sync.Mutex
}

type connection interface {
	Send(msg Message)
}

func initHub(nodeUUID string, options *Options, tspt Transport, dscv Discovery) *hub {
	h := &hub{
		nodeUUID:    nodeUUID,
		transport:   tspt,
		discovery:   dscv,
		connections: map[string]connection{},
		lock:        sync.Mutex{},
	}

	// start transport, then discovery if each have been configured (can have transport but no discovery)
	if h.transport != nil {
		transportOpts := &TransportOpts{
			NodeUUID: nodeUUID,
			Port:     options.Port,
			Logger:   options.Logger,
		}

		go func() {
			if err := h.transport.Serve(transportOpts, h.Connect); err != nil {
				options.Logger.Error(errors.Wrap(err, "failed to Serve transport"))
			}

			if h.discovery != nil {
				discoveryOpts := &DiscoveryOpts{
					NodeUUID:      nodeUUID,
					TransportPort: transportOpts.Port,
					Logger:        options.Logger,
				}

				if err := h.discovery.Start(discoveryOpts, h.transport, h.Connect); err != nil {
					options.Logger.Error(errors.Wrap(err, "failed to Start discovery"))
				}
			}
		}()
	}

	return h
}

func (h *hub) ConnectEndpoint(endpoint string) error {
	if h.transport == nil {
		return ErrTransportNotConfigured
	}

	return h.transport.ConnectEndpoint(endpoint)
}
