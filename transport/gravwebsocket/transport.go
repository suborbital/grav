package gravwebsocket

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vlog"
)

var upgrader = websocket.Upgrader{}

// GravTransportWebsocket is a websocket manager that connects Grav nodes via standard websockets
type GravTransportWebsocket struct {
	pod         *grav.Pod
	connections map[string]*websocket.Conn

	log *vlog.Logger

	sync.RWMutex
}

// New creates a new websocket transport
func New(opts *grav.TransportOpts) *GravTransportWebsocket {
	g := &GravTransportWebsocket{
		pod:         nil, // will be provided upon call to Serve
		connections: map[string]*websocket.Conn{},
		log:         opts.Logger,
		RWMutex:     sync.RWMutex{},
	}

	return g
}

// Serve creates a request server to handle incoming messages (not yet implemented)
func (g *GravTransportWebsocket) Serve(pod *grav.Pod) error {
	// serving independently is not yet supported, use the handler func methods
	g.Lock()
	defer g.Unlock()

	if g.pod == nil {
		g.pod = pod
		g.pod.On(g.messageHandler())
	}

	return nil
}

// ConnectEndpoint adds a websocket endpoint to emit messages to
func (g *GravTransportWebsocket) ConnectEndpoint(endpoint string, connect grav.ConnectFunc) error {
	g.Lock()
	defer g.Unlock()

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

	// runs as long as the connection is alive and handles removing the connection when it dies
	go g.connectionHandler(c)

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

		// blocks as long as the connection is alive and handles removing the connection when it dies
		g.connectionHandler(c)
	}
}

func (g *GravTransportWebsocket) addConnection(uuid string, conn *websocket.Conn) {
	g.Lock()
	defer g.Unlock()

	g.connections[uuid] = conn
}

func (g *GravTransportWebsocket) removeConnection(uuid string) {
	g.Lock()
	defer g.Unlock()

	delete(g.connections, uuid)
}

func (g *GravTransportWebsocket) connectionHandler(conn *websocket.Conn) {
	// assign the connection a UUID and add it to the connection pool
	uuid := uuid.New().String()
	g.addConnection(uuid, conn)

	g.log.Debug("[transport-websocket] added connection", uuid)

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
