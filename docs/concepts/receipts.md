## Grav has been deprecated. You can use the new [bus package](https://github.com/suborbital/e2core/tree/main/bus), which is a drop-in continuation of this project

# Receipts

When you send a message using a Pod, it will return a `MsgReceipt`. This object is a reference to the message that you sent, and allows you to easily get replies to the message. A receipt is an extension of the Pod that sent the original message, so any methods that you call are essentially called on the Pod itself. This means that calling `receipt.WaitOn(msgFunc)` is a shortcut for calling `WaitOn` on the Pod itself \(with filtering enabled to only receive replies to the original message\).

Receipts are the basis for request/reply with Grav. They will be discussed in detail in the [Usage](https://github,com/suborbital/grav/docs/usage) section.

