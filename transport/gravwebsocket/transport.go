package gravwebsocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vlog"
)

var upgrader = websocket.Upgrader{}

// GravTransportWebsocket is a websocket manager that connects Grav nodes via standard websockets
type GravTransportWebsocket struct {
	opts        *grav.TransportOpts
	pod         *grav.Pod
	connections map[string]*websocket.Conn

	log *vlog.Logger

	sync.RWMutex
}

// New creates a new websocket transport
func New() *GravTransportWebsocket {
	g := &GravTransportWebsocket{
		pod:         nil, // will be provided upon call to Serve
		connections: map[string]*websocket.Conn{},
		RWMutex:     sync.RWMutex{},
	}

	return g
}

// Serve creates a request server to handle incoming messages (not yet implemented)
func (g *GravTransportWebsocket) Serve(opts *grav.TransportOpts, connect grav.ConnectFunc) error {
	// serving independently is not yet supported, use the handler func methods
	g.Lock()
	defer g.Unlock()

	g.opts = opts
	g.log = opts.Logger

	if g.pod == nil {
		g.pod = connect()
		g.pod.On(g.messageHandler())
	}

	return nil
}

// ConnectEndpoint adds a websocket endpoint to emit messages to
func (g *GravTransportWebsocket) ConnectEndpoint(endpoint string, connect grav.ConnectFunc) error {
	return g.ConnectEndpointWithUUID("", endpoint, connect)
}

// ConnectEndpointWithUUID adds a websocket endpoint to emit messages to
func (g *GravTransportWebsocket) ConnectEndpointWithUUID(uuid, endpoint string, connect grav.ConnectFunc) error {
	g.Lock()
	defer g.Unlock()

	if _, exists := g.connections[uuid]; exists {
		// nothing to be done, it's likely a re-discovered peer
		return nil
	}

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	if endpointURL.Scheme == "" {
		endpointURL.Scheme = "ws"
	}

	c, _, err := websocket.DefaultDialer.Dial(endpointURL.String(), nil)
	if err != nil {
		return errors.Wrap(err, "[transport-websocket] failed to Dial endpoint")
	}

	ackUUID, err := g.performOutgoingHandshake(c)
	if err != nil {
		c.Close()
		return errors.Wrap(err, "[transport-websocket] failed to performOutgoingHandshake")
	} else if uuid == "" {
		uuid = ackUUID
	} else if ackUUID != uuid {
		c.Close()
		return fmt.Errorf("[transport-websocket] handshake ack UUID (%s) did not match discovery UUID (%s)", ackUUID, uuid)
	}

	// runs as long as the connection is alive and handles removing the connection when it dies
	go g.connectionHandler(uuid, c)

	return nil
}

// HTTPHandlerFunc returns an http.HandlerFunc for incoming connections
func (g *GravTransportWebsocket) HTTPHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			g.log.Error(errors.Wrap(err, "[transport-websocket] failed to upgrade connection"))
			return
		}

		g.log.Debug("[transport-websocket] upgraded connection:", r.URL.String())

		uuid, err := g.performIncomingHandshake(c)
		if err != nil {
			g.log.Error(errors.Wrap(err, "[transport-websocket] faield to performIncomingHandshake on connection"))
			c.Close()
			return
		}

		g.connectionHandler(uuid, c)
	}
}

func (g *GravTransportWebsocket) addConnection(uuid string, conn *websocket.Conn) error {
	g.Lock()
	defer g.Unlock()

	if _, exists := g.connections[uuid]; exists {
		return fmt.Errorf("connection with UUID %s already exists", uuid)
	}

	g.connections[uuid] = conn

	return nil
}

func (g *GravTransportWebsocket) removeConnection(uuid string) {
	g.Lock()
	defer g.Unlock()

	delete(g.connections, uuid)
}

func (g *GravTransportWebsocket) connectionHandler(uuid string, conn *websocket.Conn) {
	// add to the connection pool
	if err := g.addConnection(uuid, conn); err != nil {
		g.log.Debug("[transport-websocket] failed to add connection for", uuid, ":", err.Error())
		conn.Close()
		return
	}

	g.log.Info("[transport-websocket] added connection", uuid)

	// remove the connection from the pool upon closing
	conn.SetCloseHandler(func(code int, text string) error {
		g.log.Warn(fmt.Sprintf("[transport-websocket] connection closing with code: %d", code))

		g.removeConnection(uuid)

		return nil
	})

	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			g.log.Error(errors.Wrap(err, "[transport-websocket] failed to ReadMessage, terminating connection"))
			g.removeConnection(uuid)
			break
		}

		g.log.Debug("[transport-websocket] recieved message via", uuid)

		msg, err := grav.MsgFromBytes(message)
		if err != nil {
			g.log.Error(errors.Wrap(err, "[transport-websocket] failed to MsgFromBytes"))
			continue
		}

		g.pod.Send(msg)
	}
}

func (g *GravTransportWebsocket) messageHandler() grav.MsgFunc {
	return func(msg grav.Message) error {
		g.RLock()
		defer g.RUnlock()

		msgBytes, err := msg.Marshal()
		if err != nil {
			// not exactly sure what to do here (we don't want this going into the dead letter queue)
			g.log.Error(errors.Wrap(err, "[transport-websocket] failed to Marshal message"))
			return nil
		}

		for uuid := range g.connections {
			realUUID := uuid
			conn := g.connections[realUUID]

			go func() {
				g.log.Debug("[transport-websocket] sending message to connection", realUUID)

				if err := conn.WriteMessage(grav.TransportMsgTypeUser, msgBytes); err != nil {
					g.log.Error(errors.Wrap(err, "[transport-websocket] failed to WriteMessage"))
					g.removeConnection(realUUID)
					conn.Close()
				}

				g.log.Debug("[transport-websocket] sent message to connection", realUUID)
			}()
		}

		return nil
	}
}

// performOutgoingHandshake performs a connection handshake and returns the UUID of the node that we're connected to
// so that it can be validated against the UUID that was provided in discovery (or if none was provided)
func (g *GravTransportWebsocket) performOutgoingHandshake(c *websocket.Conn) (string, error) {
	handshakeMsg := grav.TransportHandshake{
		UUID: g.opts.NodeUUID,
	}

	handshakeJSON, err := json.Marshal(handshakeMsg)
	if err != nil {
		return "", errors.Wrap(err, "failed to Marshal handshake JSON")
	}

	g.log.Debug("[transport-websocket] sending handshake")

	if err := c.WriteMessage(grav.TransportMsgTypeHandshake, handshakeJSON); err != nil {
		return "", errors.Wrap(err, "failed to WriteMessage handshake")
	}

	mt, message, err := c.ReadMessage()
	if err != nil {
		return "", errors.Wrap(err, "failed to ReadMessage for handshake ack, terminating connection")
	}

	if mt != grav.TransportMsgTypeHandshake {
		return "", errors.New("first message recieved was not handshake ack")
	}

	g.log.Debug("[transport-websocket] recieved handshake ack")

	ack := grav.TransportHandshakeAck{}
	if err := json.Unmarshal(message, &ack); err != nil {
		return "", errors.Wrap(err, "failed to Unmarshal handshake ack")
	}

	return ack.UUID, nil
}

// performIncomingHandshake performs a connection handshake and returns the UUID of the node that we're connected to
// so that it can be validated against the UUID that was provided in discovery (or if none was provided)
func (g *GravTransportWebsocket) performIncomingHandshake(c *websocket.Conn) (string, error) {
	mt, message, err := c.ReadMessage()
	if err != nil {
		return "", errors.Wrap(err, "failed to ReadMessage for handshake, terminating connection")
	}

	if mt != grav.TransportMsgTypeHandshake {
		return "", errors.New("first message recieved was not handshake")
	}

	g.log.Debug("[transport-websocket] recieved handshake")

	handshake := grav.TransportHandshake{}
	if err := json.Unmarshal(message, &handshake); err != nil {
		return "", errors.Wrap(err, "failed to Unmarshal handshake")
	}

	ackMsg := grav.TransportHandshakeAck{
		UUID: g.opts.NodeUUID,
	}

	ackJSON, err := json.Marshal(ackMsg)
	if err != nil {
		return "", errors.Wrap(err, "failed to Marshal handshake JSON")
	}

	g.log.Debug("[transport-websocket] sending handshake ack")

	if err := c.WriteMessage(grav.TransportMsgTypeHandshake, ackJSON); err != nil {
		return "", errors.Wrap(err, "failed to WriteMessage handshake")
	}

	return handshake.UUID, nil
}
