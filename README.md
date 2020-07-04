# Grav

Grav is a pure Go distributed message bus. It is designed with a very narrow purpose, and is not meant to replace more general-purpose systems such as RabbitMQ, Kafka, or others. This project has several goals and a few non-goals:

Goals:
- Have very low resource and memory consumption
- Be very resilient against data loss due to node and network failure
- Act as a flexible transport layer upon which more complex behaviour can be built
- Define a minimal data format that is meant to be extended for each particular purpose
- Support request/reply and pub/sub message patterns
- Provide secure communication and secure defaults
- Support internal (in-process) and external (networked) publishers and consumers equally

Non-Goals:
- Replace a brokered message queue for large-scale systems
- Support every type of messaging pattern

## Background

In a search for the right messaging system to use in concert with Hive and Vektor, many options were evaluated. Unfortunately, every option either required the use of CGO, or relied on a centralized broker which would complicate deployments. Reluctantly, the decision was made to implement a message bus of our own. I say reluctantly as there is nothing worse than re-inventing the wheel, but alas none of the mainstream projects support the use-cases that Suborbital's frameworks are aiming to handle.

To avoid "Yak Shaving" as much as possible, Grav is designed to be a transport layer only, and is built as a thin layer on top of gRPC to take advantage of its robustness and protobuf serialization format.

## Project status

Grav is currently in prototype stages and will be developed alongside Vektor and Hive, who are nearing their own beta status.

Copyright Suborbital contributors 2020