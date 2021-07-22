# Transports

Transports are plugins that allow Grav instances to connect to one another over the network. In order to keep Grav itself as simple as possible, all messages remain in-process only unless a transport plugin is configured.

Grav has two "first-party" transports:

* gravhttp, a simplistic transport using HTTP requests to emit messages.
* gravwebsocket, a streaming transport based on standard websockets.

Grav transports are designed as plugins, and as such anyone can create one for their own purposes. Transports for various popular products such as NATS and Kafka are planned. See the [transport](https://github.com/suborbital/grav/blob/main/transport) directory to see example transport code.

