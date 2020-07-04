package main

import (
	"fmt"
	"time"

	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/grav/message"
)

func main() {
	g := grav.New()

	pod := g.Connect()

	pod.On(func(msg message.Message) error {
		fmt.Println(string(msg.Data()))
		return nil
	})

	pod.Emit(message.New("default", "12", []byte("hello, world")))

	time.Sleep(time.Duration(time.Second * 5))
}
