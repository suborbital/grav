package grav

import (
	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

type connectionHandler struct {
	UUID      string
	Conn      Connection
	Pod       *Pod
	Signaler  *WithdrawSignaler
	ErrChan   chan error
	BelongsTo string
	Interests []string
	Log       *vlog.Logger
}

// Start starts up a listener to read messages from the connection into the Grav bus
func (c *connectionHandler) Start() {
	c.Pod.On(func(msg Message) error {
		if c.Signaler.PeerWithdrawn.Load().(bool) {
			return nil
		}

		if err := c.Conn.SendMsg(msg); err != nil {
			c.Log.Error(errors.Wrapf(err, "[grav] failed to SendMsg to connection %s", c.UUID))
			c.ErrChan <- err
		}

		return nil
	})

	withdrawSignal := c.Signaler.Ctx.Done()

	go func() {
		for {
			select {
			case <-withdrawSignal:
				c.Log.Debug("sending withdraw and disconnecting")

				if err := c.Conn.SendWithdraw(&Withdraw{}); err != nil {
					c.Log.Error(errors.Wrapf(err, "[grav] failed to SendWithdraw to connection %s", c.UUID))
					c.ErrChan <- err
				}

				c.Signaler.DoneChan <- true
				return
			default:
				// continue as normal
			}

			msg, withdraw, err := c.Conn.ReadMsg()
			if err != nil {
				c.Log.Error(errors.Wrapf(err, "[grav] failed to SendMsg to connection %s", c.UUID))
				c.ErrChan <- err
				return
			}

			if withdraw != nil {
				c.Log.Debug("peer has withdrawn, disconnecting")

				c.Pod.Disconnect()
				c.Signaler.PeerWithdrawn.Store(true)
				return
			}

			c.Pod.Send(msg)
		}
	}()
}

// Close stops outgoing messages and closes the underlying connection
func (c *connectionHandler) Close() error {
	c.Pod.Disconnect()

	if err := c.Conn.Close(); err != nil {
		return errors.Wrap(err, "failed to Conn.Close")
	}

	return nil
}
