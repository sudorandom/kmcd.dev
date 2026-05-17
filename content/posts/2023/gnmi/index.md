---
categories: ["opinion"]
tags: ["networking", "gnmi", "snmp", "monitoring", "network-management", "open-source", "grpc"]
keywords: ["gnmi vs grpc"]
date: "2023-11-04"
description: "gNMI is better than SNMP and more people need to know about it."
cover: "cover.svg"
images: ["/posts/gnmi/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Why you should use gNMI over SNMP in 2026"
slug: "gnmi"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/gnmi/
mastodonID: "112277288984060202"
---

Network engineers deal with a unique set of headaches when managing infrastructure. [SNMP](https://en.wikipedia.org/wiki/Simple_Network_Management_Protocol) is over 30 years old, and most networks still depend on it today. We finally have a strong modern alternative and it is time to move on.

SNMP has been the standard for decades, but its flaws are hard to ignore now. It is clunky, inefficient, and simply does not scale in modern environments.

[gNMI](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md) (gRPC Network Management Interface) solves these problems. It is cleaner, faster, and gives administrators much better control over what data they pull and how they receive it.

The protocol relies on three main operations:
* **Get:** Pull data from a device.
* **Set:** Change a configuration.
* **Subscribe:** Get automated updates whenever data changes.

### Why gNMI beats SNMP

The benefits come down to a few key architectural shifts.

* **Model-driven design:** gNMI uses [YANG](https://datatracker.ietf.org/doc/html/rfc6020) to define data. This makes writing automation scripts much easier because you actually know what the data structure looks like without hunting through MIBs.
* **Truly bidirectional:** You can use gNMI for both telemetry and configuration. This lets you handle everything from provisioning to troubleshooting with one tool.
* **Efficiency and scale:** gNMI uses a streaming approach. It can handle high traffic volumes across massive networks without killing device performance.
* **Modern security:** It is built on [HTTP/2](https://httpwg.org/specs/rfc7540.html) and uses [TLS](https://datatracker.ietf.org/doc/html/rfc8446) to encrypt traffic by default. You get a secure management plane right out of the box.

Here is a quick look at how they stack up:

| Feature | SNMP | gNMI |
| :--- | :--- | :--- |
| **Transport** | UDP (mostly) | HTTP/2 (TCP) |
| **Data Format** | [ASN.1](https://en.wikipedia.org/wiki/ASN.1) (MIBs) | Protocol Buffers (modeled with YANG/OpenConfig) |
| **Speed** | 30s to 5min intervals | Near real-time streaming |
| **Security** | Shared secrets (v2) / Complex USM (v3) | Certificate-based Mutual TLS |

### Subscriptions: Stop Polling, Start Streaming

The "streaming" aspect is a massive upgrade. Because gNMI uses [gRPC](https://grpc.io/docs/what-is-grpc/core-concepts/#server-streaming-rpc), it can hold a persistent connection where the device pushes updates to the client. SNMP has no suitable way to do this{{< footnote 1 >}}. Instead, SNMP forces you into a repetitive request and response loop.

Look at a typical SNMP setup:

{{< d2 >}}
shape: sequence_diagram
client: SNMP Client
device: SNMP Device

client -> device: Get Interface Statistics
device -> client: "Interface Statistics: Ethernet1/inOctets = 1000000"

client -> device: Get Interface Statistics
device -> client: "Interface Statistics: Ethernet1/inOctets = 1000000"

client -> device: Get Interface Statistics
device -> client: "Interface Statistics: Ethernet1/inOctets = 1000000"

client -> device: Get Interface Statistics
device -> client: "Interface Statistics: Ethernet1/inOctets = 1000420"
{{< /d2 >}}

The client has to ask for the same data over and over, often getting the exact same answer. I am also sparing you the typical SNMP mess where you have to manually map index numbers to interface names. If an interface is "too fast," you have to mess with [ifHCInOctets](https://datatracker.ietf.org/doc/html/rfc2233#section-3.1.6) values. With SNMP, you have to poll frequently to get resolution on the data.

Now look at a gNMI subscription:

{{< d2 >}}
shape: sequence_diagram
client: gNMI Client
device: gNMI Device

client -> device: Subscribe To Interface Statistics
device -> client: Subscription established

device -> client: "/interfaces/interface[name=Ethernet1]/state/counters/in-octets = 1000000"

device -> client: "/interfaces/interface[name=Ethernet1]/state/counters/in-octets = 1000420"
{{< /d2 >}}

You set the subscription once and the device sends updates only when the value changes. If nothing changes, the device stays quiet. This massive reduction in "chatter" lowers the load on your hardware and your network.

### Architecture: How it actually works

Moving to gNMI means rethinking where your data goes. SNMP usually feeds into a monolithic Network Management System (NMS). gNMI typically flows into a Time Series Database (TSDB) like [Prometheus](https://prometheus.io/) or [InfluxDB](https://www.influxdata.com/) via a telemetry collector that translates the stream into metrics Prometheus can scrape.

The data itself is sent as binary using [Protocol Buffers](https://protobuf.dev/) (Protobuf). This makes it incredibly efficient over the wire, but it does mean you cannot just read it in plain text with Wireshark unless you have the right dissectors configured.

A major architectural shift here is **Dial-Out** telemetry. With traditional Dial-In, your collector connects to every single device. With Dial-Out, the devices are configured to actively push data to a central destination. This simplifies firewall rules and bootstrapping, but it also shifts connection management and scaling complexity onto the devices themselves, especially in very large deployments.

### What about NETCONF?

Since we are talking about YANG models, you might wonder why we are not just using [NETCONF](https://datatracker.ietf.org/doc/html/rfc6241). Both have their place in modern networks.

NETCONF uses [XML](https://www.w3.org/XML/) and is heavily focused on transactional configuration. It is fantastic when you need to apply a complex, multi-device configuration change and ensure it either fully succeeds or rolls back. However, XML is heavy. For high-speed telemetry and streaming state data, gNMI with its binary Protobuf format is far superior.

### The Gotchas

I will admit gNMI is not a perfect solution. Advocacy is useless if we ignore the hurdles.

First, there is a CPU tax. gRPC and TLS encryption require more overhead on the network device than a simple UDP-based SNMP poll. Older hardware might actually struggle with this load.

Second, navigating [OpenConfig](https://www.openconfig.net/projects/models/) models can be intimidating at first. While YANG is infinitely better than hunting through ancient MIBs, you still have to understand the "YANG tree" structure to know exactly what paths to subscribe to. The learning curve is definitely steeper.

### Better Tooling and Open Standards

Despite the learning curve, the ecosystem is catching up fast. Tools like [`gNMIc`](https://gnmic.openconfig.net/) provide a much better user experience than old school commands like `snmpget`. Plus, gNMI is an [open standard](https://github.com/openconfig/gnmi). It is not locked to one vendor. Even when using vendor specific data models, they are almost always described in YANG, which makes documentation and automation much more predictable.

gNMI is the logical choice for most modern networks. I even suspect it is a great fit for smaller setups like homelabs, though I will save that for a later post. There is plenty more to dive into, including different subscription types like [STREAM](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md#35152-stream-subscriptions) or [ONCE](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md#35151-once-subscriptions), but those deserve their own deep dives. Thanks for reading.

{{< references >}}
{{< footnotelist >}}
{{< footnoteitem 1 "SNMP’s push mechanisms (traps/informs) are unreliable and not suited for structured telemetry." >}}
{{< /footnotelist >}}
{{< /references >}}
