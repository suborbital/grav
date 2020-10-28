package main

import (
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/gravhttp"
	"github.com/suborbital/vektor/vlog"
)

func main() {
	ghttp := gravhttp.New(&gravhttp.Options{Logger: vlog.Default()})
	g := grav.NewWithTransport(ghttp, grav.DefaultTransportOpts())

	g.ConnectEndpoint("http://hello.com")
}
