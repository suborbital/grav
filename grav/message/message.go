package message

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MsgFunc is a callback function that accepts a message and returns an error
type MsgFunc func(Message) error

// MsgChan is a channel that accepts a message
type MsgChan chan Message

// Message represents a message
type Message interface {
	// Unique ID for this message
	UUID() string
	// ID of the parent event or request, such as HTTP request
	ParentID() string
	// The UUID of the message being responded to, if any
	ResponseTo() string
	// Type of message (application-specific)
	Type() string
	// Time the message was emitted
	Timestamp() time.Time
	// Raw data of message
	Data() []byte
	// Encoded Message object
	Marshal() []byte
	// Unmarshal encoded Message into object
	Unmarshal([]byte) error
}

// New creates a new Message with the built-in `_message` type
func New(msgType string, data []byte) Message {
	return new(msgType, "", "", data)
}

// NewWithParentID returns a new message with the provided parent ID
func NewWithParentID(msgType, parentID string, data []byte) Message {
	return new(msgType, parentID, "", data)
}

// NewResponseTo creates a new message in response to a previous message
func NewResponseTo(msgType, responseTo string, data []byte) Message {
	return new(msgType, "", responseTo, data)
}

func new(msgType, parentID, responseTo string, data []byte) Message {
	uuid := uuid.New()

	m := &_message{
		Meta: _meta{
			UUID:       uuid.String(),
			ParentID:   parentID,
			ResponseTo: responseTo,
			MsgType:    msgType,
			Timestamp:  time.Now(),
		},
		Payload: _payload{
			Data: data,
		},
	}

	return m
}

// _message is a basic built-in implementation of Message
// most applications should define their own data structure
// that implements the interface
type _message struct {
	Meta    _meta    `json:"meta"`
	Payload _payload `json:"payload"`
}

type _meta struct {
	UUID       string    `json:"uuid"`
	ParentID   string    `json:"parent_id"`
	ResponseTo string    `json:"response_to"`
	MsgType    string    `json:"msg_type"`
	Timestamp  time.Time `json:"timestamp"`
}

type _payload struct {
	Data []byte `json:"data"`
}

func (m *_message) UUID() string {
	return m.Meta.UUID
}

func (m *_message) ParentID() string {
	return m.Meta.ParentID
}

func (m *_message) ResponseTo() string {
	return m.Meta.ResponseTo
}

func (m *_message) Type() string {
	return m.Meta.MsgType
}

func (m *_message) Timestamp() time.Time {
	return m.Meta.Timestamp
}

func (m *_message) Data() []byte {
	return m.Payload.Data
}

func (m *_message) Marshal() []byte {
	bytes, _ := json.Marshal(m)

	return bytes
}

func (m *_message) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, m)
}
