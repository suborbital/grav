package bus

import (
	"github.com/suborbital/grav/grav/message"
	"github.com/suborbital/grav/grav/pod"
)

// Bus is responsible for emitting events among the connected pods
// and managing the failure cases for those pods
type Bus struct {
	busChan message.MsgChan
	pool    *connectionPool
}

// New creates a new bus
func New() *Bus {
	b := &Bus{
		busChan: make(chan message.Message, 256),
		pool:    newConnectionPool(),
	}

	b.start()

	return b
}

// BusChan returns the recieve chan
func (b *Bus) BusChan() message.MsgChan {
	return b.busChan
}

// AddPod adds a pod to the connection pool
func (b *Bus) AddPod(pod *pod.Pod) {
	b.pool.insert(pod)
}

func (b *Bus) start() {
	go func() {
		// continually take new messages and for each message,
		// grab the next active connection from the ring and then
		// start traversing around the ring to emit the message to
		// each connection until landing back at the beginning of the
		// ring, and repeat forever when each new message arrives
		for msg := range b.busChan {
			startingConn := b.pool.next()

			b.traverse(msg, startingConn)
		}
	}()
}

func (b *Bus) traverse(msg message.Message, start *podConnection) {
	conn := start
	for {
		if err := conn.checkErr(); err != nil {
			// may want to do something here in the future
			// such as mark the connection as unhealthy
			// currently it just buffers the failed message
			// into the connction's "failed" queue
		} else {
			// if the most recent error check comes back clean,
			// then tell the connection to flush any failed messages
			// this is a no-op if there are no failed messages queued
			conn.flushFailed()
		}

		// send the message to the pod
		conn.emit(msg)

		next := b.pool.next()
		if next.ID == start.ID {
			// if we have arrived back at the starting point
			// on the ring, we have done our job and are ready
			// for the next message
			break
		}

		conn = next
	}
}
