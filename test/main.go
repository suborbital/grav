package main

import (
	"fmt"
	"time"

	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/grav/message"
)

func main() {
	g := grav.New()

	for i := 0; i < 10; i++ {
		p := g.Connect()
		j := i

		p.On(func(msg message.Message) error {
			fmt.Println(fmt.Sprintf("pod %d", j), string(msg.Data()))
			return nil
		})
	}

	pod := g.Connect()

	for i := 0; i < 10; i++ {
		pod.Emit(message.New("default", "12", []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Duration(time.Second))
}
