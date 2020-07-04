package pod

import (
	"sync"

	"github.com/suborbital/grav/grav/message"
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
// Pods are bi-directional. message.Messages can be sent
// through them to its owner, and messages can be emitted
// through them from its owner. Pods are meant to be
// extremely lightweight with no persistence
// they are meant to quickly and immediately route a message between
// its owner and the Bus. The Bus is responsible for all "smarts".
type Pod struct {
	GroupName string

	onFunc message.MsgFunc // the onFunc is called whenever a message is recieved

	messageChan message.MsgChan // messageChan is used to recieve messages coming from the bus
	errorChan   message.MsgChan // errorChan is used to send failed messages back to the bus

	busChan message.MsgChan // busChan is used to emit messages to the bus

	lock sync.Mutex
}

// New creates a new Pod
func New(group string, busChan message.MsgChan) *Pod {
	p := &Pod{
		GroupName:   group,
		messageChan: make(chan message.Message, defaultPodChanSize),
		errorChan:   make(chan message.Message, defaultPodChanSize),
		busChan:     busChan,
	}

	p.start()

	return p
}

// On allows the pod creator to recieve messages sent through this pod from the bus
// Setting the On func can only be done once.
func (p *Pod) On(onFunc message.MsgFunc) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.onFunc = onFunc
}

// Emit emits a message to be routed to the bus
func (p *Pod) Emit(msg message.Message) {
	go func() {
		p.busChan <- msg
	}()
}

// BusChans returns the messageChan and errorChan to be used by the bus
func (p *Pod) BusChans() (message.MsgChan, message.MsgChan) {
	return p.messageChan, p.errorChan
}

func (p *Pod) start() {
	go func() {
		// this loop ends when the bus closes the messageChan
		for msg := range p.messageChan {
			p.lock.Lock() // lock in case the onFunc gets replaced

			if p.onFunc != nil {
				if err := p.onFunc(msg); err != nil {
					go func() {
						// if the onFunc fails, send it back to the bus to be re-sent later
						p.errorChan <- msg
					}()
				}
			}

			p.lock.Unlock()
		}
	}()
}
