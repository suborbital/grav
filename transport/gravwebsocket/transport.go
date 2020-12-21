package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vlog"
)

var upgrader = websocket.Upgrader{}

// Transport is a transport that connects Grav nodes via standard websockets
type Transport struct {
	opts *grav.TransportOpts
	log  *vlog.Logger

	incomingConnFunc func(grav.Connection)
}

// Conn implements transport.Connection and represents a websocket connection
type Conn struct {
	nodeUUID string
	conn     *websocket.Conn
	log      *vlog.Logger

	recvFunc func(nodeUUID string, msg grav.Message)
}

// New creates a new websocket transport
func New() *Transport {
	t := &Transport{}

	return t
}

// Setup creates a request server to handle incoming messages (not yet implemented)
func (t *Transport) Setup(opts *grav.TransportOpts) error {
	t.opts = opts
	t.log = opts.Logger

	return nil
}

// CreateConnection adds a websocket endpoint to emit messages to
func (t *Transport) CreateConnection(endpoint, uuid string) (grav.Connection, error) {
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	if endpointURL.Scheme == "" {
		endpointURL.Scheme = "ws"
	}

	c, _, err := websocket.DefaultDialer.Dial(endpointURL.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "[transport-websocket] failed to Dial endpoint")
	}

	conn := &Conn{
		conn:     c,
		nodeUUID: uuid,
		log:      t.log,
	}

	return conn, nil
}

// UseConnectionFunc allows Grav to set a function to be used when the transport gets a new connection
func (t *Transport) UseConnectionFunc(connFunc func(grav.Connection)) {
	t.incomingConnFunc = connFunc
}

// HTTPHandlerFunc returns an http.HandlerFunc for incoming connections
func (t *Transport) HTTPHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if t.incomingConnFunc == nil {
			t.log.ErrorString("[transport-websocket] incoming connection received, but no connFunc configured")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.log.Error(errors.Wrap(err, "[transport-websocket] failed to upgrade connection"))
			return
		}

		t.log.Debug("[transport-websocket] upgraded connection:", r.URL.String())

		conn := &Conn{
			conn:     c,
			nodeUUID: "",
			log:      t.log,
		}

		t.incomingConnFunc(conn)
	}
}

func (c *Conn) start(uuid string, conn *websocket.Conn) {
	// remove the connection from the pool upon closing
	conn.SetCloseHandler(func(code int, text string) error {
		c.log.Warn(fmt.Sprintf("[transport-websocket] connection closing with code: %d", code))
		return nil
	})

	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			c.log.Error(errors.Wrap(err, "[transport-websocket] failed to ReadMessage, terminating connection"))
			break
		}

		c.log.Debug("[transport-websocket] recieved message via", uuid)

		msg, err := grav.MsgFromBytes(message)
		if err != nil {
			c.log.Error(errors.Wrap(err, "[transport-websocket] failed to MsgFromBytes"))
			continue
		}

		t.pod.Send(msg)
	}
}

// Send sends a message to the connection
func (c *Conn) Send(msg grav.Message) error {
	msgBytes, err := msg.Marshal()
	if err != nil {
		// not exactly sure what to do here (we don't want this going into the dead letter queue)
		c.log.Error(errors.Wrap(err, "[transport-websocket] failed to Marshal message"))
		return nil
	}

	c.log.Debug("[transport-websocket] sending message to connection", c.nodeUUID)

	if err := c.conn.WriteMessage(grav.TransportMsgTypeUser, msgBytes); err != nil {
		c.log.Error(errors.Wrap(err, "[transport-websocket] failed to WriteMessage"))
		return grav.ErrConnectionClosed
	}

	c.log.Debug("[transport-websocket] sent message to connection", c.nodeUUID)

	return nil
}

// DoOutgoingHandshake performs a connection handshake and returns the UUID of the node that we're connected to
// so that it can be validated against the UUID that was provided in discovery (or if none was provided)
func (c *Conn) DoOutgoingHandshake(handshake *grav.TransportHandshake) (*grav.TransportHandshakeAck, error) {
	handshakeJSON, err := json.Marshal(handshake)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal handshake JSON")
	}

	c.log.Debug("[transport-websocket] sending handshake")

	if err := c.conn.WriteMessage(grav.TransportMsgTypeHandshake, handshakeJSON); err != nil {
		return nil, errors.Wrap(err, "failed to WriteMessage handshake")
	}

	mt, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadMessage for handshake ack, terminating connection")
	}

	if mt != grav.TransportMsgTypeHandshake {
		return nil, errors.New("first message recieved was not handshake ack")
	}

	c.log.Debug("[transport-websocket] recieved handshake ack")

	ack := grav.TransportHandshakeAck{}
	if err := json.Unmarshal(message, &ack); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal handshake ack")
	}

	return &ack, nil
}

// DoIncomingHandshake performs a connection handshake and returns the UUID of the node that we're connected to
// so that it can be validated against the UUID that was provided in discovery (or if none was provided)
func (c *Conn) DoIncomingHandshake(handshakeAck *grav.TransportHandshakeAck) (*grav.TransportHandshake, error) {
	mt, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadMessage for handshake, terminating connection")
	}

	if mt != grav.TransportMsgTypeHandshake {
		return nil, errors.New("first message recieved was not handshake")
	}

	c.log.Debug("[transport-websocket] recieved handshake")

	handshake := grav.TransportHandshake{}
	if err := json.Unmarshal(message, &handshake); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal handshake")
	}

	if c.nodeUUID != "" && handshake.UUID != c.nodeUUID {
		return nil, grav.ErrNodeUUIDMismatch
	} else if c.nodeUUID == "" {
		c.nodeUUID = handshake.UUID
	}

	ackJSON, err := json.Marshal(handshakeAck)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal handshake JSON")
	}

	c.log.Debug("[transport-websocket] sending handshake ack")

	if err := c.conn.WriteMessage(grav.TransportMsgTypeHandshake, ackJSON); err != nil {
		return nil, errors.Wrap(err, "failed to WriteMessage handshake ack")
	}

	c.log.Debug("[transport-websocket] sent handshake ack")

	return &handshake, nil
}
