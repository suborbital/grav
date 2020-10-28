package gravhttp

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

// GravTransportHTTP is an HTTP handler manager that pushes messages into a bus
type GravTransportHTTP struct {
	pod       *grav.Pod
	endpoints []url.URL

	log *vlog.Logger

	sync.RWMutex
}

// New creates a new http transport
func New() *GravTransportHTTP {
	g := &GravTransportHTTP{
		pod:       nil, // will be provided upon call to Serve
		endpoints: []url.URL{},
		RWMutex:   sync.RWMutex{},
	}

	return g
}

// Serve creates a request server to handle incoming messages (not yet implemented)
func (g *GravTransportHTTP) Serve(opts *grav.TransportOpts, connect grav.ConnectFunc) error {
	// serving independently is not yet supported, use the handler func methods
	g.Lock()
	defer g.Unlock()

	g.log = opts.Logger

	if g.pod == nil {
		g.pod = connect()
		g.pod.On(g.messageHandler())
	}

	return nil
}

// ConnectEndpoint adds an HTTP/S endpoint to emit messages to
func (g *GravTransportHTTP) ConnectEndpoint(endpoint string, connect grav.ConnectFunc) error {
	return g.ConnectEndpointWithUUID("", endpoint, connect)
}

// ConnectEndpointWithUUID adds an HTTP/S endpoint to emit messages to
func (g *GravTransportHTTP) ConnectEndpointWithUUID(uuid, endpoint string, connect grav.ConnectFunc) error {
	g.Lock()
	defer g.Unlock()

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	if endpointURL.Scheme == "" {
		endpointURL.Scheme = "http"
	}

	g.endpoints = append(g.endpoints, *endpointURL)

	return nil
}

// HandlerFunc returns a vk handlerFunc for incoming messages
// use HTTPHAndlerFunc for a regular Go http.HandlerFunc
func (g *GravTransportHTTP) HandlerFunc() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		msg, err := grav.MsgFromRequest(r)
		if err != nil {
			return nil, vk.E(http.StatusBadRequest, "")
		}

		g.pod.Send(msg)

		return vk.R(http.StatusOK, nil), nil
	}
}

// HTTPHandlerFunc returns an http.HandlerFunc for incoming messages
func (g *GravTransportHTTP) HTTPHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg, err := grav.MsgFromRequest(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		g.pod.Send(msg)

		w.WriteHeader(http.StatusOK)
	}
}

func (g *GravTransportHTTP) messageHandler() grav.MsgFunc {
	return func(msg grav.Message) error {
		g.RLock()
		defer g.RUnlock()

		msgBytes, err := msg.Marshal()
		if err != nil {
			// not exactly sure what to do here (we don't want this going into the dead letter queue)
			g.log.Error(errors.Wrap(err, "[transport-http] failed to Marshal Message"))
			return nil
		}

		body := bytes.NewBuffer(msgBytes)

		for _, e := range g.endpoints {
			endpointURL := e

			go func() {
				req, err := http.NewRequest(http.MethodPost, endpointURL.String(), body)
				if err != nil {
					g.log.Error(errors.Wrap(err, "[transport-http] failed to create http.NewRequest"))
					return
				}

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					g.log.Error(errors.Wrap(err, "[transport-http] failed to Do request"))
					return
				}

				if resp.StatusCode != http.StatusOK {
					g.log.Error(fmt.Errorf("[transport-http] request returned non-200 status: %d", resp.StatusCode))
				}
			}()
		}

		return nil
	}
}
