# Welcome

### Welcome to the Grav Guide

Grav is an embedded distributed messaging mesh for Go applications. Grav allows interconnected components of your systems to communicate effectively in a reliable, asynchronous manner. HTTP and RPC are hard to scale well in modern distributed systems, but Grav is designed to be performant and resilient in various distributed environments.

This project has several goals and a few non-goals:

Goals:

* Have very low resource and memory consumption.
* Be resilient against data loss due to node failure.
* Act as a reliable core upon which more complex behaviour can be built.
* Support request/reply and broadcast message patterns.
* Support internal \(in-process\) and external \(networked\) messaging equally.

Non-Goals:

* Replace a brokered message queue for large-scale systems.
* Have extremely high throughput capabilities.
* Support every type of messaging pattern.

Grav is designed with a very narrow purpose, and is not meant to replace more general-purpose systems such as RabbitMQ, Kafka, or others

