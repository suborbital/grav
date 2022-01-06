package grav

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

const capabilityBufferSize = 256

// hub is responsible for coordinating the transport and discovery plugins
type hub struct {
	nodeUUID     string
	belongsTo    string
	capabilities []string
	transport    Transport
	discovery    Discovery
	log          *vlog.Logger
	pod          *Pod
	connectFunc  func() *Pod

	connections      map[string]*connectionHolder
	topicConnections map[string]TopicConnection

	capabilityUUIDBuffers map[string]*MsgBuffer

	lock sync.RWMutex
}

type connectionHolder struct {
	Conn         Connection
	BelongsTo    string
	Capabilities []string
}

func initHub(nodeUUID string, options *Options, connectFunc func() *Pod) *hub {
	h := &hub{
		nodeUUID:              nodeUUID,
		belongsTo:             options.BelongsTo,
		capabilities:          options.Capabilities,
		transport:             options.Transport,
		discovery:             options.Discovery,
		log:                   options.Logger,
		pod:                   connectFunc(),
		connectFunc:           connectFunc,
		connections:           map[string]*connectionHolder{},
		topicConnections:      map[string]TopicConnection{},
		capabilityUUIDBuffers: map[string]*MsgBuffer{},
		lock:                  sync.RWMutex{},
	}

	// start transport, then discovery if each have been configured (can have transport but no discovery)
	if h.transport != nil {
		transportOpts := &TransportOpts{
			NodeUUID: nodeUUID,
			Port:     options.Port,
			URI:      options.URI,
			Logger:   options.Logger,
		}

		// setup messages to be sent to all active connections
		h.pod.On(h.outgoingMessageHandler())

		go func() {
			if err := h.transport.Setup(transportOpts, h.handleIncomingConnection, h.findConnection); err != nil {
				options.Logger.Error(errors.Wrap(err, "failed to Setup transport"))
			}

			// if Grav's context is ever canceled, close all connections
			done := options.Context.Done()
			if done != nil {
				<-done

				if h.discovery != nil {
					h.discovery.Stop()
				}

				h.lock.Lock()
				defer h.lock.Unlock()

				for uuid := range h.connections {
					conn := h.connections[uuid]
					conn.Conn.Close()
					delete(h.connections, uuid)
				}
			}
		}()

		if h.discovery != nil {
			discoveryOpts := &DiscoveryOpts{
				NodeUUID:      nodeUUID,
				TransportPort: transportOpts.Port,
				TransportURI:  transportOpts.URI,
				Logger:        options.Logger,
			}

			go func() {
				if err := h.discovery.Start(discoveryOpts, h.discoveryHandler()); err != nil {
					options.Logger.Error(errors.Wrap(err, "failed to Start discovery"))
				}
			}()
		}
	}

	return h
}

func (h *hub) discoveryHandler() func(endpoint string, uuid string) {
	return func(endpoint string, uuid string) {
		if uuid == h.nodeUUID {
			h.log.Debug("discovered self, discarding")
			return
		}

		// connectEndpoint does this check as well, but it's better to do it here as well
		// as it reduces the number of extraneous outgoing handshakes that get attempted.
		if existing, exists := h.findConnection(uuid); exists {
			if !existing.CanReplace() {
				h.log.Debug("encountered duplicate connection, discarding")
				return
			}
		}

		if err := h.connectEndpoint(endpoint, uuid); err != nil {
			h.log.Error(errors.Wrap(err, "failed to connectEndpoint for discovered peer"))
		}
	}
}

// connectEndpoint creates a new outgoing connection
func (h *hub) connectEndpoint(endpoint, uuid string) error {
	if h.transport == nil {
		return ErrTransportNotConfigured
	}

	if h.transport.Type() == TransportTypeBridge {
		return ErrBridgeOnlyTransport
	}

	h.log.Debug("connecting to endpoint", endpoint)

	conn, err := h.transport.CreateConnection(endpoint)
	if err != nil {
		return errors.Wrap(err, "failed to transport.CreateConnection")
	}

	h.setupOutgoingConnection(conn, uuid)

	return nil
}

// connectBridgeTopic creates a new outgoing connection
func (h *hub) connectBridgeTopic(topic string) error {
	if h.transport == nil {
		return ErrTransportNotConfigured
	}

	if h.transport.Type() != TransportTypeBridge {
		return ErrNotBridgeTransport
	}

	h.log.Debug("connecting to topic", topic)

	conn, err := h.transport.ConnectBridgeTopic(topic)
	if err != nil {
		return errors.Wrap(err, "failed to transport.CreateConnection")
	}

	h.addTopicConnection(conn, topic)

	return nil
}

func (h *hub) setupOutgoingConnection(connection Connection, uuid string) {
	handshake := &TransportHandshake{h.nodeUUID, h.belongsTo, h.capabilities}

	ack, err := connection.DoOutgoingHandshake(handshake)
	if err != nil {
		h.log.Error(errors.Wrap(err, "failed to connection.DoOutgoingHandshake"))
		connection.Close()
		return
	}

	if !ack.Accept {
		h.log.Debug("connection handshake was not accepted, terminating connection")
		connection.Close()
		return
	} else if uuid == "" {
		if ack.UUID == "" {
			h.log.ErrorString("connection handshake returned empty UUID, terminating connection")
			connection.Close()
			return
		}

		uuid = ack.UUID
	} else if ack.UUID != uuid {
		h.log.ErrorString("connection handshake Ack did not match Discovery Ack, terminating connection")
		connection.Close()
		return
	}

	h.setupNewConnection(connection, uuid, ack.BelongsTo, ack.Capabilities)
}

func (h *hub) handleIncomingConnection(connection Connection) {
	var ack *TransportHandshakeAck

	callback := func(handshake *TransportHandshake) *TransportHandshakeAck {
		ack = &TransportHandshakeAck{
			Accept: true,
			UUID:   h.nodeUUID,
		}

		if handshake.BelongsTo != h.belongsTo && handshake.BelongsTo != "*" {
			ack.Accept = false
		} else {
			ack.BelongsTo = h.belongsTo
			ack.Capabilities = h.capabilities
		}

		return ack
	}

	handshake, err := connection.DoIncomingHandshake(callback)
	if err != nil {
		h.log.Error(errors.Wrap(err, "failed to connection.DoIncomingHandshake"))
		connection.Close()
		return
	}

	if handshake.UUID == "" {
		h.log.ErrorString("connection handshake returned empty UUID, terminating connection")
		connection.Close()
		return
	}

	if !ack.Accept {
		h.log.Debug("rejecting connection with incompatible BelongsTo", handshake.BelongsTo)
		connection.Close()
		return
	}

	h.setupNewConnection(connection, handshake.UUID, handshake.BelongsTo, handshake.Capabilities)
}

func (h *hub) setupNewConnection(connection Connection, uuid, belongsTo string, capabilities []string) {
	// if an existing connection is found, check if it can be replaced and do so if possible
	if existing, exists := h.findConnection(uuid); exists {
		if !existing.CanReplace() {
			connection.Close()
			h.log.Debug("encountered duplicate connection, discarding")
		} else {
			existing.Close()
			h.replaceConnection(connection, uuid, belongsTo, capabilities)
		}
	} else {
		h.addConnection(connection, uuid, belongsTo, capabilities)
	}
}

func (h *hub) outgoingMessageHandler() MsgFunc {
	return func(msg Message) error {
		// read-lock while dispatching all of the goroutines to prevent concurrent read/write
		h.lock.RLock()
		defer h.lock.RUnlock()

		for u := range h.connections {
			uuid := u
			conn := h.connections[uuid]

			go func() {
				h.log.Debug("sending message", msg.UUID(), "to", uuid)

				if err := conn.Conn.Send(msg); err != nil {
					if errors.Is(err, ErrConnectionClosed) {
						h.log.Debug("attempted to send on closed connection, will remove")
					} else {
						h.log.Warn("error sending to connection, will remove", uuid, ":", err.Error())
					}

					h.removeConnection(uuid)
				}
			}()
		}

		return nil
	}
}

func (h *hub) incomingMessageHandler(uuid string) ReceiveFunc {
	return func(msg Message) {
		h.log.Debug("received message ", msg.UUID(), "from node", uuid)

		h.pod.Send(msg)
	}
}

func (h *hub) addConnection(connection Connection, uuid, belongsTo string, capabilities []string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.log.Debug("adding connection for", uuid)

	connection.Start(h.incomingMessageHandler(uuid))

	holder := &connectionHolder{
		Conn:         connection,
		BelongsTo:    belongsTo,
		Capabilities: capabilities,
	}

	h.connections[uuid] = holder

	for _, c := range capabilities {
		if _, exists := h.capabilityUUIDBuffers[c]; !exists {
			h.capabilityUUIDBuffers[c] = NewMsgBuffer(capabilityBufferSize)
		}

		h.capabilityUUIDBuffers[c].Push(NewMsg(c, []byte(uuid)))
	}
}

func (h *hub) addTopicConnection(connection TopicConnection, topic string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.log.Debug("adding bridge connection for", topic)

	connection.Start(h.connectFunc())

	h.topicConnections[topic] = connection
}

func (h *hub) replaceConnection(newConnection Connection, uuid, belongsTo string, capabilities []string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.log.Debug("replacing connection for", uuid)

	delete(h.connections, uuid)

	newConnection.Start(h.incomingMessageHandler(uuid))

	newHolder := &connectionHolder{
		Conn:         newConnection,
		BelongsTo:    belongsTo,
		Capabilities: capabilities,
	}

	h.connections[uuid] = newHolder
}

func (h *hub) removeConnection(uuid string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.log.Debug("removing connection for", uuid)

	delete(h.connections, uuid)
}

func (h *hub) findConnection(uuid string) (Connection, bool) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	conn, exists := h.connections[uuid]
	if exists && conn.Conn != nil {
		return conn.Conn, true
	}

	return nil, false
}

func (h *hub) sendTunneledMessage(capability string, msg Message) error {
	buffer, exists := h.capabilityUUIDBuffers[capability]
	if !exists {
		return ErrTunnelNotEstablished
	}

	// iterate a reasonable number of times to find a connection that's not removed or dead
	for i := 0; i < capabilityBufferSize; i++ {
		uuid := string(buffer.Next().Data())

		h.lock.RLock()
		conn, exists := h.connections[uuid]
		h.lock.RUnlock()

		if exists && conn.Conn != nil {
			if err := conn.Conn.Send(msg); err != nil {
				h.log.Error(errors.Wrap(err, "failed to Send on tunneled connection, will remove"))
				h.removeConnection(uuid)
			} else {
				return nil
			}
		}
	}

	return ErrTunnelNotEstablished
}
