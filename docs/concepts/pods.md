## Grav has been deprecated. You can use the new [bus package](https://github.com/suborbital/e2core/tree/main/bus), which is a drop-in continuation of this project

# Pods

The `Grav` instance on its own is not very useful without anything connected to it. A `Pod` is a lightweight bi-directional connection to the Grav instance. A Pod is created by calling the `Connect()` method on the Grav instance. The Grav instance can be connected to many Pods, up to thousands at once. A Pod can send messages to be routed through the bus, and it can receive messages from the bus.

The two core methods of a Pod are `Send(msg)` and  `On(msgFunc)`, which allow sending and receiving messages. The caller of `On` provides a function to handle incoming messages \(called the `receive function`\), and it is called each time a message comes from the Grav instance. There are other methods to do more specific operations, which will be discussed in the [Usage](https://github.com/suborbital/grav/docs/usage) section.

A Pod is mainly just a connection, allowing the Grav instance to do the majority of the work, but it does contain some limited "smarts". For example, Pods include a filter that gives you control over which messages get received. Pods are in constant communication with the Grav instance, and they are able to report things such as their health, failed message delivery, and more. Unhealthy Pods will be disconnected after a certain number of failed deliveries, but pods that recover automatically get failed messages replayed.

Using Pods is discussed in detail in the [Usage](.https://github.com/suborbital/grav/usage) section.

