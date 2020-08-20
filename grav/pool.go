package grav

import (
	"errors"
	"sync"
)

const (
	highWaterMark = 64
)

var (
	errFailedMessage    = errors.New("pod reports failed message")
	errFailedMessageMax = errors.New("pod reports max number of failed messages, will terminate connection")
)

// connectionPool is a ring of connections to pods
// which will be iterated over constantly in order to send
// incoming messages to them
type connectionPool struct {
	current *podConnection

	maxID int64
	lock  sync.Mutex
}

func newConnectionPool() *connectionPool {
	p := &connectionPool{
		current: nil,
		maxID:   0,
		lock:    sync.Mutex{},
	}

	return p
}

// peek returns a peek at the next connection in the ring wihout advancing the ring's current location
func (c *connectionPool) peek() *podConnection {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.current.next
}

// next returns the next connection in the ring
func (c *connectionPool) next() *podConnection {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.current = c.current.next

	return c.current
}

// deleteNext deletes the next connection in the ring
// this is useful after having checkError'd the next conn
// and seeing that it's unhealthy
func (c *connectionPool) deleteNext() {
	c.lock.Lock()
	defer c.lock.Unlock()

	next := c.current.next

	// lock the deadlock so any straggling send()s block forever
	next.deadLock.Lock()

	// close the messageChan so the pod can know it's been cut off
	close(next.messageChan)

	if next == c.current {
		// if there's only one thing in the ring, empty the ring
		c.current = nil
	} else {
		// cut out `next` and link `current` to `next-next`
		c.current.next = next.next
	}
}

// insert inserts a new connection into the ring
func (c *connectionPool) insert(pod *Pod) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.maxID++
	id := c.maxID

	conn := newPodConnection(id, pod)

	// if there's nothing in the ring, create a "ring of one"
	if c.current == nil {
		conn.next = conn
		c.current = conn
	} else {
		c.current.insertAfter(conn)
	}
}

// podConnection is a connection to a pod via its messageChan
// podConnection is also a circular linked list/ring of connections
// that is meant to be iterated around and inserted into/removed from
// forever as the bus sends events to the registered pods
type podConnection struct {
	ID          int64
	messageChan MsgChan
	errorChan   MsgChan

	failed []Message

	deadLock sync.RWMutex

	next *podConnection
}

func newPodConnection(id int64, pod *Pod) *podConnection {
	msgChan, errChan := pod.busChans()

	p := &podConnection{
		ID:          id,
		messageChan: msgChan,
		errorChan:   errChan,
		failed:      []Message{},
		deadLock:    sync.RWMutex{},
		next:        nil,
	}

	return p
}

func (p *podConnection) send(msg Message) {
	go func() {
		// get read-lock to ensure we're not dead
		p.deadLock.RLock()
		defer p.deadLock.RUnlock()

		p.messageChan <- msg
	}()
}

// checkErr checks the pod's errChan for any failed messages and drains the channel into the failed Message buffer
func (p *podConnection) checkErr() error {
	var err error

	done := false
	for !done {
		select {
		case failedMsg := <-p.errorChan:
			if failedMsg != nil {
				p.failed = append(p.failed, failedMsg)
				err = errFailedMessage
			} else {
				done = true
			}
		default:
			// if there's no nil on the channel, then we don't know if there's any new successes
			if len(p.failed) > 0 {
				err = errFailedMessage
			}

			done = true
		}
	}

	if len(p.failed) >= highWaterMark {
		err = errFailedMessageMax
	}

	return err
}

// flushFailed takes all of the failed messages in the failed queue
// and pushes them back out onto the pod's channel
func (p *podConnection) flushFailed() {
	for i := range p.failed {
		failedMsg := p.failed[i]

		p.send(failedMsg)
	}

	if len(p.failed) > 0 {
		p.failed = []Message{}
	}
}

// insertAfter inserts a new connection into the ring
func (p *podConnection) insertAfter(conn *podConnection) {
	next := p
	if p.next != nil {
		next = p.next
	}

	p.next = conn
	conn.next = next
}
