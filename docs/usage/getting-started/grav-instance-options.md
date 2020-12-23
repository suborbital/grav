# Grav Instance Options

When calling `grav.New`, you can optionally include some options:

```go
// UseLogger allows a custom logger to be used
func UseLogger(logger *vlog.Logger) OptionsModifier

// UseTransport sets the transport plugin to be used.
func UseTransport(transport Transport) OptionsModifier

// UseEndpoint sets the endpoint settings for the instance to broadcast for discovery
// Pass empty strings for either if you would like to keep the defaults (8080 and /meta/message)
func UseEndpoint(port, uri string) OptionsModifier

// UseDiscovery sets the discovery plugin to be used
func UseDiscovery(discovery Discovery) OptionsModifier
```

If no options are passed, then the default options are used:

```go
Logger:    vlog.Default(),
Port:      "8080",
URI:       "/meta/message",
Transport: nil,
Discovery: nil,
```

