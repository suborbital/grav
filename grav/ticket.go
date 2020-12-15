package grav

import "github.com/pkg/errors"

// ErrNoTicket is returned when a method is called on a nil ticket
var ErrNoTicket = errors.New("message ticket is nil")

// MessageTicket represents a "ticket" that references a message that was sent with the hopes of getting a response
// The embedded pod is a pointer to the pod that sent the original message, and therefore any ticket methods used
// will replace the OnFunc of the pod.
type MessageTicket struct {
	UUID string
	pod  *Pod
}

// Wait will block until a response to the message is recieved
func (m *MessageTicket) Wait(mfn MsgFunc) error {
	if m == nil {
		return ErrNoTicket
	}

	return m.pod.waitOnReply(m, nil, mfn)
}

// WaitUntil will block until a response to the message is recieved or the timeout elapses
func (m *MessageTicket) WaitUntil(timeout TimeoutFunc, mfn MsgFunc) error {
	if m == nil {
		return ErrNoTicket
	}

	return m.pod.waitOnReply(m, timeout, mfn)
}

// Then will set the pod's OnFunc to the provided MsgFunc and set it to run when a reply is received
func (m *MessageTicket) Then(mfn MsgFunc) error {
	if m == nil {
		return ErrNoTicket
	}

	go func() {
		m.pod.waitOnReply(m, nil, mfn)
	}()

	return nil
}
