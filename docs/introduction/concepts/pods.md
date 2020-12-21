# Pods

The `Grav` instance on its own is not very useful without anything connected to it. A `Pod` is a lightweight bi-directional connection to the Grav instance. A Pod is created by calling the `Connect()` method on the Grav instance. The Grav instance can be connected to many pods, up to thousands at once. A Pod can send messages to be routed through the bus, and it can receive messages from the bus. 

