package main

import (
	"fmt"

	"github.com/suborbital/grav/grav"
)

func gettingStarted() {
	g := grav.New()

	fmt.Println(g.NodeUUID)
}
