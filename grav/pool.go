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

// prepareNext ensures that the next pod connection in the ring is ready to recieve
// new messages by checking its status, deleting it if unhealthy, replaying the message
// buffer if needed, or flushing failed messages back onto its channel if needeed.
func (c *connectionPool) prepareNext(buffer *msgBuffer) error {
	// peek gives us the next conn without advancing the ring
	// this makes it easy to delete the next conn if it's unhealthy
	next := c.peek()

	// check the state of the next connection
	status := next.checkStatus()

	if status.Error != nil {
		// if the connection has an issue, handle it
		if status.Error == errFailedMessageMax {
			c.deleteNext()
			return errors.New("deleting next podConnection")
		}
	} else {
		if status.WantsReplay {
			// if the pod has indicated that it wants a replay of recent messages, do so
			c.replayNext(buffer)
		} else {
			// if the most recent error check comes back clean,
			// then tell the connection to flush any failed messages
			// this is a no-op if there are no failed messages queued
			next.flushFailed()
		}
	}

	return nil
}

// replayNext replays the current message buffer into the next connection
func (c *connectionPool) replayNext(buffer *msgBuffer) {
	next := c.peek()

	// iterate over the buffer and send each message to the pod
	buffer.Iter(func(msg Message) error {
		next.send(msg)

		return nil
	})
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

// podConnection is a connection to a pod via its messageChan
// podConnection is also a circular linked list/ring of connections
// that is meant to be iterated around and inserted into/removed from
// forever as the bus sends events to the registered pods
type podConnection struct {
	ID           int64
	messageChan  MsgChan
	feedbackChan MsgChan

	failed []Message

	deadLock sync.RWMutex

	next *podConnection
}

// connStatus is used to communicate the status of a podConnection back to the bus
type connStatus struct {
	Error       error
	WantsReplay bool
}

func newPodConnection(id int64, pod *Pod) *podConnection {
	msgChan, feedbackChan := pod.busChans()

	p := &podConnection{
		ID:           id,
		messageChan:  msgChan,
		feedbackChan: feedbackChan,
		failed:       []Message{},
		deadLock:     sync.RWMutex{},
		next:         nil,
	}

	return p
}

// send asynchronously writes a message to a connection's messageChan
// ordering to the messageChan if it becomes full is not guaranteed, this
// is sacrificed to ensure that the bus does not block because of a delinquient pod
func (p *podConnection) send(msg Message) {
	go func() {
		// get read-lock to ensure we're not dead
		p.deadLock.RLock()
		defer p.deadLock.RUnlock()

		p.messageChan <- msg
	}()
}

// checkStatus checks the pod's feedback for any information or failed messages and drains the failures into the failed Message buffer
func (p *podConnection) checkStatus() *connStatus {
	status := &connStatus{
		Error:       nil,
		WantsReplay: false,
	}

	done := false
	for !done {
		select {
		case feedbackMsg := <-p.feedbackChan:
			if feedbackMsg != nil {
				if feedbackMsg == podFeedbackMsgReplay {
					status.WantsReplay = true
				} else {
					p.failed = append(p.failed, feedbackMsg)
					status.Error = errFailedMessage
				}
			} else {
				done = true
			}
		default:
			// if there's no nil on the channel, then we don't know if there's any new successes
			if len(p.failed) > 0 {
				status.Error = errFailedMessage
			}

			done = true
		}
	}

	if len(p.failed) >= highWaterMark {
		status.Error = errFailedMessageMax
	}

	return status
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
