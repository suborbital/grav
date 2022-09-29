## Grav has been deprecated. You can use the new [bus package](https://github.com/suborbital/e2core/tree/main/bus), which is a drop-in continuation of this project. You can find the [docs](https://github,com/suborbital/grav/docs) for this deprecated project in this repo.

Grav is an embedded distributed messaging library for Go applications. Grav allows interconnected components of your system to communicate effectively in a reliable, asynchronous manner. HTTP and RPC are hard to scale well in modern distributed systems, so we created Grav to add a performant and resilient messaging system to various distributed environments.

Grav's main purpose is to act as a flexible abstraction that allows your application to discover and communicate using a variety of protocols without needing to re-write any code.

Grav messages can be sent in-process (such as between Goroutines), or to other nodes via **transport plugins** such as [Websocket](./transport/websocket/README.md) and [NATS](./transport/nats/README.md). Transport plugins extend the core Grav bus to become a networked distributed messaging system. Grav nodes can also be configured to automatically discover each other using **discovery plugins**. Grav can operate as a decentralized mesh or integrate with centralized streaming platforms, making it extremely flexible.

Copyright Suborbital Contributors 2021.
