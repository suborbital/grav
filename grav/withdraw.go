package grav

import (
	"context"
	"sync/atomic"
)

// WithdrawSignaler allows a connection to be notified about a withdraw event and report
// back to the hub that a withdraw has completed (using the Ctx and DoneChan, respectively)
type WithdrawSignaler struct {
	Ctx           context.Context
	DoneChan      chan bool
	PeerWithdrawn atomic.Value
}

// NewSignaler creates a new WithdrawSignaler based on the provided context
func NewSignaler(ctx context.Context) *WithdrawSignaler {
	w := &WithdrawSignaler{
		Ctx:           ctx,
		DoneChan:      make(chan bool, 1),
		PeerWithdrawn: atomic.Value{},
	}

	w.PeerWithdrawn.Store(false)

	return w
}
