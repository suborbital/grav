package grav

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestGravSanity(t *testing.T) {
	g := New()

	lock := sync.Mutex{}
	count := 0

	for i := 0; i < 10; i++ {
		p := g.Connect()

		p.On(func(msg Message) error {
			lock.Lock()
			defer lock.Unlock()

			count++

			return nil
		})
	}

	pod := g.Connect()

	for i := 0; i < 10; i++ {
		pod.Emit(NewMsg(DefaultMsgType, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Duration(time.Second))

	if count != 100 {
		t.Errorf("incorrect number of messages, expected 100, got %d", count)
	}
}

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
		p1.Emit(NewMsg(DefaultMsgType, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Duration(time.Second))

	// only 10 should be tracked because p1 should have filtered out the messages that it sent
	// and then they should not reach its own onFunc
	if count != 10 {
		t.Errorf("incorrect number of messages, expected 10, got %d", count)
	}
}
