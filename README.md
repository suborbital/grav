![logo_transparent](https://user-images.githubusercontent.com/5942370/88551418-d623ea00-cff0-11ea-87d8-e9b94174aaa2.png)

Grav is an embedded distributed messaging mesh for Go applications. Grav allows interconnected components of your systems to communicate effectively in a reliable, asynchronous manner. HTTP and RPC are hard to scale well in modern distributed systems, but Grav is designed to be performant and resilient in various distributed environments.

Since Grav is embedded, it is instantiated as a `grav.Grav` object which your application code connects to in order to send and recieve messages. Grav connects to other nodes via **transport plugins** such as [gravwebsocket](./transport/gravwebsocket/README.md) which extends the Grav core to become a networked distributed messaging system. Grav nodes can also be configured to automatically discover each other using **discovery plugins**. Grav does not require a centralized broker, and as such has some limitations, but for certain applications it is vastly simpler (and more extensible) than a centralized messaging system.

## Documentation
Full documentation can be found on the [Grav docs site](https://grav.suborbital.dev).

## Project status

Grav is currently in beta, and is being developed alongside (and is designed to integrate with) [Vektor](https://github.com/suborbital/vektor) and [Hive](https://github.com/suborbital/hive). It is also the messaging core that powers Suborbital's flagship project, [Atmo](https://github.com/suborbital/atmo).

Copyright Suborbital contributors 2021.
