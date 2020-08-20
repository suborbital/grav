package grav

import (
	"errors"
	"sync"
)

const (
	defaultPodChanSize = 64
)

/**
                              ┌─────────────────────┐
                              │                     │
            ──messageChan─────▶─────────────────────▶─────On────▶
┌────────┐                    │       		        │             ┌───────────────┐
│  Bus   │                    │        Pod          │             │   Pod Owner   │
└────────┘                    │       		        │             └───────────────┘
            ◀───BusChan------─◀─────────────────────◀────Send────
                              │                     │
                              └─────────────────────┘

Created with Monodraw
**/

// Pod is a connection to Grav
// Pods are bi-directional. Messages can be sent to them from the bus, and they can be used to send messages
// to the bus. Pods are meant to be extremely lightweight with no persistence they are meant to quickly
// and immediately route a message between its owner and the Bus. The Bus is responsible for any "smarts".
// Messages coming from the bus are filtered using the pod's messageFilter, which is configurable by the caller.
type Pod struct {
	onFunc MsgFunc // the onFunc is called whenever a message is recieved

	messageChan MsgChan // messageChan is used to recieve messages coming from the bus
	errorChan   MsgChan // errorChan is used to send failed messages back to the bus
	busChan     MsgChan // busChan is used to emit messages to the bus

	*messageFilter // the embedded messageFilter controls which messages reach the onFunc

	dead bool
	sync.RWMutex
}

// newPod creates a new Pod
func newPod(group string, busChan MsgChan) *Pod {
	p := &Pod{
		messageChan:   make(chan Message, defaultPodChanSize),
		errorChan:     make(chan Message, defaultPodChanSize),
		busChan:       busChan,
		messageFilter: newMessageFilter(),
		dead:          false,
		RWMutex:       sync.RWMutex{},
	}

	p.start()

	return p
}

// On sets the function to be called whenever this pod recieves a message from the bus. If nil is passed, the pod will ignore all messages.
// Calling On multiple times causes the function to be overwritten. To recieve using two different functions, create two pods.
func (p *Pod) On(onFunc MsgFunc) {
	p.Lock()
	defer p.Unlock()

	// reset the message filter when the onFunc is changed
	p.messageFilter = newMessageFilter()

	p.onFunc = onFunc
}

// OnType sets the function to be called whenever this pod recieves a message and sets the pod's filter to only include certain message types
func (p *Pod) OnType(onFunc MsgFunc, msgTypes ...string) {
	p.Lock()
	defer p.Unlock()

	// reset the message filter when the onFunc is changed
	p.messageFilter = newMessageFilter()
	p.TypeInclusive = false // only allow the listed types

	for _, t := range msgTypes {
		p.FilterType(t, true)
	}

	p.onFunc = onFunc
}

// ErrMsgNotWanted is used by WaitOn to determine if the current message is what's being waited on
var ErrMsgNotWanted = errors.New("message not wanted")

// WaitOn takes a function to be called whenever this pod recieves a message and blocks until that function returns
// something other than ErrMsgNotWanted. WaitOn should be used if there is a need to wait for a particular message.
// When the onFunc returns something other than ErrMsgNotWanted (such as nil or a different error), WaitOn will return and set
// the onFunc to nil. If an error other than ErrMsgNotWanted is returned from the onFunc, it will be propogated to the caller.
func (p *Pod) WaitOn(onFunc MsgFunc) error {
	p.Lock()
	errChan := make(chan error)

	p.onFunc = func(msg Message) error {
		if err := onFunc(msg); err != ErrMsgNotWanted {
			errChan <- err
		}

		return nil
	}
	p.Unlock() // can't stay locked here or the onFunc will never be called

	err := <-errChan

	p.Lock()
	defer p.Unlock()

	p.onFunc = nil

	return err
}

// Send emits a message to be routed to the bus
func (p *Pod) Send(msg Message) {
	p.RLock()
	defer p.RUnlock()

	if p.dead {
		return
	}

	go func() {
		p.FilterUUID(msg.UUID(), false) // don't allow the same message to bounce back through this pod

		p.busChan <- msg
	}()
}

// busChans returns the messageChan and errorChan to be used by the bus
func (p *Pod) busChans() (MsgChan, MsgChan) {
	return p.messageChan, p.errorChan
}

func (p *Pod) start() {
	go func() {
		// this loop ends when the bus closes the messageChan
		for msg := range p.messageChan {
			p.RLock() // in case the onFunc gets replaced

			// run the message through the filter before passing it to the onFunc
			if p.onFunc != nil && p.allow(msg) {
				err := p.onFunc(msg)
				go func() {
					if err != nil {
						// if the onFunc failed, send it back to the bus to be re-sent later
						p.errorChan <- msg
					} else {
						// if it was successful, a nil on the channel lets the conn know all is well
						p.errorChan <- nil
					}
				}()
			}

			p.RUnlock()
		}

		// if we've gotten here, the podConnection was closed and we should no longer do anything
		p.Lock()
		defer p.Unlock()

		p.dead = true
	}()
}
