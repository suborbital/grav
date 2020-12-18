package grav

import "github.com/pkg/errors"

// ErrNoReceipt is returned when a method is called on a nil ticket
var ErrNoReceipt = errors.New("message receipt is nil")

// MsgReceipt represents a "ticket" that references a message that was sent with the hopes of getting a response
// The embedded pod is a pointer to the pod that sent the original message, and therefore any ticket methods used
// will replace the OnFunc of the pod.
type MsgReceipt struct {
	UUID string
	pod  *Pod
}

// Wait will block until a response to the message is recieved
func (m *MsgReceipt) Wait(mfn MsgFunc) error {
	if m == nil {
		return ErrNoReceipt
	}

	return m.pod.waitOnReply(m, nil, mfn)
}

// WaitUntil will block until a response to the message is recieved or the timeout elapses
func (m *MsgReceipt) WaitUntil(timeout TimeoutFunc, mfn MsgFunc) error {
	if m == nil {
		return ErrNoReceipt
	}

	return m.pod.waitOnReply(m, timeout, mfn)
}

// Then will set the pod's OnFunc to the provided MsgFunc and set it to run when a reply is received
func (m *MsgReceipt) Then(mfn MsgFunc) error {
	if m == nil {
		return ErrNoReceipt
	}

	go func() {
		m.pod.waitOnReply(m, nil, mfn)
	}()

	return nil
}
