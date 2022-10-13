## Grav has been deprecated. You can use the new [bus package](https://github.com/suborbital/e2core/tree/main/bus), which is a drop-in continuation of this project. 

# Grav

### The distributed async message bus for Go

Grav is an embedded distributed messaging library for Go applications 
that allows interconnected components of your system to communicate 
effectively in a reliable, asynchronous manner. HTTP and RPC are difficult 
to scale well in modern distributed systems, so Grav was created with
end goal of adding a performant and resilient messaging system to 
various distributed environments.

Grav's main purpose is to act as a flexible abstraction that allows 
your application to discover and communicate using a variety of 
protocols without needing to re-write any code.

As of today, this project has several goals and a few non-goals as listed below.

**Goals:**

* Have very low resource and memory consumption.
* Be resilient against data loss due to node failure.
* Act as a reliable core upon which more complex behaviour can be built.
* Support request/reply and broadcast message patterns.
* Support internal \(in-process\) and external \(networked\) messaging equally.

**Non-Goals:**

* Support every type of messaging pattern.

Since Grav is embedded, it is instantiated as a `grav.Grav` object
that your application code connects to in order to send and receive messages. 
Grav connects to other nodes via [transport plugins](https://github.com/suborbital/grav/docs/networking/transports) which extend
the Grav core to become a networked distributed messaging mesh. 
Grav instances can also discover each other automatically using 
[discovery plugins](https://github,com/suborbital/grav/docs/networking/discovery). 
 
 Grav does not require a centralized broker, and as such has some limitations, 
 but for certain applications it is vastly simpler (and more extensible) 
 than a centralized messaging system.