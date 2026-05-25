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

When you lack the schema at build time, things get complicated. Pipelines handling dynamic message descriptors at runtime (such as message registries, API event gateways, dynamic proxies, or developer tools) cannot rely on code generation. They need to load schema descriptors on the fly and parse incoming binary payloads dynamically. 

Historically, this required Go's native reflection-based package, [`dynamicpb`](https://pkg.go.dev/google.golang.org/protobuf/types/dynamicpb). Standard `dynamicpb` is notoriously slow and allocation-heavy.

This post examines why reflection hurts dynamic parsing, how Buf's [`hyperpb`](https://github.com/bufbuild/hyperpb) bypasses reflection using table-driven bytecode compilation, and the real-world impact of switching [FauxRPC](https://fauxrpc.com) to use `hyperpb`.

---

## FauxRPC and hyperpb

FauxRPC generates mock data from arbitrary Protobuf schemas. It ingests schemas after startup. Users upload `.proto` schemas or descriptors dynamically, and FauxRPC instantly configures itself to parse, generate, and route gRPC mock requests.

Because of this runtime flexibility, FauxRPC historically relied on `dynamicpb`. It worked fine for local development mocks, but the overhead of reflection-heavy parsing on large payloads or fast mock workloads created a noticeable CPU bottleneck.

I have switched FauxRPC to use `hyperpb` for dynamic schemas specifically on `amd64` and `arm64` platforms, where the optimized bytecode engine is fully supported. The switch dramatically improved parsing performance and reduced allocation overhead.

---

## The Bottleneck: Reflection-Based dynamicpb

Go's standard `dynamicpb` package takes a `protoreflect.MessageDescriptor` and constructs a dynamic message representation at runtime. 

Since the layout is unknown at compile time, `dynamicpb` relies heavily on Go's reflection subsystem to inspect types, allocate objects, and map fields. This creates two major performance bottlenecks:

1. **Massive Allocation Overhead**: Deserializing a complex, nested binary payload dynamically means constructing a tree of runtime structures. Without compile-time type definitions, the Go runtime allocates heap pointers and interfaces for map entries, repeated fields, and nested messages. This causes significant garbage collector (GC) pressure.
2. **Pointer Chasing and Cache Misses**: Traversing a reflection-heavy object graph requires following layers of pointers. At the CPU level, this leads to frequent cache misses and degrades branch prediction compared to flat, contiguous memory access.

---

## Bytecode Compilation with hyperpb

Buf's `hyperpb` library takes a different approach. It compiles the message descriptor into dedicated, optimized table-driven parser bytecode at runtime.

The bytecode engine parses binary payloads directly, bypassing Go's reflection model, and decodes field tags with speeds approaching statically generated code.

### Eliminating Heap Allocations

CPU cycle efficiency only solves part of the problem. Memory allocation is the other major factor. 

`hyperpb` provides a thread-local, pre-allocated memory arena pool through `hyperpb.Shared`. Pairing bytecode parsing with a reusable memory arena allows you to recycle memory buffers across multiple requests. This eliminates runtime heap churn for read-only pipelines:

```go
shared := new(hyperpb.Shared) // Instantiated once per goroutine/worker

for _, payload := range incoming {
    // Reuses the underlying pre-allocated memory arena
    msg := shared.NewMessage(mType) 
    _ = proto.Unmarshal(payload, msg)
    
    route(msg)    // Note: Must be handled synchronously
    shared.Free() // Recycles the arena back to the pool
}
```

*Note: Passing `msg` to a background goroutine before calling `shared.Free()` will cause data corruption. The pipeline must be synchronous.*

---

## Performance Evaluation

To verify the performance improvements, I evaluated three dynamic parsing strategies against statically generated Go Protobuf code:

| Variant | Format | Description |
| :--- | :---: | :--- |
| **dynamicpb** | Protobuf | Evaluates dynamic descriptor parsing and reflection-based Protobuf handling using Go's standard [`dynamicpb`](https://pkg.go.dev/google.golang.org/protobuf/types/dynamicpb) package. |
| **hyperpb** | Protobuf | Evaluates dynamic parsing using Buf's table-driven [`hyperpb`](https://github.com/bufbuild/hyperpb) library. |
| **hyperpb + Shared** | Protobuf | Evaluates dynamic parsing using `hyperpb` paired with a thread-local, pre-allocated `hyperpb.Shared` memory arena to recycle allocations. |
| **Concrete (proto)** | Protobuf | Statically compiled Go Protobuf code (provided as a baseline comparison). |
| **Concrete (vtproto)** | Protobuf | Statically compiled, reflection-free PlanetScale [`vtproto`](https://github.com/planetscale/vtproto) code (provided as a baseline comparison). |

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

The performance difference scales drastically with payload size.

### 1. Execution Speedup
On a **Large Payload**, standard reflection-based `dynamicpb` takes 298,918 ns. 
- Standard `hyperpb` executes in 29,197 ns (a 10x speedup).
- `hyperpb + Shared` executes in 22,074 ns (a 13.5x speedup).

Both `hyperpb` configurations outperform compile-time generated static Protobuf code (`Concrete (proto)` at 66,722 ns and reflection-free `Concrete (vtproto)` at 42,896 ns). Standard static Protobuf still incurs the runtime cost of allocating individual heap pointers and structs for every nested sub-message in the list. The memory arena in `hyperpb + Shared` flattens these allocations contiguously.

### 2. Allocation Elimination
The allocation statistics highlight the biggest architectural advantage. On a Large Payload:
- `dynamicpb`: 4,117 heap allocations per message.
- `Concrete (proto)`: 1,509 heap allocations.
- `hyperpb`: 12 heap allocations.
- `hyperpb + Shared`: 1 heap allocation. (This single allocation is typically just the top-level message struct wrapper returned by the pool).

Eliminating thousands of heap allocations per request removes the CPU bottleneck of garbage collection. For hot-path event routing or proxying services, this lowers tail latency (p99) and reduces CPU utilization under load.

---

## When to Use hyperpb

`hyperpb` solves a specific set of problems. It is not a generic drop-in replacement for all Go Protobuf code.

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

Standard reflection-based parsing (`dynamicpb`) creates a significant performance bottleneck for systems requiring runtime schema flexibility. `hyperpb` demonstrates an effective alternative. By compiling descriptors into optimized bytecode tables and utilizing thread-local memory arenas, `hyperpb` bridges the performance gap between dynamic reflection and static compilation. In high-throughput scenarios, its allocation efficiency can even outperform static generation.

For FauxRPC, adopting `hyperpb` transformed dynamic schema mock generation from a latency-prone bottleneck into a highly efficient, allocation-free pipeline.

---

## References & Further Reading

For a deeper dive into `hyperpb` and Go memory arenas, check out the following resources:

- **Official hyperpb Announcement:**
  - ["Introducing hyperpb: 10x faster dynamic Protobuf parsing that’s even 3x faster than generated code"](https://buf.build/blog/hyperpb) on the *Buf Blog*.
- **Deep Dives by Sunny (mcyoung):**
  - ["Parsing Protobuf Like Never Before"](https://mcyoung.xyz/2025/07/16/hyperpb/) (the internals of `hyperpb`'s bytecode compiler design).
  - ["Cheating the Reaper in Go"](https://mcyoung.xyz/2025/04/21/go-arenas/) (a deep dive into the design and tradeoffs of memory arenas in Go).
