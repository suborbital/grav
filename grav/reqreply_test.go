package grav

import (
	"fmt"
	"testing"

	"github.com/suborbital/grav/testutil"
)

func TestRequestReply(t *testing.T) {
	g := New()
	p1 := g.Connect()

	ticket := p1.Send(NewMsg(MsgTypeDefault, []byte("joey")))

	p2 := g.ConnectWithReplay()
	p2.On(func(msg Message) error {
		data := string(msg.Data())

		p2.ReplyTo(msg, NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hey %s", data))))

		return nil
	})

	counter := testutil.NewAsyncCounter(10)

	go func() {
		ticket.Wait(func(msg Message) error {
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
		p1.Send(NewMsg(MsgTypeDefault, []byte("joey"))).Wait(func(msg Message) error {
			counter.Count()
			return nil
		})
	}()

	p2 := g.ConnectWithReplay()
	p2.On(func(msg Message) error {
		data := string(msg.Data())

		reply := NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hey %s", data)))
		p2.ReplyTo(msg, reply)

		return nil
	})

	counter.Wait(1, 1)
}

func TestRequestReplyTimeout(t *testing.T) {
	g := New()
	p1 := g.Connect()

	counter := testutil.NewAsyncCounter(10)

	go func() {
		if err := p1.Send(NewMsg(MsgTypeDefault, []byte("joey"))).WaitUntil(TO(1), func(msg Message) error {
			return nil
		}); err == ErrWaitTimeout {
			counter.Count()
		}
	}()

	counter.Wait(1, 2)
}
