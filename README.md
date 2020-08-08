![logo_transparent](https://user-images.githubusercontent.com/5942370/88551418-d623ea00-cff0-11ea-87d8-e9b94174aaa2.png)

Grav is an embedded distributed message bus. It is designed with a very narrow purpose, and is not meant to replace more general-purpose systems such as RabbitMQ, Kafka, or others. Grav is designed to allow interconnected components of your systems to communicate effectively in a reliable, asynchronous manner. HTTP and RPC do not scale well in modern distributed systems, but Grav is designed to be performant and resilient in various distributed environments. This project has several goals and a few non-goals:

Goals:
- Have very low resource and memory consumption.
- Be resilient against data loss due to node failure.
- Act as a reliable core upon which more complex behaviour can be built.
- Define a minimal data format that is meant to be extended for a particular purpose.
- Support request/reply and pub/sub message patterns.
- Support internal (in-process) and external (networked, via transport plugins) publishers and consumers equally.

Non-Goals:
- Replace a brokered message queue for large-scale systems.
- Have extremely high throughput capabilities.
- Support every type of messaging pattern.

## Background

In a search for the right messaging system to use in concert with Hive and Vektor, many options were evaluated. Unfortunately, every option either required the use of CGO, or relied on a centralized broker which would complicate deployments. Reluctantly, the decision was made to implement a message bus of our own. I say reluctantly as there is nothing worse than re-inventing the wheel, but alas none of the mainstream projects support the use-cases that Suborbital's frameworks are aiming to handle.

To avoid "Yak Shaving" as much as possible, Grav is designed to be a reliable core that is focussed on being a very reliable, performant message bus. Anything beyond that, including the transport layer, is not part of Grav itself. We will be providing a few transport implementations, likely using HTTP and gRPC, and the hope is that it should be incredibly easy to extend Grav for your particular use-case.

## Intro

Grav is an embedded message bus, meaning that it is instantiated as a `grav.Grav` object, and then your application code connects to that object to send and recieve messages. Grav connects to other nodes via transport plugins such as [grav-transport-http](https://github.com/suborbital/grav-transport-http) which extends the Grav core to become a networked distributed messaging system. Grav does not require a centralized broker, and as such has some limitations, but for certain applications it is vastly simpler (and more extensible) than a centralized messaging system.

## Project status

Grav is currently in prototype stages and is being developed alongside (and is designed to integrate with) [Vektor](https://github.com/suborbital/vektor) and [Hive](https://github.com/suborbital/hive), who are nearing their own beta status.

Documentation will be coming soon.

Copyright Suborbital contributors 2020