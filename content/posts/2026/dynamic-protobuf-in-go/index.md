---
title: "Making Dynamic Protobuf Fast in Go"
date: "2026-07-07T10:00:00Z"
categories: ["article"]
tags: ["protobuf", "go", "performance", "software-architecture"]
description: "Using Buf’s hyperpb to speed up FauxRPC’s runtime-loaded Protobuf request path."
cover: "cover.svg"
images: ["/posts/dynamic-protobuf-in-go/cover.svg"]
featuredalt: ""
featuredpath: "date"
slug: "dynamic-protobuf-in-go"
type: "posts"
devtoSkip: true
---

Most Go Protobuf services get to cheat: their schemas are known at build time. [`protoc-gen-go`](https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go) turns those schemas into concrete message types, accessor methods, and runtime metadata that the protobuf runtime can use efficiently.

[FauxRPC](https://fauxrpc.com) does not get that luxury. It loads user-provided schemas at runtime so it can mock arbitrary gRPC, gRPC-Web, and Connect services without asking users to install `protoc`, Buf, or a Go toolchain first. Without generated Go types for each schema, it has to parse and inspect payloads dynamically.

That flexibility pushed the request path toward Go's standard [`dynamicpb`](https://pkg.go.dev/google.golang.org/protobuf/types/dynamicpb) package. `dynamicpb` is flexible, but it pays for that flexibility with extra allocations, descriptor lookups, and reflection-heavy access paths.

Buf introduced [`hyperpb`](https://github.com/bufbuild/hyperpb), a dynamic Protobuf parser that compiles descriptors into optimized parser bytecode at runtime. I wanted to see whether it could make FauxRPC's read path faster without giving up runtime-loaded schemas.

The short version: yes, with caveats. `hyperpb` is read-only, so FauxRPC still uses `dynamicpb` to build mock responses. But for parsing incoming requests, the performance difference was large enough to be worth the split.

---

## Reflection-Based dynamicpb

Go's standard `dynamicpb` package takes a `protoreflect.MessageDescriptor` and constructs a dynamic message representation at runtime.

Since message layouts are only known at runtime, `dynamicpb` must route field access through descriptors and generic message representations rather than generated message types and runtime metadata. This introduces additional indirection and dynamic dispatch. The cost mostly shows up in two places:

1. **It allocates a lot.** Nested messages, repeated fields, map entries, and interface values turn into a pile of heap objects. This causes severe garbage collector (GC) pressure.
2. **It chases pointers.** Once the decoded message is spread across many small objects, the CPU spends more time bouncing around memory instead of reading predictable, contiguous data.

---

## Bytecode Compilation with hyperpb

Buf's `hyperpb` library takes a different approach. It compiles the message descriptor into dedicated, optimized table-driven parser bytecode at runtime.

The parser avoids Go’s reflection-heavy dynamic message construction in the unmarshalling hot path, while still exposing the parsed result through the standard `protoreflect` APIs. In many cases, it can parse dynamic payloads at speeds close to, or even faster than, generated Go protobuf messages.

### Pre-Compiling at Runtime

Because `hyperpb` uses a custom VM under the hood, it requires a compilation phase before you can parse any payloads. Similar to compiling a regular expression with Go's `regexp.Compile`, you must compile the schema definition at runtime. `hyperpb.CompileFileDescriptorSet` compiles a specific message type out of a `FileDescriptorSet`, while `hyperpb.CompileMessageDescriptor` compiles an already-resolved message descriptor:

```go
// Done once at startup/initialization
hyperMsgType := hyperpb.CompileMessageDescriptor(messageDesc)
```

This compilation cost is paid once per message type, so the compiled types should be cached and reused.

### Moving Allocation Out of the Hot Path

The real story is not just speed. It is allocation behavior.

`hyperpb` provides a reusable, pre-allocated memory arena pool through `hyperpb.Shared`. Pairing bytecode parsing with a reusable memory arena allows you to recycle memory buffers across multiple requests. This removes most of the per-message heap churn for read-only pipelines:

```go
shared := new(hyperpb.Shared) // Instantiated once per goroutine/worker

for _, payload := range incoming {
    // Reuses the underlying pre-allocated memory arena
    msg := shared.NewMessage(mType)
    _ = proto.Unmarshal(payload, msg)

    route(msg)    // Note: Must be handled synchronously and complete before Free()
    shared.Free() // Recycles the arena back to the pool
}
```

{{% warning-box %}}
Because the data fields in `msg` are backed by the pre-allocated pool's memory arena, any references to those fields become invalid (and will read corrupted data or panic) after `shared.Free()` is called. The processing pipeline must handle the message completely synchronously (e.g., no asynchronous routing, lazy field reading, or passing to background goroutines) before the arena is recycled.
{{% /warning-box %}}

---

## Dynamic Reflection in Practice

To see how `dynamicpb` and `hyperpb` compare in code, we can use the classic ConnectRPC/Buf [Eliza service](https://buf.build/connectrpc/eliza) schema.

A complete set of runnable examples is available in the [dynamic-protobuf-in-go/go](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/dynamic-protobuf-in-go/go) directory. It uses `buf` to compile the Protobuf definitions into a binary descriptor set, which is then loaded at runtime to perform dynamic serialization and reflection.

### 1. Compiling Protobuf Descriptors with Buf

Before using dynamic messages, we must compile the `.proto` schema into a `FileDescriptorSet` (a serialized binary image of the schemas). Using the Buf CLI, this is done with a single command:

```bash
buf build -o eliza.binpb
```

### 2. Loading Descriptors at Runtime

In Go, we read this descriptor set, unmarshal it into a `descriptorpb.FileDescriptorSet`, and load it into a `protoregistry.Files` registry:

{{% render-code file="go/main.go" language="go" start="// start: register" end="// end: register" %}}

Once registered, we can look up the message descriptor by its full name and locate individual fields dynamically:

{{% render-code file="go/main.go" language="go" start="// start: lookup" end="// end: lookup" %}}

### 3. Dynamic Access with dynamicpb

Standard `dynamicpb` creates dynamic messages that support both reading and writing field values. The standard `protoreflect` interface is used for field access:

{{% render-code file="go/dynamicpb.go" language="go" start="// start: dynamicpb" end="// end: dynamicpb" %}}

### 4. High-Performance Read-Only Access with hyperpb

Because `hyperpb` is built for high-performance ingestion and routing, it only supports **read-only** access. Message descriptors must be compiled into optimized parser bytecode, and any attempt to write or mutate a message will panic:

{{% render-code file="go/hyperpb.go" language="go" start="// start: hyperpb" end="// end: hyperpb" %}}

When running these examples (with `go run .`), we get the following output, verifying that both implementations decode the reflection values identically:

```text
--- Step 1: dynamicpb (Standard Go Reflection) ---
Serialized bytes: 0a1948656c6c6f20456c697a612c20686f772061726520796f753f
Decoded message: Hello Eliza, how are you?

--- Step 2: hyperpb (Table-Driven Bytecode VM) ---
Decoded message: Hello Eliza, how are you?

--- Step 3: hyperpb + Shared (Memory Reuse Arena) ---
Decoded message: Hello Eliza, how are you?
Memory arena recycled.
```

By using the exact same standard `protoreflect` interface, `hyperpb` acts as a drop-in replacement for downstream read operations while executing faster and allocating much less in these benchmarks. So I wanted to check whether this held up in my own tiny benchmark goblin cave.

---

## Performance Evaluation

I benchmarked three dynamic parsing strategies against statically generated Go Protobuf code to measure the difference in this setup:

| Variant | Description |
| :--- | :--- |
| **dynamicpb** | Evaluates dynamic descriptor parsing and reflection-based Protobuf handling using Go's standard [`dynamicpb`](https://pkg.go.dev/google.golang.org/protobuf/types/dynamicpb) package. |
| **hyperpb** | Evaluates dynamic parsing using Buf's table-driven [`hyperpb`](https://github.com/bufbuild/hyperpb) library. |
| **hyperpb + Shared** | Evaluates dynamic parsing using `hyperpb` paired with a reusable `hyperpb.Shared` memory arena to recycle allocations. |
| **Concrete (proto)** | Statically compiled Go Protobuf code (provided as a baseline comparison). |
| **Concrete (vtproto)** | Statically compiled, reflection-free PlanetScale [`vtproto`](https://github.com/planetscale/vtproto) code (provided as a baseline comparison). |

The benchmarks evaluate performance across three payload scales:
- **Small**: A flat message with 4 primitive fields (ID, status, age, score).
- **Medium**: A nested event message containing an actor object, tags, and a metadata map.
- **Large**: An array repeating the Medium event 100 times.

The source code and setup for these benchmarks are available in the [dynamic-protobuf-in-go/benchmarks](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/dynamic-protobuf-in-go/benchmarks) directory.

*All benchmarks were executed on an Apple M1 Pro (`darwin/arm64`) using Go 1.26. Descriptor compilation was excluded from timing and performed once during benchmark initialization. Measurements represent deserialization only (`proto.Unmarshal`) and were collected using `go test -bench=. -benchmem`.*

### Benchmark Results

{{< tabs >}}
  {{< tab name="Small Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "dynamicpb",
      "hyperpb",
      "hyperpb + Shared",
      "Concrete (proto)",
      "Concrete (vtproto)"
    ],
    "datasets": [{
      "label": "ns/op",
      "data": [622, 334, 120, 113, 25],
      "backgroundColor": [
        "rgba(148, 163, 184, 0.75)",
        "rgba(168, 85, 247, 0.75)",
        "rgba(168, 85, 247, 0.75)",
        "rgba(0, 191, 255, 0.75)",
        "rgba(0, 191, 255, 0.75)"
      ],
      "borderColor": [
        "rgba(148, 163, 184, 1)",
        "rgba(168, 85, 247, 1)",
        "rgba(168, 85, 247, 1)",
        "rgba(0, 191, 255, 1)",
        "rgba(0, 191, 255, 1)"
      ],
      "borderWidth": 1
    }]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Dynamic Parsing Performance (Small Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "dynamicpb", "color": "rgba(148, 163, 184, 0.75)" },
          { "text": "hyperpb", "color": "rgba(168, 85, 247, 0.75)" },
          { "text": "protobuf", "color": "rgba(0, 191, 255, 0.75)" }
        ]
      }
    },
    "scales": {
      "x": {
        "type": "linear",
        "min": 0,
        "ticks": { "color": "#fff" }
      },
      "y": {
        "ticks": { "color": "#fff" }
      }
    }
  }
}
{{< /chart >}}

<details>
<summary><b>Show data table</b></summary>

| Benchmark (Small Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **Concrete (vtproto)** | **25 ns** | **16 B** | **1** |
| **Concrete (proto)** | 113 ns | 96 B | 2 |
| **hyperpb + Shared** | 120 ns | 65 B | 1 |
| **hyperpb** | 334 ns | 798 B | 4 |
| **dynamicpb** | 622 ns | 616 B | 11 |

</details>
  {{< /tab >}}
  {{< tab name="Medium Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "dynamicpb",
      "hyperpb",
      "Concrete (proto)",
      "Concrete (vtproto)",
      "hyperpb + Shared"
    ],
    "datasets": [{
      "label": "ns/op",
      "data": [2368, 600, 564, 306, 286],
      "backgroundColor": [
        "rgba(148, 163, 184, 0.75)",
        "rgba(168, 85, 247, 0.75)",
        "rgba(0, 191, 255, 0.75)",
        "rgba(0, 191, 255, 0.75)",
        "rgba(168, 85, 247, 0.75)"
      ],
      "borderColor": [
        "rgba(148, 163, 184, 1)",
        "rgba(168, 85, 247, 1)",
        "rgba(0, 191, 255, 1)",
        "rgba(0, 191, 255, 1)",
        "rgba(168, 85, 247, 1)"
      ],
      "borderWidth": 1
    }]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Dynamic Parsing Performance (Medium Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "dynamicpb", "color": "rgba(148, 163, 184, 0.75)" },
          { "text": "hyperpb", "color": "rgba(168, 85, 247, 0.75)" },
          { "text": "protobuf", "color": "rgba(0, 191, 255, 0.75)" }
        ]
      }
    },
    "scales": {
      "x": {
        "type": "linear",
        "min": 0,
        "ticks": { "color": "#fff" }
      },
      "y": {
        "ticks": { "color": "#fff" }
      }
    }
  }
}
{{< /chart >}}

<details>
<summary><b>Show data table</b></summary>

| Benchmark (Medium Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **hyperpb + Shared** | **286 ns** | **356 B** | **1** |
| **Concrete (vtproto)** | 306 ns | 432 B | 14 |
| **Concrete (proto)** | 564 ns | 560 B | 15 |
| **hyperpb** | 600 ns | 1,444 B | 5 |
| **dynamicpb** | 2,368 ns | 2,072 B | 43 |

</details>
  {{< /tab >}}
  {{< tab name="Large Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "dynamicpb",
      "Concrete (proto)",
      "Concrete (vtproto)",
      "hyperpb",
      "hyperpb + Shared"
    ],
    "datasets": [{
      "label": "ns/op",
      "data": [241009, 54078, 35350, 24664, 17967],
      "backgroundColor": [
        "rgba(148, 163, 184, 0.75)",
        "rgba(0, 191, 255, 0.75)",
        "rgba(0, 191, 255, 0.75)",
        "rgba(168, 85, 247, 0.75)",
        "rgba(168, 85, 247, 0.75)"
      ],
      "borderColor": [
        "rgba(148, 163, 184, 1)",
        "rgba(0, 191, 255, 1)",
        "rgba(0, 191, 255, 1)",
        "rgba(168, 85, 247, 1)",
        "rgba(168, 85, 247, 1)"
      ],
      "borderWidth": 1
    }]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Dynamic Parsing Performance (Large Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "dynamicpb", "color": "rgba(148, 163, 184, 0.75)" },
          { "text": "hyperpb", "color": "rgba(168, 85, 247, 0.75)" },
          { "text": "protobuf", "color": "rgba(0, 191, 255, 0.75)" }
        ]
      }
    },
    "scales": {
      "x": {
        "type": "linear",
        "min": 0,
        "ticks": { "color": "#fff" }
      },
      "y": {
        "ticks": { "color": "#fff" }
      }
    }
  }
}
{{< /chart >}}

<details>
<summary><b>Show data table</b></summary>

| Benchmark (Large Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **hyperpb + Shared** | **17,967 ns** | **21,848 B** | **1** |
| **hyperpb** | 24,664 ns | 60,013 B | 12 |
| **Concrete (vtproto)** | 35,350 ns | 58,168 B | 1,508 |
| **Concrete (proto)** | 54,078 ns | 58,232 B | 1,509 |
| **dynamicpb** | 241,009 ns | 205,753 B | 4,117 |

</details>
  {{< /tab >}}
{{< /tabs >}}

---

## Analysis: Speed and Memory Efficiency

The numbers get more interesting as the payloads get larger.

### 1. Execution Speedup
On a **Large Payload**, reflection-based `dynamicpb` takes 241,009 ns.
- Standard `hyperpb` executes in 24,664 ns (a 9.7x speedup).
- `hyperpb + Shared` executes in 17,967 ns (a 13.4x speedup).

*Note: These benchmarks measure parsing and deserialization costs only. Work performed after unmarshalling (such as downstream data manipulation or field access) is excluded.*

Interestingly, both `hyperpb` configurations outperform compile-time generated static Protobuf code (`Concrete (proto)` at 54,078 ns and reflection-free `Concrete (vtproto)` at 35,350 ns). That is the part that surprised me. It does not mean `hyperpb`'s parser engine is inherently faster than generated Go code. Rather, the combination of bytecode parsing and arena-backed allocation reduces object creation costs for large nested payloads. Standard generated Protobuf still allocates individual heap objects for the nested sub-messages in this benchmark. The memory arena in `hyperpb + Shared` allocates this memory contiguously.

In these benchmarks, this crossover point, where dynamic parsing paired with a memory arena beats statically compiled generated code, occurs even at the **Medium Payload** scale. At that size, `hyperpb + Shared` (286 ns) already edges out reflection-free `Concrete (vtproto)` (306 ns).

### 2. Reducing Heap Allocations
The allocation statistics highlight the biggest architectural advantage. On a Large Payload:
- `dynamicpb`: 4,117 heap allocations per message.
- `Concrete (proto)`: 1,509 heap allocations.
- `hyperpb`: 12 heap allocations.
- `hyperpb + Shared`: 1 heap allocation. (The remaining allocation appears to come from the top-level message pointer escaping to the heap as a `proto.Message` interface wrapper. Because the standard `proto.Unmarshal` signature requires passing an interface, this top-level escape cannot easily be avoided and prevents hitting an absolute zero allocation count.)

For hot-path event routing or proxying services, fewer allocations should translate into less GC pressure, which can help with CPU usage and tail latency under real load.

---

## When to Use hyperpb

`hyperpb` is built for specific use cases. It is not a universal drop-in replacement for standard Go Protobuf code.

### Ideal Use Cases
- **Dynamic Gateways & Proxies**: Systems receiving dynamic schemas at runtime that must inspect or forward payloads without ahead-of-time code generation.
- **Developer Tooling**: Tools like FauxRPC that mock interfaces, fuzz test services, or interact dynamically with user-supplied schemas.
- **High-Throughput Pipelines**: Pipelines with dynamic schemas where reflection overhead is a bottleneck.

### Trade-offs and Constraints
- **Experimental API**: `hyperpb` is still pre-v1, so I would avoid wrapping it deeply into public APIs without a small compatibility layer.
- **Platform Specificity**: `hyperpb` relies on specialized runtime assembly and bytecode generators tailored for 64-bit little-endian architectures. It is officially supported only on `amd64` and `arm64` platforms. Compiling for other architectures requires the manual build tag `hyperpb.unsupported`, which compiles a slower generic parser backend.
- **Read-Only vs Mutable**: Reusing buffers via `hyperpb.Shared` works best for read-only access pipelines. If you need to mutate the parsed message or pass it asynchronously to other goroutines, you must copy the data or avoid using the shared arena. This increases allocations, though still resulting in fewer than standard `dynamicpb`.
- **AOT Compilation**: If your schema is known at build time, compiling your Protobuf definitions remains the best approach. Static compilation (`vtproto` or standard `proto`) offers strict type safety and requires no runtime bytecode compilation overhead.

---

## Conclusion

For FauxRPC, the interesting part of `hyperpb` is that it offers a way to speed up the read-heavy parts of dynamic schema handling.

Even with only the request path changed, the difference was not subtle. FauxRPC still uses `dynamicpb` where it needs to build and mutate response messages. But for request parsing, where payloads are read-only, `hyperpb` is a nice optimization. The benchmarks show a clear difference: fewer heap objects, less GC pressure, and faster unmarshalling on larger payloads.

If your pipeline fits those constraints, `hyperpb` is worth trying.

---

## References & Further Reading

For a deeper dive into `hyperpb` and Go memory arenas, check out the following resources:

- **Official hyperpb Announcement:**
  - ["Introducing hyperpb: 10x faster dynamic Protobuf parsing that’s even 3x faster than generated code"](https://buf.build/blog/hyperpb) on the *Buf Blog*.
- **Deep Dives by Sunny (mcyoung):**
  - ["Parsing Protobuf Like Never Before"](https://mcyoung.xyz/2025/07/16/hyperpb/) (the internals of `hyperpb`'s bytecode compiler design).
  - ["Cheating the Reaper in Go"](https://mcyoung.xyz/2025/04/21/go-arenas/) (a deep dive into the design and tradeoffs of memory arenas in Go).
