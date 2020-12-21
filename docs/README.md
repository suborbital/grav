# Welcome

### Welcome to the Grav Guide

Grav is an embedded distributed message bus. It is designed with a very narrow purpose, and is not meant to replace more general-purpose systems such as RabbitMQ, Kafka, or others. Grav is designed to allow interconnected components of your systems to communicate effectively in a reliable, asynchronous manner. HTTP and RPC are hard to scale well in modern distributed systems, but Grav is designed to be performant and resilient in various distributed environments.

Since Grav is embedded, it is instantiated as a `grav.Grav` object which your application code connects to in order to send and recieve messages. Grav connects to other nodes via transport plugins such as [gravwebsocket](https://github.com/suborbital/grav/blob/master/transport/gravwebsocket/README.md) which extends the Grav core to become a networked distributed messaging system. Grav does not require a centralized broker, and as such has some limitations, but for certain applications it is vastly simpler \(and more extensible\) than a centralized messaging system.

This project has several goals and a few non-goals:

Goals:

* Have very low resource and memory consumption.
* Be resilient against data loss due to node failure.
* Act as a reliable core upon which more complex behaviour can be built.
* Define a minimal data format that is meant to be extended for a particular purpose.
* Support request/reply and broadcast message patterns.
* Support internal \(in-process\) and external \(networked, via transport plugins\) publishers and consumers equally.

Non-Goals:

* Replace a brokered message queue for large-scale systems.
* Have extremely high throughput capabilities.
* Support every type of messaging pattern.

