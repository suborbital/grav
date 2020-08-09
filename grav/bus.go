package grav

// messageBus is responsible for emitting events among the connected pods
// and managing the failure cases for those pods
type messageBus struct {
	busChan MsgChan
	pool    *connectionPool
}

// newMessageBus creates a new messageBus
func newMessageBus() *messageBus {
	b := &messageBus{
		busChan: make(chan Message, 256),
		pool:    newConnectionPool(),
	}

	b.start()

	return b
}

// addPod adds a pod to the connection pool
func (b *messageBus) addPod(pod *Pod) {
	b.pool.insert(pod)
}

func (b *messageBus) start() {
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

func (b *messageBus) traverse(msg Message, start *podConnection) {
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
