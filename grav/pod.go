package grav

import (
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
            ◀───BusChan------─◀─────────────────────◀────Emit────
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
type Pod struct {
	GroupName string

	onFunc MsgFunc // the onFunc is called whenever a message is recieved

	messageChan MsgChan // messageChan is used to recieve messages coming from the bus
	errorChan   MsgChan // errorChan is used to send failed messages back to the bus
	busChan     MsgChan // busChan is used to emit messages to the bus

	*messageFilter // the embedded messageFilter controls which messages reach the onFunc

	sync.Mutex
}

// newPod creates a new Pod
func newPod(group string, busChan MsgChan) *Pod {
	p := &Pod{
		GroupName:     group,
		messageChan:   make(chan Message, defaultPodChanSize),
		errorChan:     make(chan Message, defaultPodChanSize),
		busChan:       busChan,
		messageFilter: newMessageFilter(),
	}

	p.start()

	return p
}

// On allows the pod creator to recieve messages sent through this pod from the bus
// Setting the On func can only be done once.
func (p *Pod) On(onFunc MsgFunc) {
	p.Lock()
	defer p.Unlock()

	p.onFunc = onFunc
}

// Emit emits a message to be routed to the bus
func (p *Pod) Emit(msg Message) {
	go func() {
		p.FilterUUID(msg.UUID(), false) // don't allow the same message to bounce back through this pod

		p.busChan <- msg
	}()
}

// BusChans returns the messageChan and errorChan to be used by the bus
func (p *Pod) BusChans() (MsgChan, MsgChan) {
	return p.messageChan, p.errorChan
}

func (p *Pod) start() {
	go func() {
		// this loop ends when the bus closes the messageChan
		for msg := range p.messageChan {
			p.Lock() // in case the onFunc gets replaced

			// run the message through the filter before passing it to the onFunc
			if p.onFunc != nil && p.allow(msg) {
				if err := p.onFunc(msg); err != nil {
					go func() {
						// if the onFunc fails, send it back to the bus to be re-sent later
						p.errorChan <- msg
					}()
				}
			}

			p.Unlock()
		}
	}()
}
