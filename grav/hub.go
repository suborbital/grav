package grav

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

// hub is responsible for coordinating the transport and discovery plugins
type hub struct {
	nodeUUID  string
	transport Transport
	discovery Discovery
	pod       *Pod
	log       *vlog.Logger

	connections map[string]Connection

	lock sync.RWMutex
}

func initHub(nodeUUID string, options *Options, tspt Transport, dscv Discovery, pod *Pod) *hub {
	h := &hub{
		nodeUUID:    nodeUUID,
		transport:   tspt,
		discovery:   dscv,
		pod:         pod,
		log:         options.Logger,
		connections: map[string]Connection{},
		lock:        sync.RWMutex{},
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
		pod.On(h.outgoingMessageHandler())

		go func() {
			h.transport.UseConnectionFunc(h.handleIncomingConnection)

			if err := h.transport.Setup(transportOpts); err != nil {
				options.Logger.Error(errors.Wrap(err, "failed to Setup transport"))
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
				h.discovery.UseDiscoveryFunc(h.discoveryHandler())

				if err := h.discovery.Start(discoveryOpts); err != nil {
					options.Logger.Error(errors.Wrap(err, "failed to Start discovery"))
				}
			}()
		}
	}

	return h
}

// connectEndpoint allows the Grav instance owner to "manually" connect to an endpoint
func (h *hub) connectEndpoint(endpoint, uuid string) error {
	if h.transport == nil {
		return ErrTransportNotConfigured
	}

	h.log.Debug("connecting to endpoint", endpoint)

	conn, err := h.transport.CreateConnection(endpoint)
	if err != nil {
		return errors.Wrap(err, "failed to transport.CreateConnection")
	}

	h.setupOutgoingConnection(conn, uuid)

	return nil
}

func (h *hub) discoveryHandler() func(endpoint string, uuid string) {
	return func(endpoint string, uuid string) {
		if uuid == h.nodeUUID {
			h.log.Debug("discovered self, discarding")
			return
		}

		if h.hasConnection(uuid) {
			h.log.Debug("discovered duplicate connection, ignoring")
			return
		}

		if err := h.connectEndpoint(endpoint, uuid); err != nil {
			h.log.Error(errors.Wrap(err, "failed to connectEndpoint for discovered peer"))
		}
	}
}

func (h *hub) setupOutgoingConnection(connection Connection, uuid string) {
	connection.UseReceiveFunc(h.incomingMessageHandler())

	handshake := &TransportHandshake{h.nodeUUID}

	ack, err := connection.DoOutgoingHandshake(handshake)
	if err != nil {
		h.log.Error(errors.Wrap(err, "failed to connection.DoOutgoingHandshake"))
		connection.Close()
		return
	}

	if uuid == "" {
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

	if h.hasConnection(uuid) {
		h.log.Debug("encountered duplicate connection, discarding")
		connection.Close()
		return
	}

	h.addConnection(connection, uuid)
}

func (h *hub) handleIncomingConnection(connection Connection) {
	connection.UseReceiveFunc(h.incomingMessageHandler())

	ack := &TransportHandshakeAck{h.nodeUUID}

	handshake, err := connection.DoIncomingHandshake(ack)
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

	if h.hasConnection(handshake.UUID) {
		h.log.Debug("encountered duplicate connection, discarding")
		connection.Close()
		return
	}

	h.addConnection(connection, handshake.UUID)
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

				if err := conn.Send(msg); err != nil {
					if errors.Is(err, ErrConnectionClosed) {
						h.log.Debug("attempted to send on closed connection, will remove")
					} else {
						h.log.Warn("error sending to connection", uuid, ":", err.Error())
					}

					h.removeConnection(uuid)
				}
			}()
		}

		return nil
	}
}

func (h *hub) incomingMessageHandler() func(nodeUUID string, msg Message) {
	return func(nodeUUID string, msg Message) {
		h.log.Debug("received message ", msg.UUID(), "from node", nodeUUID)

		h.pod.Send(msg)
	}
}

func (h *hub) addConnection(connection Connection, uuid string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.log.Debug("adding connection for", uuid)

	connection.Start()

	h.connections[uuid] = connection
}

func (h *hub) removeConnection(uuid string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.log.Debug("removing connection for", uuid)

	delete(h.connections, uuid)
}

func (h *hub) hasConnection(uuid string) bool {
	h.lock.RLock()
	defer h.lock.RUnlock()

	_, exists := h.connections[uuid]

	return exists
}
