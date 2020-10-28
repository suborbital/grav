package main

import (
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/gravhttp"
)

func main() {
	ghttp := gravhttp.New(grav.DefaultTransportOpts())
	g := grav.NewWithTransport(ghttp, nil)

	g.ConnectEndpoint("http://hello.com")
}
