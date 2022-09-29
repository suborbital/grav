## Grav has been deprecated. You can use the new [bus package](https://github.com/suborbital/e2core/tree/main/bus), which is a drop-in continuation of this project

# Concepts

Since Grav is embedded, it is instantiated as a `grav.Grav` object which your application code connects to in order to send and receive messages. Grav connects to other nodes via [transport plugins](https://github.com/suborbital/grav/docs/networking/transports/) which extend the Grav core to become a networked distributed messaging mesh. Grav instances can also discover each other automatically using [discovery plugins](https://github.com/suborbital/grav/docs/networking/discovery/). Grav does not require a centralized broker, and as such has some limitations, but for certain applications it is vastly simpler \(and more extensible\) than a centralized messaging system.

To start off, this guide will cover some concepts that are important to understand before delving into using Grav.

