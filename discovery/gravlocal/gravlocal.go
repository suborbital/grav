package gravlocal

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/schollz/peerdiscovery"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vlog"
)

// GravDiscoveryLocalNetwork is a grav Discovery plugin using local network multicast
type GravDiscoveryLocalNetwork struct {
	Logger *vlog.Logger
}

// New creates a new local discovery plugin
func New(opts *grav.DiscoveryOpts) *GravDiscoveryLocalNetwork {
	g := &GravDiscoveryLocalNetwork{
		Logger: opts.Logger,
	}

	return g
}

// LocalDPayload is a discovery payload
type LocalDPayload struct {
	UUID string `json:"uuid"`
	Port string `json:"port"`
	Path string `json:"path"`
}

// Start starts discovery
func (g *GravDiscoveryLocalNetwork) Start(tspt grav.Transport, connectFunc grav.ConnectFunc) error {
	g.Logger.Info("[discovery-local] starting discovery")

	selfUUID := uuid.New().String()

	payloadFunc := func() []byte {
		payload := LocalDPayload{
			UUID: selfUUID,
			Port: "8081",
			Path: "/meta/message",
		}

		payloadBytes, _ := json.Marshal(payload)
		return payloadBytes
	}

	connected := map[string]bool{}
	lock := sync.Mutex{}

	notifyFunc := func(d peerdiscovery.Discovered) {
		g.Logger.Debug("[discovery-local] potential peer found:", d.Address)

		payload := LocalDPayload{}
		if err := json.Unmarshal(d.Payload, &payload); err != nil {
			g.Logger.Debug("[discovery-local] peer did not offer correct payload, discarding")
			return
		}

		if payload.UUID == selfUUID {
			g.Logger.Debug("[discovery-local] found self, discarding")
			return
		} else {
			lock.Lock()
			if yup, exists := connected[payload.UUID]; yup && exists {
				g.Logger.Debug("[discovery-local] found existing peer, discarding")
				lock.Unlock()
				return
			}

			connected[payload.UUID] = true

			lock.Unlock()
		}

		endpoint := fmt.Sprintf("ws://%s:%s%s", d.Address, payload.Port, payload.Path)

		// transport is responsible for ensuring that each node gets a single connection.
		// the discovery can only do so much when incoming and outgoing discovery can happen
		// at the same time. ConnectEndpoint should be thread safe, also.
		if err := tspt.ConnectEndpointWithUUID(payload.UUID, endpoint, connectFunc); err != nil {
			g.Logger.Error(errors.Wrapf(err, "[discovery-local] failed to ConnectEndpoint for %s", d.Address))
			return
		}

		g.Logger.Debug("[discovery-local] connected to", d.Address)
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
