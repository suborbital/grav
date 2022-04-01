package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vlog"
)

const MsgTypeWebsocketMessage = "websocket.message"

var upgrader = websocket.Upgrader{}

// Transport is a transport that connects Grav nodes via standard websockets
type Transport struct {
	opts *grav.TransportOpts
	log  *vlog.Logger

	connectionFunc func(grav.Connection)
}

// Conn implements transport.Connection and represents a websocket connection
type Conn struct {
	nodeUUID string
	log      *vlog.Logger

	conn *websocket.Conn
	lock sync.Mutex

	selfWithdrawn atomic.Value
	peerWithdrawn atomic.Value

	recvFunc grav.ReceiveFunc
	signaler *grav.WithdrawSignaler
}

// New creates a new websocket transport
func New() *Transport {
	t := &Transport{}

	return t
}

// Type returns the transport's type
func (t *Transport) Type() grav.TransportType {
	return grav.TransportTypeMesh
}

// Setup sets up the transport
func (t *Transport) Setup(opts *grav.TransportOpts, connFunc grav.ConnectFunc, findFunc grav.FindFunc) error {
	// independent serving is not yet implemented, use the HTTP handler

	t.opts = opts
	t.log = opts.Logger
	t.connectionFunc = connFunc

	return nil
}

// CreateConnection adds a websocket endpoint to emit messages to
func (t *Transport) CreateConnection(endpoint string) (grav.Connection, error) {
	if !strings.HasPrefix(endpoint, "ws") {
		endpoint = fmt.Sprintf("ws://%s", endpoint)
	}

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	c, _, err := websocket.DefaultDialer.Dial(endpointURL.String(), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "[transport-websocket] failed to Dial endpoint")
	}

	conn := &Conn{
		log:           t.log,
		conn:          c,
		selfWithdrawn: atomic.Value{},
		peerWithdrawn: atomic.Value{},
		lock:          sync.Mutex{},
	}

	conn.selfWithdrawn.Store(false)
	conn.peerWithdrawn.Store(false)

	return conn, nil
}

// ConnectBridgeTopic connects to a topic if the transport is a bridge
func (t *Transport) ConnectBridgeTopic(topic string) (grav.TopicConnection, error) {
	return nil, grav.ErrNotBridgeTransport
}

// HTTPHandlerFunc returns an http.HandlerFunc for incoming connections
func (t *Transport) HTTPHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if t.connectionFunc == nil {
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
			conn:          c,
			log:           t.log,
			selfWithdrawn: atomic.Value{},
			peerWithdrawn: atomic.Value{},
		}

		conn.selfWithdrawn.Store(false)
		conn.peerWithdrawn.Store(false)

		t.connectionFunc(conn)
	}
}

// Start begins the receiving of messages
func (c *Conn) Start(recvFunc grav.ReceiveFunc, signaler *grav.WithdrawSignaler) {
	c.recvFunc = recvFunc
	c.signaler = signaler

	go func() {
		// wait until a withdraw happens and notify peer
		<-c.signaler.Ctx.Done()

		c.log.Info("[transport-websocket] connection context canceled, sending withdraw message")

		if err := c.WriteMessage(websocket.TextMessage, []byte("WITHDRAW")); err != nil {
			if err == grav.ErrNodeWithdrawn {
				// that's fine, consider the withdraw completed
				c.selfWithdrawn.Store(true)
				c.signaler.DoneChan <- true
			} else {
				c.log.Error(errors.Wrap(err, "[transport-websocket] failed to WriteMessage for withdraw"))
			}
		}
	}()

	go func() {
		for {
			msgType, message, err := c.conn.ReadMessage()
			if err != nil {
				c.log.Error(errors.Wrap(err, "[transport-websocket] failed to ReadMessage, closing"))

				c.Close()

				break
			}

			if msgType == websocket.TextMessage {

				if string(message) == "WITHDRAW" {
					c.log.Info("[transport-websocket] peer has withdrawn, marking connection")

					if err := c.WriteMessage(websocket.TextMessage, []byte("WITHDRAW ACK")); err != nil {
						c.log.Error(errors.Wrap(err, "[transport-websocket] failed to WriteMessage for withdraw ack"))
					}

					c.peerWithdrawn.Store(true)
				} else if string(message) == "WITHDRAW ACK" {
					c.log.Info("[transport-websocket] peer acked withdraw")

					c.selfWithdrawn.Store(true)
					c.signaler.DoneChan <- true
				}

				continue
			}

			msg, err := grav.MsgFromBytes(message)
			if err != nil {
				c.log.Debug(errors.Wrap(err, "[transport-websocket] failed to MsgFromBytes, falling back to raw data").Error())

				msg = grav.NewMsg(MsgTypeWebsocketMessage, message)
			}

			c.log.Debug("[transport-websocket] received message", msg.UUID(), "via", c.nodeUUID)

			// send to the Grav instance
			c.recvFunc(msg)
		}
	}()
}

// Send sends a message to the connection
func (c *Conn) Send(msg grav.Message) error {
	msgBytes, err := msg.Marshal()
	if err != nil {
		// not exactly sure what to do here (we don't want this going into the dead letter queue)
		c.log.Error(errors.Wrap(err, "[transport-websocket] failed to Marshal message"))
		return nil
	}

	c.log.Debug("[transport-websocket] sending message", msg.UUID(), "to connection", c.nodeUUID)

	if err := c.WriteMessage(websocket.BinaryMessage, msgBytes); err != nil {
		if errors.Is(err, websocket.ErrCloseSent) {
			return grav.ErrConnectionClosed
		} else if err == grav.ErrNodeWithdrawn {
			return err
		}

		return errors.Wrap(err, "[transport-websocket] failed to WriteMessage")
	}

	c.log.Debug("[transport-websocket] sent message", msg.UUID(), "to connection", c.nodeUUID)

	return nil
}

// CanReplace returns true if the connection can be replaced
func (c *Conn) CanReplace() bool {
	return false
}

// DoOutgoingHandshake performs a connection handshake and returns the UUID of the node that we're connected to
// so that it can be validated against the UUID that was provided in discovery (or if none was provided)
func (c *Conn) DoOutgoingHandshake(handshake *grav.TransportHandshake) (*grav.TransportHandshakeAck, error) {
	handshakeJSON, err := json.Marshal(handshake)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal handshake JSON")
	}

	c.log.Debug("[transport-websocket] sending handshake")

	if err := c.WriteMessage(websocket.BinaryMessage, handshakeJSON); err != nil {
		return nil, errors.Wrap(err, "failed to WriteMessage handshake")
	}

	mt, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadMessage for handshake ack, terminating connection")
	}

	if mt != websocket.BinaryMessage {
		return nil, errors.New("first message recieved was not handshake ack")
	}

	c.log.Debug("[transport-websocket] recieved handshake ack")

	ack := grav.TransportHandshakeAck{}
	if err := json.Unmarshal(message, &ack); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal handshake ack")
	}

	c.nodeUUID = ack.UUID

	return &ack, nil
}

// DoIncomingHandshake performs a connection handshake and returns the UUID of the node that we're connected to
// so that it can be validated against the UUID that was provided in discovery (or if none was provided)
func (c *Conn) DoIncomingHandshake(handshakeCallback grav.HandshakeCallback) (*grav.TransportHandshake, error) {
	mt, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadMessage for handshake, terminating connection")
	}

	if mt != websocket.BinaryMessage {
		return nil, errors.New("first message recieved was not handshake")
	}

	c.log.Debug("[transport-websocket] recieved handshake")

	handshake := &grav.TransportHandshake{}
	if err := json.Unmarshal(message, handshake); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal handshake")
	}

	ack := handshakeCallback(handshake)

	ackJSON, err := json.Marshal(ack)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal handshake ack JSON")
	}

	c.log.Debug("[transport-websocket] sending handshake ack")

	if err := c.WriteMessage(websocket.BinaryMessage, ackJSON); err != nil {
		return nil, errors.Wrap(err, "failed to WriteMessage handshake ack")
	}

	c.log.Debug("[transport-websocket] sent handshake ack")

	c.nodeUUID = handshake.UUID

	return handshake, nil
}

// Close closes the underlying connection
func (c *Conn) Close() error {
	c.log.Debug("[transport-websocket] connection for", c.nodeUUID, "is closing")

	if err := c.conn.Close(); err != nil {
		return errors.Wrap(err, "[transport-websocket] failed to Close connection")
	}

	return nil
}

// WriteMessage is a concurrent-safe wrapper around the websocket WriteMessage
func (c *Conn) WriteMessage(messageType int, data []byte) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if withdrawn := c.peerWithdrawn.Load(); withdrawn.(bool) {
		return grav.ErrNodeWithdrawn
	}

	return c.conn.WriteMessage(messageType, data)
}
