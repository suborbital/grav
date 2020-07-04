package grav

import "sync"

type connectionPool struct {
	current *podConnection

	lock sync.Mutex
}

// next returns the next connection in te ring
func (c *connectionPool) next() *podConnection {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.current = c.current.next

	return c.current
}

// insert inserts a new connection into the ring
func (c *connectionPool) insert(conn *podConnection) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.current == nil {
		c.current = conn
	} else {
		c.current.insertAfter(conn)
	}
}

// podConnection is a connection to a pod via its emitChan
// podConnection is also a circular linked list/ring of connections
// that is meant to be iterated around and inserted into/removed from
// forever as the bus emits events to the registered pods
type podConnection struct {
	ID       int64
	emitChan MsgChan

	next *podConnection
}

func newPodConnection(id int64, emitChan MsgChan) *podConnection {
	p := &podConnection{
		ID:       id,
		emitChan: emitChan,
		next:     nil,
	}

	return p
}

func (p *podConnection) emit(msg Message) {
	go func() {
		p.emitChan <- msg
	}()
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
