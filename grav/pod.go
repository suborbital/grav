package grav

import (
	"sync"
)

const (
	defaultPodChanSize = 256
)

// MsgFunc is a callback function that accepts a message
type MsgFunc func(Message)

// MsgChan is a channel that accepts a message
type MsgChan chan Message

/**
                              ┌─────────────────────┐
                              │                     │
            ──────BusEmit─────▶─────────────────────▶─────On────▶
┌────────┐                    │       		        │             ┌───────────────┐
│  Bus   │                    │        Pod          │             │   Pod Owner   │
└────────┘                    │       		        │             └───────────────┘
            ◀───BusListenFunc─◀─────────────────────◀────Emit────
                              │                     │
                              └─────────────────────┘

Created with Monodraw
**/

// Pod is a connection to the bus
// Pods are bi-directional. Messages can be sent
// through them to its owner, and messages can be emitted
// through them from its owner. Pods are meant to be
// extremely lightweight with no persistence
// they are meant to quickly and immediately route a message between
// its owner and the Bus. The Bus is responsible for all "smarts".
type Pod interface {
	On(MsgFunc)   // On registers a function to be called when an event is sent through this pod from the bus
	Emit(Message) // Emit emits an event to be handled by the bus. Routed to BusListenFunc.

	EmitChan() MsgChan // EmitChan returns the channel that should be written on by the bus when a new message is destined for this pod.
}

// memPod represents an in-memory Pod for in-process messages
type memPod struct {
	GroupName string

	onFunc MsgFunc

	messageChan MsgChan
	busChan     MsgChan

	listenOnce sync.Once
	lock       sync.Mutex
}

func newMemPod(group string, busChan MsgChan) *memPod {
	m := &memPod{
		GroupName:   group,
		messageChan: make(chan Message, 256),
		busChan:     busChan,
	}

	m.start()

	return m
}

// On allows the pod creator to recieve messages sent through this pod from the bus
// Setting the On func can only be done once.
func (m *memPod) On(onFunc MsgFunc) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.onFunc = onFunc
}

// Emit emits a message to be routed to the bus
func (m *memPod) Emit(msg Message) {
	go func() {
		m.busChan <- msg
	}()
}

// EmitChan takes in a message from the Bus and routes it to the OnFunc
func (m *memPod) EmitChan() MsgChan {
	return m.messageChan
}

func (m *memPod) start() {
	go func() {
		for msg := range m.messageChan {
			m.lock.Lock() // lock in case the onFunc gets replaced

			m.onFunc(msg)

			m.lock.Unlock()
		}
	}()
}
