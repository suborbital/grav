package grav

import (
	"fmt"
	"testing"

	"github.com/suborbital/grav/testutil"
)

func TestRequestReply(t *testing.T) {
	g := New()
	p1 := g.Connect()

	msg := NewMsg(MsgTypeDefault, []byte("joey"))
	p1.Send(msg)

	p2 := g.ConnectWithReplay()
	p2.On(func(msg Message) error {
		data := string(msg.Data())

		reply := NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hey %s", data)))
		p2.ReplyTo(msg.Ticket(), reply)

		return nil
	})

	counter := testutil.NewAsyncCounter(10)

	go func() {
		p1.WaitOnReply(msg.Ticket(), nil, func(msg Message) error {
			if string(msg.Data()) == "hey joey" {
				counter.Count()
			}

			return nil
		})
	}()

	counter.Wait(1, 1)
}

func TestRequestReplySugar(t *testing.T) {
	g := New()
	p1 := g.Connect()

	counter := testutil.NewAsyncCounter(10)

	go func() {
		msg := NewMsg(MsgTypeDefault, []byte("joey"))
		p1.SendAndWaitOnReply(msg, nil, func(msg Message) error {
			counter.Count()
			return nil
		})
	}()

	p2 := g.ConnectWithReplay()
	p2.On(func(msg Message) error {
		data := string(msg.Data())

		reply := NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hey %s", data)))
		p2.ReplyTo(msg.Ticket(), reply)

		return nil
	})

	counter.Wait(1, 1)
}

func TestRequestReplyTimeout(t *testing.T) {
	g := New()
	p1 := g.Connect()

	counter := testutil.NewAsyncCounter(10)

	go func() {
		msg := NewMsg(MsgTypeDefault, []byte("joey"))
		if err := p1.SendAndWaitOnReply(msg, TO(1), func(msg Message) error {
			return nil
		}); err == ErrWaitTimeout {
			counter.Count()
		}
	}()

	counter.Wait(1, 2)
}
