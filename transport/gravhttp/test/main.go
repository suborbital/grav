package main

import (
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/gravhttp"
)

func main() {
	ghttp := gravhttp.New()

	g := grav.New(
		grav.UseTransport(ghttp),
	)

	g.ConnectEndpoint("http://hello.com")
}
