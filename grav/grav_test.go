package grav

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestGravSingle(t *testing.T) {
	g := New()

	count := 0

	p1 := g.Connect()

	p1.On(func(msg Message) error {
		count++

		return nil
	})

	p2 := g.Connect()
	p2.Send(NewMsg(MsgTypeDefault, []byte("hello, world")))

	time.Sleep(time.Duration(time.Second))

	if count != 1 {
		t.Errorf("incorrect number of messages, expected 1, got %d", count)
	}
}

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
		pod.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Duration(time.Second))

	if count != 100 {
		t.Errorf("incorrect number of messages, expected 100, got %d", count)
	}
}

func TestGravBench(t *testing.T) {
	g := New()

	doneChan := make(chan bool)
	lock := sync.Mutex{}
	count := 0

	for i := 0; i < 100000; i++ {
		p := g.Connect()

		p.On(func(msg Message) error {
			lock.Lock()
			defer lock.Unlock()

			count++

			if count == 1000000 {
				doneChan <- true
			} else if count > 1000000 {
				t.Error("recieved more than 100 messages!")
			}

			return nil
		})
	}

	pod := g.Connect()

	start := time.Now()
	for i := 0; i < 10; i++ {
		pod.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	<-doneChan
	duration := time.Since(start).Seconds()

	fmt.Println(duration, "seconds")
	if duration > 2 {
		t.Error("delivering 1M messages took too long!")
	}

	time.Sleep(time.Duration(time.Second)) // let any errors filter in from above
}
