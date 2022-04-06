package grav

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

const capabilityBufferSize = 256

// hub is responsible for coordinating the transport and discovery plugins
type hub struct {
	nodeUUID    string
	belongsTo   string
	interests   []string
	mesh        MeshTransport
	bridge      BridgeTransport
	discovery   Discovery
	context     context.Context
	cancelFunc  func()
	log         *vlog.Logger
	pod         *Pod
	connectFunc func() *Pod

	meshConnections   map[string]*connectionHandler
	bridgeConnections map[string]BridgeConnection

	capabilityUUIDBuffers map[string]*MsgBuffer

	lock sync.RWMutex
}

func initHub(nodeUUID string, options *Options, connectFunc func() *Pod) *hub {
	ctx, cancel := context.WithCancel(context.Background())

	h := &hub{
		nodeUUID:              nodeUUID,
		belongsTo:             options.BelongsTo,
		interests:             options.Interests,
		mesh:                  options.MeshTransport,
		bridge:                options.BridgeTransport,
		discovery:             options.Discovery,
		context:               ctx,
		cancelFunc:            cancel,
		log:                   options.Logger,
		connectFunc:           connectFunc,
		meshConnections:       map[string]*connectionHandler{},
		bridgeConnections:     map[string]BridgeConnection{},
		capabilityUUIDBuffers: map[string]*MsgBuffer{},
		lock:                  sync.RWMutex{},
	}

	// start mesh transport, then discovery if each have been configured (can have transport but no discovery)
	if h.mesh != nil {
		transportOpts := &MeshOptions{
			NodeUUID: nodeUUID,
			Port:     options.Port,
			URI:      options.URI,
			Logger:   options.Logger,
		}

		go func() {
			if err := h.mesh.Setup(transportOpts, h.handleIncomingConnection); err != nil {
				h.log.Error(errors.Wrap(err, "failed to Setup transport"))
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

	if h.bridge != nil {
		transportOpts := &BridgeOptions{
			NodeUUID: nodeUUID,
			Logger:   options.Logger,
		}

		go func() {
			if err := h.bridge.Setup(transportOpts); err != nil {
				h.log.Error(errors.Wrap(err, "failed to Setup transport"))
			}
		}()
	}

	return h
}

func (h *hub) discoveryHandler() func(endpoint string, uuid string) {
	return func(endpoint string, uuid string) {
		if uuid == h.nodeUUID {
			h.log.Debug("discovered self, discarding")
			return
		}

		// this reduces the number of extraneous outgoing handshakes that get attempted.
		if _, exists := h.findConnection(uuid); exists {
			h.log.Debug("encountered duplicate connection, discarding")
			return
		}

		if err := h.connectEndpoint(endpoint, uuid); err != nil {
			h.log.Error(errors.Wrap(err, "failed to connectEndpoint for discovered peer"))
		}
	}
}

// connectEndpoint creates a new outgoing connection
func (h *hub) connectEndpoint(endpoint, uuid string) error {
	if h.mesh == nil {
		return ErrTransportNotConfigured
	}

	h.log.Debug("connecting to endpoint", endpoint)

	conn, err := h.mesh.Connect(endpoint)
	if err != nil {
		return errors.Wrap(err, "failed to transport.CreateConnection")
	}

	h.setupOutgoingConnection(conn, uuid)

	return nil
}

// connectBridgeTopic creates a new outgoing connection
func (h *hub) connectBridgeTopic(topic string) error {
	if h.bridge == nil {
		return ErrTransportNotConfigured
	}

	h.log.Debug("connecting to topic", topic)

	conn, err := h.bridge.ConnectTopic(topic)
	if err != nil {
		return errors.Wrap(err, "failed to transport.CreateConnection")
	}

	h.addTopicConnection(conn, topic)

	return nil
}

func (h *hub) setupOutgoingConnection(connection Connection, uuid string) {
	handshake := &TransportHandshake{h.nodeUUID, h.belongsTo, h.interests}

	ack, err := connection.OutgoingHandshake(handshake)
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

	h.setupNewConnection(connection, uuid, ack.BelongsTo, ack.Interests)
}

func (h *hub) handleIncomingConnection(connection Connection) {
	var handshake *TransportHandshake
	var ack *TransportHandshakeAck

	callback := func(incomingHandshake *TransportHandshake) *TransportHandshakeAck {
		handshake = incomingHandshake

		ack = &TransportHandshakeAck{
			Accept: true,
			UUID:   h.nodeUUID,
		}

		if incomingHandshake.BelongsTo != h.belongsTo && incomingHandshake.BelongsTo != "*" {
			ack.Accept = false
		} else {
			ack.BelongsTo = h.belongsTo
			ack.Interests = h.interests
		}

		return ack
	}

	if err := connection.IncomingHandshake(callback); err != nil {
		h.log.Error(errors.Wrap(err, "failed to connection.DoIncomingHandshake"))
		connection.Close()
		return
	}

	if handshake == nil || handshake.UUID == "" {
		h.log.ErrorString("connection handshake returned empty UUID, terminating connection")
		connection.Close()
		return
	}

	if !ack.Accept {
		h.log.Debug("rejecting connection with incompatible BelongsTo", handshake.BelongsTo)
		connection.Close()
		return
	}

	h.setupNewConnection(connection, handshake.UUID, handshake.BelongsTo, handshake.Interests)
}

func (h *hub) setupNewConnection(connection Connection, uuid, belongsTo string, interests []string) {
	if _, exists := h.findConnection(uuid); exists {
		connection.Close()
		h.log.Debug("encountered duplicate connection, discarding")
	} else {
		h.addConnection(connection, uuid, belongsTo, interests)
	}
}

func (h *hub) incomingMessageHandler(uuid string) ReceiveFunc {
	return func(msg Message) {
		h.log.Debug("received message ", msg.UUID(), "from node", uuid)

		h.pod.Send(msg)
	}
}

func (h *hub) addConnection(connection Connection, uuid, belongsTo string, interests []string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.log.Debug("adding connection for", uuid)

	signaler := NewSignaler(h.context)

	handler := &connectionHandler{
		UUID:      uuid,
		Conn:      connection,
		Pod:       h.connectFunc(),
		Signaler:  signaler,
		ErrChan:   make(chan error),
		BelongsTo: belongsTo,
		Interests: interests,
		Log:       h.log,
	}

	handler.Start()

	h.meshConnections[uuid] = handler

	for _, c := range interests {
		if _, exists := h.capabilityUUIDBuffers[c]; !exists {
			h.capabilityUUIDBuffers[c] = NewMsgBuffer(capabilityBufferSize)
		}

		h.capabilityUUIDBuffers[c].Push(NewMsg(c, []byte(uuid)))
	}
}

func (h *hub) addTopicConnection(connection BridgeConnection, topic string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.log.Debug("adding bridge connection for", topic)

	connection.Start(h.connectFunc())

	h.bridgeConnections[topic] = connection
}

func (h *hub) removeMeshConnection(uuid string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.log.Debug("removing connection for", uuid)

	delete(h.meshConnections, uuid)
}

func (h *hub) findConnection(uuid string) (Connection, bool) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	conn, exists := h.meshConnections[uuid]
	if exists && conn.Conn != nil {
		return conn.Conn, true
	}

	return nil, false
}

// scanFailedMeshConnections should be run on a goroutine to constantly
// check for failed connections and clean them up
func (h *hub) scanFailedMeshConnections() {
	for {
		// we don't want to edit the `meshConnections` map while in the loop, so do it after
		toRemove := []string{}

		// for each connection, check if it has errored or if its peer has withdrawn,
		// and in either case close it and remove it from circulation
		for _, conn := range h.meshConnections {
			select {
			case <-conn.ErrChan:
				if err := conn.Close(); err != nil {
					h.log.Error(errors.Wrapf(err, "[grav] failed to Close %s", conn.UUID))
				}

				toRemove = append(toRemove, conn.UUID)
			default:
				if conn.Signaler.PeerWithdrawn.Load().(bool) {
					if err := conn.Close(); err != nil {
						h.log.Error(errors.Wrapf(err, "[grav] failed to Close %s", conn.UUID))
					}

					toRemove = append(toRemove, conn.UUID)
				}
			}
		}

		for _, uuid := range toRemove {
			h.removeMeshConnection(uuid)
		}

		time.Sleep(time.Second)
	}
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
		conn, exists := h.meshConnections[uuid]
		h.lock.RUnlock()

		if exists && conn.Conn != nil {
			if err := conn.Conn.SendMsg(msg); err != nil {
				h.log.Error(errors.Wrap(err, "[grav] failed to SendMsg on tunneled connection, will remove"))
				h.removeMeshConnection(uuid)
			} else {
				return nil
			}
		}
	}

	return ErrTunnelNotEstablished
}

func (h *hub) withdraw() error {
	if h.discovery != nil {
		h.discovery.Stop()
	}

	// calling cancelFunc will cancel h.context, which signals
	// all active connections to start the withdraw procedure
	h.cancelFunc()

	h.lock.Lock()
	defer h.lock.Unlock()

	// the withdraw attempt will time out after 5 seconds
	timeoutChan := time.After(time.Second * 5)
	doneChan := make(chan bool)

	go func() {
		count := len(h.meshConnections)

		// continually go through each connection and check if its withdraw is complete
		// until we've gotten the signal from every single one
		for {
			for uuid := range h.meshConnections {
				conn := h.meshConnections[uuid]

				select {
				case <-conn.Signaler.DoneChan:
					count--
				default:
					//continue
				}
			}

			if count == 0 {
				doneChan <- true
				break
			}
		}
	}()

	// return when either the withdraw is complete or we timed out
	select {
	case <-doneChan:
		//cool, done
	case <-timeoutChan:
		return ErrWaitTimeout
	}

	return nil
}

func (h *hub) stop() error {
	var lastErr error

	for _, c := range h.meshConnections {
		if err := c.Conn.Close(); err != nil {
			lastErr = err
			h.log.Error(errors.Wrapf(err, "[grav] failed to Close connection %s", c.UUID))
		}
	}

	return lastErr
}
