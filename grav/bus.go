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
		// continually take new messages and for each,
		// grab the next active connection from the ring and then
		// start traversing around the ring to emit the message to
		// each connection until landing back at the beginning of the
		// ring, and repeat forever when each new message arrives
		for msg := range b.busChan {
			for {
				// make sure the pod we start with is healthy
				if err := b.pool.checkNextPod(); err == nil {
					break
				}
			}

			startingConn := b.pool.next()

			b.traverse(msg, startingConn)
		}
	}()
}

func (b *messageBus) traverse(msg Message, start *podConnection) {
	startID := start.ID
	conn := start

	for {
		// send the message to the pod
		conn.send(msg)

		next := b.pool.peek()
		if err := b.pool.checkNextPod(); err != nil {
			if startID == next.ID {
				startID = next.next.ID
			}
		}

		// now advance the ring
		conn = b.pool.next()

		if startID == conn.ID {
			// if we have arrived back at the starting point on the ring
			// we have done our job and are ready for the next message
			break
		}
	}
}
