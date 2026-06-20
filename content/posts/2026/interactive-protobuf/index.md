---
categories: ["article", "project"]
tags: ["protobuf", "wasm", "go", "webassembly", "dataviz"]
date: "2026-06-25T10:00:00Z"
description: "Announcing protobuf.kmcd.dev, an interactive explainer and playground for exploring the binary details of Protocol Buffers."
cover: "cover.svg"
images: ["/posts/interactive-protobuf/cover.png"]
featuredalt: "A stylized illustration of the protobuf.kmcd.dev interface."
featuredpath: "date"
title: "Exploring Protocol Buffers Interactively"
slug: "interactive-protobuf"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/interactive-protobuf/
---

Protocol Buffers (protobuf) are a core part of modern RPCs, schema-driven APIs, and microservices. They work well, but learning how the format operates under the hood can be dry. Most of us stick to the high-level basics (like message fields and primitive types) and rarely need to dig into the wire format, varints, ZigZag encoding, or compiler plugins.

I wanted to change that by building a visual, hands-on guide.

Today I am sharing **[protobuf.kmcd.dev](https://protobuf.kmcd.dev)**. It is an interactive playground for exploring Protocol Buffers, from schema design and validation to binary wire formats and serialization efficiency.

{{< figure src="fixed-size-diagram.png" alt="A diagram showing fixed-size fields being encoded into protobuf" loading="lazy" >}}

This builds on a previous experiment, **[bgp.kmcd.dev](https://bgp.kmcd.dev)**, which I introduced in [Let's Learn About BGP](/posts/bgp-kmcd-dev/). That project showed me how much easier it is to understand the inner workings of a protocol when you can see them update in real time. I wanted to apply that same approach here.

---

## A Quick Tour

Instead of just writing static text, I built several custom widgets and sandboxes to show what happens when you compile and serialize protobuf data.

### Deep Dive into the Basics
The [Basics](https://protobuf.kmcd.dev/basics/) section covers schema design syntax and core protobuf concepts like enums, nested messages, repeated fields, maps, and oneofs. Code comparisons help explain primitive scalars, Google's Well-Known Types, and optimal integer guidelines.

### Visualizing the Wire Format
The [Binary & Wire Format](https://protobuf.kmcd.dev/binary/) page contains the core interactive tools on the site, and it is the section I am most interested in getting your thoughts on. It explains some basics of working with binary and then it breaks down how bytes are physically laid out on the wire:

* **Varint Explainer:** An interactive bit-level calculator. Input any integer to see how it converts from decimal to binary, splits into 7-bit groups, attaches continuation flags (MSB), and outputs final hex bytes.
* **ZigZag Explainer:** Input signed numbers and watch how the sign bit shifts to the LSB, showing the resulting bit arrangement and ZigZag value.
* **Tag Calculator:** Select a field number and a wire type to see the mathematical shift (`<< 3`), bitwise OR operations, and the resulting hex tag byte.
* **Interactive Wire Type Visualizers:** Flowcharts that explain wire-type parsing for Varints, Fixed width (32-bit and 64-bit), and Length-delimited payloads.
* **Binary Explorer (Segment Inspector):** An interactive split layout. You write a custom protobuf schema and standard JSON data. The tool compiles the schema, encodes the JSON to raw binary bytes, and maps them to a hex grid. Hovering or clicking on individual bytes in the grid highlights them and opens an inspector pane that breaks down exactly which field number, wire type, length, or payload value those bytes represent. You can also toggle a live Protoscope view to see the binary translate into diagnostic text.

{{< figure src="binary-explorer.png" alt="The Binary Explorer split layout showing a schema compiling to a raw hex grid" loading="lazy" >}}

* **Protoscope Lab:** A diagnostic sandbox where you can write raw Protoscope text on the left and see it compile into hex bytes on the right.

*I am particularly interested in whether this section hits the mark. Are these visualizations actually helpful for understanding the low-level encoding, or do they feel too contrived?*

{{< figure src="varint-calculator.png" alt="The Varint Explainer widget showing a live bit-level decimal to binary conversion" loading="lazy" >}}

### Real-Time Benchmarking & Efficiency
The [Efficiency & Performance](https://protobuf.kmcd.dev/efficiency/) page features a simulator with a JSON editor on the left and live metrics on the right.

You can write your own JSON structure or load preset profiles (Basic User, Nested Data, Packed List). The app compiles the schema, encodes the JSON into protobuf binary, and runs both payloads through browser-native GZIP compression. It renders bar charts comparing minified/gzipped JSON versus protobuf binary sizes, showing the payload reduction ratio. It also includes alerts for things like Gzip overhead if your payload is too small to benefit from compression.

{{< figure src="efficiency.png" alt="Real-time dashboard rendering animated bar charts that compare JSON and Protobuf payload sizes" loading="lazy" >}}

### Options, Reflection, and Validation
The **[Tooling & Reflection](https://protobuf.kmcd.dev/tooling/)** and **[Advanced](https://protobuf.kmcd.dev/advanced/)** pages contain two developer sandboxes:

* **Descriptor Playground:** Write protobuf schemas and inspect the compiled `FileDescriptorSet` JSON metadata structure in real time. You can download the output directly as a raw binary descriptor (`descriptor.bin`) or a JSON file.
* **Validation Lab:** An interactive sandbox illustrating the `protovalidate` specification. You can write schemas containing CEL (Common Expression Language) validation rules (like `this.age >= 18`) and test them against JSON payloads. If violations occur, the page displays a list of validation errors, naming the violating fields and the failed constraint rules.

### The Protobuf Ecosystem

This page contains a searchable index of protobuf-related projects, including compilers, linters, formatters, validators, RPC frameworks, and language-specific tooling. Whether you're looking for a code generator, validation framework, or language-specific library, the explorer makes it easier to discover tools across the broader protobuf ecosystem.

Explore the complete list at **[protobuf.kmcd.dev/ecosystem](https://protobuf.kmcd.dev/ecosystem/)**.

---

## Powered by WebAssembly (WASM)

Building interactive compilers in a web browser usually requires a round-trip to a backend server or rewriting everything in TypeScript. To avoid the server costs of passing custom schemas to a backend or the development time of porting several different libraries to TypeScript, this site runs Go's compiler tooling directly inside the browser using WebAssembly.

As I noted in [Zero-Friction Demos with WASM](/posts/wasm-demos/), compiling Go libraries into WASM works really well for client-side tools. The WASM binary handles the schema compilation, JSON-to-binary serialization, and Protoscope compilation natively on the client. It is fast, private, and works offline.

## Give it a Spin

If you want to look at the lower-level mechanics of Protocol Buffers, or if you just need a zero-install scratchpad to inspect wire formats and CEL validations, check out the site starting from the **[Introduction](https://protobuf.kmcd.dev/intro/)**:

👉 **[protobuf.kmcd.dev](https://protobuf.kmcd.dev)**

The source code and widgets are open source and available on GitHub at **[sudorandom/protobuf.kmcd.dev](https://github.com/sudorandom/protobuf.kmcd.dev)**. Please give the site a try and let me know your thoughts (especially on the binary section), bug reports, or suggestions!
