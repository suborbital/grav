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
	UUID() string
	RequestID() string
	Type() string
	Timestamp() time.Time
	Data() []byte
	Marshal() []byte
	Unmarshal([]byte) error
}

// New creates a new Message with the built-in `_message` type
func New(msgType, requestID string, data []byte) Message {
	uuid := uuid.New()

	m := &_message{
		Meta: _meta{
			UUID:      uuid.String(),
			RequestID: requestID,
			MsgType:   msgType,
			Timestamp: time.Now(),
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
	UUID      string    `json:"uuid"`
	RequestID string    `json:"request_id"`
	MsgType   string    `json:"msg_type"`
	Timestamp time.Time `json:"timestamp"`
}

type _payload struct {
	Data []byte `json:"data"`
}

func (m *_message) UUID() string {
	return m.Meta.UUID
}

func (m *_message) RequestID() string {
	return m.Meta.RequestID
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
