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

Modern network automation is shifting away from protocols like [SNMP](https://en.wikipedia.org/wiki/Simple_Network_Management_Protocol). While my [previous post](/posts/gnmi/) covered why [gNMI](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md) is the future of telemetry and management, protocols only define how data moves. They do not define the data itself.

[YANG](https://datatracker.ietf.org/doc/html/rfc7950) (Yet Another Next Generation) is a data modeling language used to define the configuration, state data, RPCs, and notifications for network devices. 

A critical distinction is that YANG models the data completely independent of the transport or encoding format. Software engineers are familiar with tools like [OpenAPI](https://www.openapis.org/), which explicitly document the types and endpoints for a specific REST API. YANG operates differently because it models the domain itself. The data described by a YANG schema can be encoded as [XML](https://www.w3.org/XML/), [JSON](https://www.json.org/), or [Protocol Buffers](https://protobuf.dev/), and it can be transported over [NETCONF](https://datatracker.ietf.org/doc/html/rfc6241), [RESTCONF](https://datatracker.ietf.org/doc/html/rfc8040), or gNMI. The schema defines the physical and logical reality of the device, not the mechanism used to communicate with it.

The networking domain created a Domain Specific Language (DSL) specifically to model data for network devices. To understand why, it helps to look at the historical context of its creation and how it compares to the tools software engineers use today.

### What was lacking in other options?

Software engineers looking at YANG today often ask: *Why not just use [JSON Schema](https://json-schema.org/), or modern interface languages like [GraphQL SDL](https://graphql.org/learn/schema/) or [AWS Smithy](https://smithy.io/)?*

The first answer is chronology. When the IETF started designing YANG in the late 2000s, GraphQL and Smithy did not exist. The networking industry relied on SNMP MIBs (Management Information Bases) and [XML Schema (XSD)](https://www.w3.org/XML/Schema). While MIBs were structured for reading state, they lacked the strong typing and transactional safeguards required for complex configuration writes. XSD was too verbose and lacked domain-specific semantics. 

Network devices strictly separate two types of data:
1. **Configuration Data:** Read/write settings applied to a device (e.g., IP addresses, port states).
2. **State Data:** Read-only operational reality of the device (e.g., packet counters, CPU temperature).

Network engineers needed a way to guarantee a configuration payload was valid before deployment while also defining available read-only telemetry paths. Even if modern tools had existed at the time, they are optimized for entirely different domains:

* **GraphQL SDL:** GraphQL's Schema Definition Language handles the state versus configuration problem cleanly through operations. The `Query` type defines the read-only operational state you can pull, while the `Mutation` type strictly defines what you are allowed to change. However, GraphQL is fundamentally bound to the API interaction layer. It defines how a client queries a server. YANG defines the physical and logical structure of the hardware itself, regardless of how you query it.
* **AWS Smithy:** Amazon's Smithy is conceptually the closest modern cousin to YANG. Smithy models cloud services independently of their underlying protocol, allowing developers to generate OpenAPI specs or Protobuf schemas from a single source of truth. YANG does the exact same thing for network infrastructure. The difference is the target domain. Smithy optimizes for cloud service operations and SDK generation, while YANG optimizes for hardware constraints, hierarchical configuration inheritance, and deep referential integrity. 

### The Power of the YANG DSL

YANG uses a tree hierarchy where every leaf is strongly typed. Its most vital feature for infrastructure is the ability to explicitly tag data using the `config true` (read/write) or `config false` (read-only) statements. 

Historically, early IETF YANG models would intermingle configuration and state data within the same lists. A massive step forward in the industry was the [OpenConfig](https://www.openconfig.net/) working group, which took YANG's native `config` property and built a strict structural convention around it.

Instead of mixing read/write and read-only leaves, OpenConfig enforces a pattern where every major component has a dedicated `config` container and a `state` container. Here is a simplified snippet of how this looks using the [OpenConfig interfaces model](https://github.com/openconfig/public/blob/master/release/models/interfaces/openconfig-interfaces.yang):

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

The `config false;` statement under the `state` container enforces the read-only constraint for that entire branch.

### Beyond the Basics: Advanced Schema Features

The language is deeply involved and goes far beyond basic type definitions. It includes mechanisms for strict data validation and schema reusability that are critical for managing complex infrastructure.

* **Reusability (`grouping` and `uses`):** Similar to inheritance or mixins in software development, you can define a block of nodes as a `grouping` and inject it anywhere in the tree using the `uses` statement. The OpenConfig snippet above uses this heavily.
* **Referential Integrity (`leafref`):** You can enforce that a value must exist elsewhere in the configuration. For example, if a routing protocol references a network interface, the `leafref` ensures that the specific interface actually exists before the configuration is accepted.
* **Field Validation:** YANG supports complex constraints. You can restrict strings with regular expressions (`pattern`), limit integers to specific bounds (`range`), and use XPath expressions (`must`) to enforce conditional logic across entirely different parts of the tree.
* **Mandatory and Optional Fields:** Nodes can be explicitly marked with `mandatory true` to ensure the device rejects any payload missing critical information, while optional fields can be assigned explicit `default` values.

Tools are typically used to visualize this tree structure and these constraints. For example, [`pyang`](https://github.com/mbj4668/pyang) outputs the model into a visual tree:

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

A script reading this schema knows it can modify the `enabled` boolean under `config`, but can only monitor the `in-octets` counter under `state`. This standardized relationship enables features like gNMI subscriptions.

### OpenConfig: The Dream of Vendor Neutrality

Beyond establishing structural conventions, OpenConfig drives significant vendor-neutral YANG development.

Historically, network vendors (Cisco, Juniper, Arista) structured data differently. An interface on a Cisco device had a different path than on a Juniper device, requiring automation scripts to be rewritten per vendor.

OpenConfig, led by operators like Google and Microsoft, publishes vendor-neutral YANG models. Automation can target an "OpenConfig router" rather than a specific vendor's hardware. The vendors translate the OpenConfig YANG model into their proprietary backend. This shifts control toward consumers and relies on the strict contracts YANG provides.

### The Downsides: Ecosystem and AST Performance

Building a custom DSL comes with severe trade-offs. The language specification is massive, and because of this complexity, the software ecosystem around YANG is surprisingly narrow.

While JSON or [YAML](https://yaml.org/) have robust, production-ready parsing libraries in virtually every programming language, YANG relies on a handful of specialized open-source projects. If you are developing in C or Python, you have access to [`libyang`](https://github.com/CESNET/libyang) or `pyang`. If you are working in [Go](https://go.dev/), you can use [`ygot`](https://github.com/openconfig/ygot). However, if you want to interact natively with YANG models in a language outside of this small circle, you must write your own parser. Given the size of the RFCs defining the language, this is a monumental task.

Additionally, handling the language introduces significant performance bottlenecks. To process a YANG model, a parser must resolve external module imports, expand inherited groupings, and map complex cross-tree references. This requires building an enormous Abstract Syntax Tree (AST) in memory. 

Parsing the full schema for a modern router is computationally expensive. Loading these definitions can consume hundreds of megabytes of RAM and take several seconds just to compile the AST before a single byte of actual telemetry data is evaluated.

### Lessons Learned

When systems have strict operational requirements, like differentiating intended configuration from actual operational state, a purpose-built schema provides necessary constraints. YANG moved the networking industry from bespoke CLI commands to structured, contract-driven data. It requires a steep learning curve and tolerates high computational overhead, but establishes a necessary foundation for reliable infrastructure automation.