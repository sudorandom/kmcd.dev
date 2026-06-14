---
title: "Dynamic Protobuf in Go"
date: "2026-06-22T10:00:00Z"
categories: ["article"]
tags: ["protobuf", "go", "performance", "software-architecture"]
description: "How table-driven bytecode parsing makes dynamic Protocol Buffers fast in Go."
cover: "cover.svg"
images: ["/posts/dynamic-protobuf-in-go/cover.svg"]
featuredalt: ""
featuredpath: "date"
slug: "dynamic-protobuf-in-go"
type: "posts"
devtoSkip: true
draft: true
---

Statically compiled schemas are Protocol Buffers' primary optimization lever. By compiling `.proto` definitions into concrete, statically typed Go structures at build time, the Go Protobuf compiler generates highly optimized serialization and deserialization routines.

Without a known schema at build time, this optimization pipeline breaks down. Pipelines handling dynamic message descriptors at runtime (message registries, API event gateways, dynamic proxies, developer tools) cannot rely on code generation. They need to load schema descriptors on the fly and parse incoming binary payloads dynamically. 

Historically, this required Go's native reflection-based package, [`dynamicpb`](https://pkg.go.dev/google.golang.org/protobuf/types/dynamicpb). Standard `dynamicpb` is notoriously slow and allocation-heavy.

Here is a look at why reflection hurts dynamic parsing, how Buf's [`hyperpb`](https://github.com/bufbuild/hyperpb) bypasses it using table-driven bytecode compilation, and the real-world impact of switching [FauxRPC](https://fauxrpc.com) to use `hyperpb`.

---

## FauxRPC and hyperpb

FauxRPC generates mock data from arbitrary Protobuf schemas. It ingests schemas after startup. Users upload `.proto` schemas or descriptors dynamically, and FauxRPC instantly configures itself to parse, generate, and route gRPC mock requests.

Because of this runtime flexibility, FauxRPC historically relied on dynamicpb`. It worked fine for local development mocks. However, the overhead of reflection-heavy parsing on large payloads or fast mock workloads created a noticeable CPU bottleneck.

I recently switched FauxRPC to use `hyperpb` for dynamic schemas on `amd64` and `arm64` platforms, where the optimized bytecode engine is fully supported. Because `hyperpb` is strictly read-only and does not support any of the modification reflection APIs, FauxRPC uses it exclusively for reading in and parsing protobuf requests. Writing responses still uses standard [`dynamicpb`](https://github.com/search?q=repo%3Asudorandom%2Ffauxrpc%20dynamicpb&type=code) since those messages must be modified and populated dynamically. Even with this hybrid model, the switch drastically improved parsing performance and dropped allocation overhead. If `hyperpb` adds modification support, I'll probably be one of the first to adopt it because FauxRPC is the perfect use-case for this library.

If you're curious how both libraries are used in FauxRPC, you can see the search results for [dynamicpb](https://github.com/search?q=repo%3Asudorandom%2Ffauxrpc%20dynamicpb&type=code) and [hyperpb](https://github.com/search?q=repo%3Asudorandom%2Ffauxrpc+hyperpb&type=code).

---

## The Bottleneck: Reflection-Based dynamicpb

Go's standard `dynamicpb` package takes a `protoreflect.MessageDescriptor` and constructs a dynamic message representation at runtime. 

Since the layout is unknown at compile time, `dynamicpb` relies heavily on Go's reflection subsystem to inspect types, allocate objects, and map fields. This creates two distinct performance bottlenecks:

1. **Massive Allocation Overhead**: Deserializing a complex, nested binary payload dynamically means constructing a tree of runtime structures. Without compile-time type definitions, the Go runtime allocates heap pointers and interfaces for map entries, repeated fields, and nested messages. This causes severe garbage collector (GC) pressure.
2. **Pointer Chasing and Cache Misses**: Traversing a reflection-heavy object graph requires following layers of pointers. At the CPU level, this leads to frequent cache misses and degrades branch prediction compared to flat, contiguous memory access.

---

## Bytecode Compilation with hyperpb

Buf's `hyperpb` library takes a different approach. It compiles the message descriptor into dedicated, optimized table-driven parser bytecode at runtime.

The bytecode engine parses binary payloads directly, bypassing Go's reflection model entirely, and decodes field tags with speeds approaching statically generated code.

### Eliminating Heap Allocations

Execution speed is only half the equation; memory allocation dictates the rest.

`hyperpb` provides a thread-local, pre-allocated memory arena pool through `hyperpb.Shared`. Pairing bytecode parsing with a reusable memory arena allows you to recycle memory buffers across multiple requests. This eliminates runtime heap churn for read-only pipelines:

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

*Note: Because the data fields in `msg` are backed by the pre-allocated pool's memory arena, any references to those fields become invalid (and will read corrupted data or panic) after `shared.Free()` is called. The processing pipeline must handle the message completely synchronously (e.g., no asynchronous routing, lazy field reading, or passing to background goroutines) before the arena is recycled.*

---

## Dynamic Reflection in Practice

To see how `dynamicpb` and `hyperpb` compare in code, we can use the classic ConnectRPC/Buf [Eliza service](https://buf.build/connectrpc/eliza) schema. 

A complete, working set of examples is available in the [dynamic-protobuf-in-go/go](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/dynamic-protobuf-in-go/go) directory. It uses `buf` to compile the Protobuf definitions into a binary descriptor set, which is then loaded at runtime to perform dynamic serialization and reflection.

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

Because `hyperpb` is built for high-performance ingestion and routing, it only supports **read-only** access. Sunny mentioned Message descriptors must be compiled into optimized parser bytecode. Any attempt to write or mutate a message will panic:

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

By using the exact same standard `protoreflect` interface, `hyperpb` acts as a drop-in replacement for downstream read operations while executing significantly faster and with drastically fewer allocations. That's the claim, at least. So let's test it to see just how efficient hyperpb is compared to dynamicpb.

---

## Performance Evaluation

I benchmarked three dynamic parsing strategies against statically generated Go Protobuf code to measure the exact improvements:

| Variant | Description |
| :--- | :--- |
| **dynamicpb** | Evaluates dynamic descriptor parsing and reflection-based Protobuf handling using Go's standard [`dynamicpb`](https://pkg.go.dev/google.golang.org/protobuf/types/dynamicpb) package. |
| **hyperpb** | Evaluates dynamic parsing using Buf's table-driven [`hyperpb`](https://github.com/bufbuild/hyperpb) library. |
| **hyperpb + Shared** | Evaluates dynamic parsing using `hyperpb` paired with a thread-local, pre-allocated `hyperpb.Shared` memory arena to recycle allocations. |
| **Concrete (proto)** | Statically compiled Go Protobuf code (provided as a baseline comparison). |
| **Concrete (vtproto)** | Statically compiled, reflection-free PlanetScale [`vtproto`](https://github.com/planetscale/vtproto) code (provided as a baseline comparison). |

The benchmarks evaluate performance across three payload scales:
- **Small**: A flat message with 4 primitive fields (ID, status, age, score).
- **Medium**: A nested event message containing an actor object, tags, and a metadata map.
- **Large**: An array repeating the Medium event 100 times.

The source code and setup for these benchmarks are available in the [dynamic-protobuf-in-go/benchmarks](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/dynamic-protobuf-in-go/benchmarks) directory.

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
      "data": [762.3, 381.3, 150.7, 141.2, 31.57],
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
| **Concrete (vtproto)** | **31.6 ns** | **16 B** | **1** |
| **Concrete (proto)** | 141.2 ns | 96 B | 2 |
| **hyperpb + Shared** | 150.7 ns | 64 B | 1 |
| **hyperpb** | 381.3 ns | 799 B | 4 |
| **dynamicpb** | 762.3 ns | 616 B | 11 |

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
      "data": [2930, 697.1, 673.5, 369.4, 350.2],
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
| **hyperpb + Shared** | **350.2 ns** | **357 B** | **1** |
| **Concrete (vtproto)** | 369.4 ns | 432 B | 14 |
| **Concrete (proto)** | 673.5 ns | 560 B | 15 |
| **hyperpb** | 697.1 ns | 1,446 B | 5 |
| **dynamicpb** | 2,930.0 ns | 2,072 B | 43 |

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
      "data": [298918, 66722, 42896, 29197, 22074],
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
| **hyperpb + Shared** | **22,074 ns** | **21,838 B** | **1** |
| **hyperpb** | 29,197 ns | 59,999 B | 12 |
| **Concrete (vtproto)** | 42,896 ns | 58,168 B | 1,508 |
| **Concrete (proto)** | 66,722 ns | 58,232 B | 1,509 |
| **dynamicpb** | 298,918 ns | 205,753 B | 4,117 |

</details>
  {{< /tab >}}
{{< /tabs >}}

---

## Analysis: Speed and Memory Efficiency

The benchmark numbers reveal a massive divide, particularly as payload sizes increase.

### 1. Execution Speedup
On a **Large Payload**, standard reflection-based `dynamicpb` takes 298,918 ns. 
- Standard `hyperpb` executes in 29,197 ns (a 10x speedup).
- `hyperpb + Shared` executes in 22,074 ns (a 13.5x speedup).

Interestingly, both `hyperpb` configurations outperform compile-time generated static Protobuf code (`Concrete (proto)` at 66,722 ns and reflection-free `Concrete (vtproto)` at 42,896 ns). Standard static Protobuf still incurs the runtime cost of allocating individual heap pointers and structs for every nested sub-message in the list. The memory arena in `hyperpb + Shared` allocates this memory contiguously.

Notably, this crossover point—where dynamic parsing paired with a memory arena beats statically compiled generated code—occurs even at the **Medium Payload** scale. At that size, `hyperpb + Shared` (350.2 ns) already edges out reflection-free `Concrete (vtproto)` (369.4 ns). This inversion highlights just how substantial the CPU overhead of standard heap allocation tracking and garbage collection is in Go.

### 2. Allocation Elimination
The allocation statistics highlight the biggest architectural advantage. On a Large Payload:
- `dynamicpb`: 4,117 heap allocations per message.
- `Concrete (proto)`: 1,509 heap allocations.
- `hyperpb`: 12 heap allocations.
- `hyperpb + Shared`: 1 heap allocation. (This single allocation is the top-level message pointer escaping to the heap as a `proto.Message` interface wrapper. Because the standard `proto.Unmarshal` signature requires passing an interface, this top-level escape cannot easily be avoided and prevents hitting an absolute zero allocation count.)

Eliminating thousands of heap allocations per request removes the CPU bottleneck of garbage collection. For hot-path event routing or proxying services, this lowers tail latency (p99) and reduces CPU utilization under load.

---

## When to Use hyperpb

`hyperpb` is built for specific use cases. It is not a universal drop-in replacement for standard Go Protobuf code.

### Ideal Use Cases
- **Dynamic Gateways & Proxies**: Systems receiving dynamic schemas at runtime that must inspect or forward payloads without ahead-of-time code generation.
- **Developer Tooling**: Tools like FauxRPC that mock interfaces, fuzz test services, or interact dynamically with user-supplied schemas.
- **High-Throughput Pipelines**: Pipelines with dynamic schemas where reflection overhead is a bottleneck.

### Trade-offs and Constraints
- **Platform Specificity**: `hyperpb` relies on specialized runtime assembly/bytecode generators tailored for optimized CPU architectures. Its performance benefits currently apply to `amd64` and `arm64` platforms. It falls back to standard reflection on other architectures.
- **Read-Only vs Mutable**: Reusing buffers via `hyperpb.Shared` works best for read-only access pipelines. If you need to mutate the parsed message or pass it asynchronously to other goroutines, you must copy the data or avoid using the shared arena. This increases allocations, though still resulting in fewer than standard `dynamicpb`.
- **AOT Compilation**: If your schema is known at build time, compiling your Protobuf definitions remains the best approach. Static compilation (`vtproto` or standard `proto`) offers strict type safety and requires no runtime bytecode compilation overhead.

---

## Conclusion

Reflection-based parsing limits the performance of systems requiring runtime schema flexibility. `hyperpb` proves that dynamic parsing can be highly optimized. By compiling descriptors into optimized bytecode tables and utilizing thread-local memory arenas, `hyperpb` bridges the performance gap between dynamic reflection and static compilation. In high-throughput scenarios, its allocation efficiency can even outperform static generation.

For FauxRPC, adopting `hyperpb` was a pretty easy drop-in replacement. If you're doing dynamicpb, you might want to consider adopting hyperpb for the 'unmarshalling' part of your code.

---

## References & Further Reading

For a deeper dive into `hyperpb` and Go memory arenas, check out the following resources:

- **Official hyperpb Announcement:**
  - ["Introducing hyperpb: 10x faster dynamic Protobuf parsing that’s even 3x faster than generated code"](https://buf.build/blog/hyperpb) on the *Buf Blog*.
- **Deep Dives by Sunny (mcyoung):**
  - ["Parsing Protobuf Like Never Before"](https://mcyoung.xyz/2025/07/16/hyperpb/) (the internals of `hyperpb`'s bytecode compiler design).
  - ["Cheating the Reaper in Go"](https://mcyoung.xyz/2025/04/21/go-arenas/) (a deep dive into the design and tradeoffs of memory arenas in Go).
