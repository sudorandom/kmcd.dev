---
categories: ["opinion"]
tags: ["networking", "yang", "data-modeling", "openconfig", "netconf", "gnmi", "graphql", "smithy"]
keywords: ["what is yang", "yang vs json schema", "openconfig yang", "yang vs graphql", "yang vs smithy"]
date: "2026-05-26T10:00:00Z"
description: "Why the networking world built its own data modeling language and what software engineers can learn from it."
cover: "cover.svg"
images: ["/posts/yang/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "YANG: The Schema Language Networking Desperately Needed"
slug: "yang"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/yang/
draft: true
---

We are finally moving away from relying on protocols like [SNMP](https://en.wikipedia.org/wiki/Simple_Network_Management_Protocol). My [previous post](/posts/gnmi/) covered why [gNMI](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md) is the future for telemetry, but the protocol is only half the problem. You still need to agree on what the data actually looks like.

That is where [YANG](https://datatracker.ietf.org/doc/html/rfc7950) comes in. It is a data modeling language that defines the configuration, state, and RPCs for network hardware.

The weird part for software engineers is how YANG totally decouples the model from the wire format. We usually reach for tools like [OpenAPI](https://www.openapis.org/), which tie the schema directly to REST endpoints. YANG models the actual domain. You can encode the payload in [XML](https://www.w3.org/XML/), [JSON](https://www.json.org/), or [Protocol Buffers](https://protobuf.dev/), and ship it over [NETCONF](https://datatracker.ietf.org/doc/html/rfc6241) or gNMI. The schema describes the physical router. How you transport the data is a separate problem.

Networking built its own Domain Specific Language (DSL) entirely out of necessity. Looking at it today, it is fair to ask why they didn't just adopt existing tools.

### Why not JSON Schema, GraphQL, or Smithy?

A lot of it comes down to timing. [GraphQL](https://graphql.org/learn/schema/) and [AWS Smithy](https://smithy.io/) didn't exist when the IETF started drafting YANG in the late 2000s. The industry relied on SNMP MIBs, which worked for reading state but fell apart when writing complex configs. [XML Schema (XSD)](https://www.w3.org/XML/Schema) had the structural power, but frankly, nobody wants to write SOAP envelopes to manage a core switch.

Network hardware also strictly separates two types of data:
1. **Configuration Data:** What you want the device to do (e.g., IP addresses, BGP neighbor settings).
2. **State Data:** What the device is actually doing right now (e.g., packet counters, CPU temperature).

Engineers needed a way to guarantee a config payload was valid before applying it, alongside defining read-only telemetry. Even if modern tools had been around, they solve different problems.

GraphQL SDL handles the state/config split reasonably well using Queries and Mutations. The issue is that GraphQL is an API contract detailing client-server communication. YANG maps out the hardware layout.

AWS Smithy is conceptually closer. It models cloud services independently of the protocol, letting you generate OpenAPI specs or Protobufs from one source. While YANG takes a similar approach, the targets differ. Smithy generates cloud SDKs. YANG enforces referential integrity on embedded devices and handles deep configuration inheritance.

### The Power of the DSL and OpenConfig

YANG uses a strongly typed tree hierarchy. The killer feature here is tagging data natively with `config true` (read/write) or `config false` (read-only).

Early vendor models mixed these properties together in the same lists, making automation incredibly frustrating. The [OpenConfig](https://www.openconfig.net/) working group eventually stepped in to enforce a strict structural pattern around those native `config` tags.

Now, instead of a messy mix of leaves, every major component needs a dedicated `config` container and a `state` container. Here is a simplified look at the [OpenConfig interfaces model](https://github.com/openconfig/public/blob/master/release/models/interfaces/openconfig-interfaces.yang):

```yang
container interfaces {
  description "Top level container for interfaces";

  list interface {
    key "name";

    leaf name {
      type leafref {
        path "../config/name";
      }
    }

    container config {
      description "Configurable items at the global, physical interface level";
      uses interface-phys-config;
    }

    container state {
      config false;
      description "Operational state data at the global interface level";
      uses interface-phys-config;
      uses interface-common-state;
      uses interface-counters-state;
    }
  }
}
```

That `config false;` line cascades down and locks the entire operational branch.

Complex infrastructure footprints require heavy reusability and validation. You define blocks of nodes using `grouping` and inject them with `uses`, similar to mixins. Referential integrity is handled by `leafref`. If a routing protocol references a network interface, the `leafref` guarantees the interface exists in the config before the router accepts the payload.

We usually rely on tools like [`pyang`](https://github.com/mbj4668/pyang) to visualize these constraints. It dumps the model into a readable tree:

```text
module: openconfig-interfaces
  +--rw interfaces
     +--rw interface* [name]
        +--rw name      -> ../config/name
        +--rw config
        |  +--rw name?            string
        |  +--rw type             identityref
        |  +--rw enabled?         boolean
        +--ro state
           +--ro name?            string
           +--ro type             identityref
           +--ro enabled?         boolean
           +--ro admin-status?    enumeration
           +--ro oper-status?     enumeration
           +--ro counters
              +--ro in-octets?    yang:counter64
              +--ro out-octets?   yang:counter64
```

A script reading this schema immediately knows what it can touch. It can toggle the `enabled` boolean under `config`, but it is restricted to just monitoring the `in-octets` counter under `state`.

### Vendor Neutrality and the Ecosystem Tax

OpenConfig is also the main engine behind vendor neutrality. An interface path on a Cisco box historically looked nothing like one on a Juniper switch, forcing you to rewrite automation logic per vendor.

With operators like Google and Microsoft backing OpenConfig, automation can just target a generic "OpenConfig router". The hardware vendors handle translating that standard model into their proprietary backends. Consumers actually get some leverage back.

It isn't a perfect system. A custom DSL means custom tooling, and the software ecosystem around YANG is extremely narrow. C/C++ developers get [`libyang`](https://github.com/CESNET/libyang). Python developers get `pyang`. I write a lot of Go, so I rely on [`ygot`](https://github.com/openconfig/ygot). If you want native parsing in Rust or TypeScript, you are mostly on your own with some massive RFCs to read.

Processing these models is computationally expensive. Parsing a modern router schema means resolving external module imports, expanding groupings, and mapping cross-tree references. Building that Abstract Syntax Tree (AST) in memory can eat hundreds of megabytes of RAM. You might wait several seconds just for the compilation to finish before processing a single byte of telemetry data.

The language is a beast to parse and frustrating to learn. Still, a purpose-built schema makes sense when you need absolute certainty about the difference between intended configuration and the live operational reality of a network. It is an ecosystem tax I'm willing to pay if it means I never have to regex-match CLI output again.
