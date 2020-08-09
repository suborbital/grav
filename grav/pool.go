package grav

import (
	"errors"
	"sync"
)

var errFailedMessage = errors.New("pod reports failed message")

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

// next returns the next connection in the ring
func (c *connectionPool) next() *podConnection {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.current = c.current.next

	return c.current
}

// insert inserts a new connection into the ring
func (c *connectionPool) insert(pod *Pod) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.maxID++
	id := c.maxID

	conn := newPodConnection(id, pod)

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
// forever as the bus emits events to the registered pods
type podConnection struct {
	ID          int64
	messageChan MsgChan
	errorChan   MsgChan

	failed []Message

	next *podConnection
}

func newPodConnection(id int64, pod *Pod) *podConnection {
	msgChan, errChan := pod.BusChans()

	p := &podConnection{
		ID:          id,
		messageChan: msgChan,
		errorChan:   errChan,
		failed:      []Message{},
		next:        nil,
	}

	return p
}

func (p *podConnection) emit(msg Message) {
	go func() {
		p.messageChan <- msg
	}()
}

// checkErr checks the pod's errChan for any failed messages and drains the channel into the failed Message buffer
func (p *podConnection) checkErr() error {
	var err error

	for {
		done := false

		select {
		case failedMsg := <-p.errorChan:
			p.failed = append(p.failed, failedMsg)
			err = errFailedMessage
		default:
			done = true
		}

		if done {
			break
		}
	}

	return err
}

// flushFailed takes all of the failed messages in the failed queue
// and pushes them out into the pod's connection
func (p *podConnection) flushFailed() {
	for i := range p.failed {
		failedMsg := p.failed[i]

		go func() {
			p.messageChan <- failedMsg
		}()
	}

	p.failed = []Message{}
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
