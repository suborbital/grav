package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

const gravHeaderKey = "X-Grav-UUID"
const gravHandshakeHeaderKey = "X-Grav-Handshake-UUID"

// Transport is an HTTP handler manager that pushes messages into a bus
type Transport struct {
	selfUUID string

	connectionFunc grav.ConnectFunc
	findFunc       grav.FindFunc

	log *vlog.Logger
}

// Endpoint represents an HTTP endpoint that is "connected"
type Endpoint struct {
	selfUUID string
	nodeUUID string

	URL *url.URL

	receiveFunc func(msg grav.Message)

	log *vlog.Logger
}

// New creates a new http transport
func New() *Transport {
	g := &Transport{}

	return g
}

// Type returns the transport's type
func (t *Transport) Type() grav.TransportType {
	return grav.TransportTypeMesh
}

// Setup sets the transport up for connections
func (t *Transport) Setup(opts *grav.TransportOpts, connFunc grav.ConnectFunc, findFunc grav.FindFunc) error {
	// serving independently is not yet supported, use the handler func methods

	t.selfUUID = opts.NodeUUID
	t.log = opts.Logger
	t.connectionFunc = connFunc
	t.findFunc = findFunc

	return nil
}

// CreateConnection adds an HTTP/S endpoint to emit messages to
func (t *Transport) CreateConnection(endpoint string) (grav.Connection, error) {
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = fmt.Sprintf("http://%s", endpoint)
	}

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	ept := &Endpoint{
		selfUUID: t.selfUUID,
		URL:      endpointURL,
	}

	return ept, nil
}

// ConnectBridgeTopic connects to a topic if the transport is a bridge
func (t *Transport) ConnectBridgeTopic(topic string) (grav.TopicConnection, error) {
	return nil, grav.ErrNotBridgeTransport
}

// HandlerFunc returns a vk handlerFunc for incoming messages
// use HTTPHAndlerFunc for a regular Go http.HandlerFunc
func (t *Transport) HandlerFunc() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		ctx.Log.Debug("handling request (vk)")

		resp, err := t.handleRequest(r)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to handleRequest"))
		}

		return resp, err
	}
}

// HTTPHandlerFunc returns an http.HandlerFunc for incoming messages
func (t *Transport) HTTPHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.log.Debug("handling request")

		resp, err := t.handleRequest(r)
		if err != nil {
			t.log.Error(errors.Wrap(err, "failed to handleRequest"))

			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if resp != nil {
			respJSON, err := json.Marshal(resp)
			if err != nil {
				t.log.Error(errors.Wrap(err, "failed to Marshal response"))

				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Write(respJSON)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func (t *Transport) handleRequest(req *http.Request) (interface{}, error) {
	handshakeUUID := req.Header.Get(gravHandshakeHeaderKey)
	if handshakeUUID != "" {
		t.log.Debug("responding to handshake for node", handshakeUUID)

		ept := &Endpoint{
			nodeUUID: handshakeUUID,
			log:      t.log,
		}

		t.connectionFunc(ept)

		ack := &grav.TransportHandshakeAck{UUID: t.selfUUID}
		return ack, nil
	}

	nodeUUID := req.Header.Get(gravHeaderKey)
	if nodeUUID == "" {
		return nil, vk.E(http.StatusBadRequest, "missing grav header")
	}

	t.log.Debug("received request from node", nodeUUID)

	// ask Grav if there is a Connection for the given node UUID
	conn, exists := t.findFunc(nodeUUID)
	if !exists {
		return nil, vk.E(http.StatusBadRequest, "handshake needed")
	}

	ept := conn.(*Endpoint)

	if err := ept.handleMessageRequest(req); err != nil {
		return nil, vk.E(http.StatusInternalServerError, "failed to handle message request")
	}

	return nil, nil
}

// Start "starts" the connection
func (e *Endpoint) Start(recvFunc grav.ReceiveFunc) {
	e.receiveFunc = recvFunc
}

// Send sends a message to the connection
func (e *Endpoint) Send(msg grav.Message) error {
	// if this is an incoming connection, then we don't know where to send
	if e.URL == nil {
		e.log.Debug("skipping send to incoming connection")
		return nil
	}

	msgBytes, err := msg.Marshal()
	if err != nil {
		return errors.Wrap(err, "[transport-http] failed to Marshal message")
	}

	headers := map[string]string{gravHeaderKey: e.selfUUID}
	_, err = e.doRequest(msgBytes, headers)
	if err != nil {
		return errors.Wrap(err, "[transport-http] failed to doRequest")
	}

	return nil
}

// CanReplace returns true if the connection can be replaced
func (e *Endpoint) CanReplace() bool {
	// since incoming connections don't have a URL and outgoing connections do, we allow
	// an incoming connection to be replaced by an outgoing connection if they have the same UUID
	return e.URL == nil
}

// DoOutgoingHandshake does an outgoing handshake
func (e *Endpoint) DoOutgoingHandshake(handshake *grav.TransportHandshake) (*grav.TransportHandshakeAck, error) {
	resp, err := e.doRequest(nil, map[string]string{gravHandshakeHeaderKey: handshake.UUID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to doRequest")
	}

	ack := grav.TransportHandshakeAck{}
	if err := json.Unmarshal(resp, &ack); err != nil {
		return nil, errors.Wrap(err, "[transport-http] failed to unmarshal Ack")
	}

	e.nodeUUID = ack.UUID

	return &ack, nil
}

// DoIncomingHandshake does an incoming handshake
func (e *Endpoint) DoIncomingHandshake(ack *grav.TransportHandshakeAck) (*grav.TransportHandshake, error) {
	// this is a "fake handshake" as the node UUID was already surmised during the initial request
	handshake := &grav.TransportHandshake{UUID: e.nodeUUID}

	return handshake, nil
}

// Close "closes" the connection
func (e *Endpoint) Close() {
	// nothing to do
}

func (e *Endpoint) handleMessageRequest(req *http.Request) error {
	msg, err := grav.MsgFromRequest(req)
	if err != nil {
		return errors.Wrap(err, "[transport-http] failed to MsgFromRequest")
	}

	e.receiveFunc(msg)

	return nil
}

func (e *Endpoint) doRequest(body []byte, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, e.URL.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(err, "[transport-http] failed to create http.NewRequest")
	}

	header := req.Header
	for key, val := range headers {
		header.Set(key, val)
	}

	req.Header = header

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "[transport-http] failed to Do request")
	}

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("[transport-http] request returned non-200 status: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadAll response body")
	}

	return respBody, nil
}
