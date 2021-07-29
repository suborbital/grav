# Transports

Transports are plugins that allow Grav instances to connect to one another over the network. In order to keep Grav itself as simple as possible, all messages remain in-process only unless a transport plugin is configured.

There are two types of Transports; mesh and bridge.
* Mesh transports connect Grav nodes to one another directly, forming a mesh network of instances which is completely decentralized.
* Bridge transports connect a Grav node to a 'bridge' such as a centralized broker to allow for additional topographies.

Grav has three first-party transports:

* HTTP, a simplistic transport using HTTP requests to emit messages.
* Websocket, a streaming transport based on standard websockets.
* NATS, a streaming bridge transport based on the popular NATS server.

Grav transports are designed as plugins, and as such anyone can create one for their own purposes. Transports for additional platforms such as Kafka are planned. See the [transport](https://github.com/suborbital/grav/blob/main/transport) directory to see example transport code.

