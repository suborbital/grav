package grav

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestPodFilter(t *testing.T) {
	g := New()

	lock := sync.Mutex{}
	count := 0

	onFunc := func(msg Message) error {
		lock.Lock()
		defer lock.Unlock()

		count++

		return nil
	}

	p1 := g.Connect()
	p1.On(onFunc)

	p2 := g.Connect()
	p2.On(onFunc)

	for i := 0; i < 10; i++ {
		p1.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Duration(time.Second))

	// only 10 should be tracked because p1 should have filtered out the messages that it sent
	// and then they should not reach its own onFunc
	if count != 10 {
		t.Errorf("incorrect number of messages, expected 10, got %d", count)
	}
}

func TestWaitOn(t *testing.T) {
	g := New()

	p1 := g.Connect()

	go func() {
		time.Sleep(time.Duration(time.Millisecond * 500))
		p1.Send(NewMsg(MsgTypeDefault, []byte("hello, world")))
		time.Sleep(time.Duration(time.Millisecond * 500))
		p1.Send(NewMsg(MsgTypeDefault, []byte("goodbye, world")))
	}()

	errGoodbye := errors.New("goodbye")

	p2 := g.Connect()

	if err := p2.WaitOn(func(msg Message) error {
		if bytes.Equal(msg.Data(), []byte("hello, world")) {
			return nil
		}

		return ErrMsgNotWanted
	}); err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if err := p2.WaitOn(func(msg Message) error {
		if bytes.Equal(msg.Data(), []byte("goodbye, world")) {
			return errGoodbye
		}

		return ErrMsgNotWanted
	}); err != errGoodbye {
		t.Errorf("expected errGoodbye error, got %s", err)
	}
}

const msgTypeBad = "test.bad"

func TestPodFailure(t *testing.T) {
	g := New()

	counter := make(chan bool, 1000)

	// create one pod that returns errors on "bad" messages
	p := g.Connect()
	p.On(func(msg Message) error {
		counter <- true

		if msg.Type() == msgTypeBad {
			return errors.New("bad message")
		}

		return nil
	})

	// and another 9 that don't
	for i := 0; i < 9; i++ {
		p2 := g.Connect()

		p2.On(func(msg Message) error {
			counter <- true

			return nil
		})
	}

	pod := g.Connect()

	// send 64 "bad" messages (64 reaches the highwater mark)
	for i := 0; i < 64; i++ {
		pod.Send(NewMsg(msgTypeBad, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Duration(time.Second))

	// send 10 more "bad" messages
	for i := 0; i < 10; i++ {
		pod.Send(NewMsg(msgTypeBad, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Duration(time.Second))

	count := 0
	more := true
	for more {
		select {
		case <-counter:
			count++
		default:
			more = false
		}
	}

	// 730 because the 64th message to the "bad" pod put it over the highwater
	// mark and so the last 10 message would never be delievered
	// sleeps were needed to allow all of the internal goroutines to finish execing
	// in the worst case scenario of a single process machine (which lots of containers are)
	if count != 730 {
		t.Errorf("incorrect number of messages, expected 730, got %d", count)
	}

	// the first pod should now have been disconnected, causing only 9 recievers reset and test again

	// send 10 "normal" messages
	for i := 0; i < 10; i++ {
		pod.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Duration(time.Second))

	count = 0
	more = true
	for more {
		select {
		case <-counter:
			count++
		default:
			more = false
		}
	}

	if count != 90 {
		t.Errorf("incorrect number of messages, expected 90, got %d", count)
	}
}

func TestPodFlushFailed(t *testing.T) {
	g := New()

	counter := make(chan bool, 100)

	// create a pod that returns errors on "bad" messages
	p := g.Connect()
	p.On(func(msg Message) error {
		counter <- true

		if msg.Type() == msgTypeBad {
			return errors.New("bad message")
		}

		return nil
	})

	pod := g.Connect()

	// send 5 "bad" messages
	for i := 0; i < 5; i++ {
		pod.Send(NewMsg(msgTypeBad, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Duration(time.Second))

	// replace the OnFunc to not error when the flushed messages come back through
	p.On(func(msg Message) error {
		counter <- true

		return nil
	})

	// send 10 "normal" messages
	for i := 0; i < 10; i++ {
		pod.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Duration(time.Second))

	count := 0
	more := true
	for more {
		select {
		case <-counter:
			count++
		default:
			more = false
		}
	}

	// 20 because upon handling the first "good" message, the bus should flush
	// the 5 "failed" messages back into the connection thus repeating them
	if count != 20 {
		t.Errorf("incorrect number of messages, expected 20, got %d", count)
	}
}
