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
		pod.Send(NewMsg(DefaultMsgType, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Duration(time.Second))

	if count != 100 {
		t.Errorf("incorrect number of messages, expected 100, got %d", count)
	}
}
