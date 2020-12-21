package grav

import (
	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

// TransportMsgTypeHandshake and others represent internal Transport message types used for handshakes and metadata transfer
const (
	TransportMsgTypeHandshake = 1
	TransportMsgTypeUser      = 2
)

// ErrConnectionClosed and others are transport and connection related errors
var (
	ErrConnectionClosed = errors.New("connection was closed")
	ErrNodeUUIDMismatch = errors.New("handshake UUID did not match node UUID")
)

// TransportOpts is a set of options for transports
type TransportOpts struct {
	NodeUUID string
	Port     string
	Logger   *vlog.Logger
	Custom   interface{}
}

// Transport represents a Grav transport plugin
type Transport interface {
	// Setup is a transport-specific function that allows bootstrapping
	Setup(opts *TransportOpts) error
	// CreateConnection connects to an endpoint and returns
	CreateConnection(endpoint string, uuid string) (Connection, error)
	// UseConnectionFunc gives the transport a function to use when incoming connections are established
	UseConnectionFunc(connFunc func(Connection))
}

// Connection represents a connection to another node
type Connection interface {
	// Send a message from the local instance to the connected node
	Send(msg Message) error
	// Give the connection a function to send incoming messages to the local instance
	UseReceiveFunc(recvFunc func(nodeUUID string, msg Message))
	// Initiate a handshake for an outgoing connection and return the remote Ack
	DoOutgoingHandshake(handshake *TransportHandshake) (*TransportHandshakeAck, error)
	// Wait for an incoming handshake and return the provided Ack to the remote connection
	DoIncomingHandshake(handshakeAck *TransportHandshakeAck) (*TransportHandshake, error)
	// Close requests that the Connection close itself
	Close()
}

// TransportHandshake represents a handshake sent to a node that you're trying to connect to
type TransportHandshake struct {
	UUID string `json:"uuid"`
}

// TransportHandshakeAck represents a handshake response
type TransportHandshakeAck struct {
	UUID string `json:"uuid"`
}
