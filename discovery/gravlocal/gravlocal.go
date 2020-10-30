package gravlocal

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/schollz/peerdiscovery"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vlog"
)

// GravDiscoveryLocalNetwork is a grav Discovery plugin using local network multicast
type GravDiscoveryLocalNetwork struct {
	opts   *grav.DiscoveryOpts
	Logger *vlog.Logger
}

// New creates a new local discovery plugin
func New() *GravDiscoveryLocalNetwork {
	g := &GravDiscoveryLocalNetwork{}

	return g
}

// LocalDPayload is a discovery payload
type LocalDPayload struct {
	UUID string `json:"uuid"`
	Port string `json:"port"`
	Path string `json:"path"`
}

// Start starts discovery
func (g *GravDiscoveryLocalNetwork) Start(opts *grav.DiscoveryOpts, tspt grav.Transport, connectFunc grav.ConnectFunc) error {
	g.opts = opts
	g.Logger = opts.Logger

	g.Logger.Info("[discovery-local] starting discovery")

	payloadFunc := func() []byte {
		payload := LocalDPayload{
			UUID: g.opts.NodeUUID,
			Port: opts.TransportPort,
			Path: "/meta/message",
		}

		payloadBytes, _ := json.Marshal(payload)
		return payloadBytes
	}

	notifyFunc := func(d peerdiscovery.Discovered) {
		g.Logger.Debug("[discovery-local] potential peer found:", d.Address)

		payload := LocalDPayload{}
		if err := json.Unmarshal(d.Payload, &payload); err != nil {
			g.Logger.Debug("[discovery-local] peer did not offer correct payload, discarding")
			return
		}

		if payload.UUID == g.opts.NodeUUID {
			g.Logger.Debug("[discovery-local] found self, discarding")
			return
		}

		endpoint := fmt.Sprintf("ws://%s:%s%s", d.Address, payload.Port, payload.Path)

		// transport is responsible for ensuring that each node gets a single connection.
		// the discovery can only do so much when incoming and outgoing discovery can happen
		// at the same time. ConnectEndpoint should be thread safe, also.
		if err := tspt.ConnectEndpointWithUUID(payload.UUID, endpoint, connectFunc); err != nil {
			g.Logger.Error(errors.Wrapf(err, "[discovery-local] failed to ConnectEndpoint for %s", d.Address))
			return
		}
	}

	_, err := peerdiscovery.Discover(peerdiscovery.Settings{
		Limit:       -1,
		PayloadFunc: payloadFunc,
		Delay:       10 * time.Second,
		TimeLimit:   -1,
		Notify:      notifyFunc,
		AllowSelf:   true,
	})

	return err
}
