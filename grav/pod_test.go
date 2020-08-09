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
		p1.Send(NewMsg(DefaultMsgType, []byte(fmt.Sprintf("hello, world %d", i))))
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
		p1.Send(NewMsg(DefaultMsgType, []byte("hello, world")))
		time.Sleep(time.Duration(time.Millisecond * 500))
		p1.Send(NewMsg(DefaultMsgType, []byte("goodbye, world")))
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
