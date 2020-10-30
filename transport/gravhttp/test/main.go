package main

import (
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/gravhttp"
	"github.com/suborbital/vektor/vlog"
)

func main() {
	ghttp := gravhttp.New()
	g := grav.NewWithOptions(&grav.Opts{
		Logger:    vlog.Default(),
		Port:      8080,
		Transport: ghttp,
		Discovery: nil,
	})

	g.ConnectEndpoint("http://hello.com")
}
