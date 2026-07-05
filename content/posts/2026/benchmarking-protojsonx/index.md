---
title: "Speeding Up ProtoJSON: Benchmarking protojsonx"
date: "2026-07-21T15:20:00Z"
categories: ["article"]
tags: ["protobuf", "go", "performance", "json", "protojsonx"]
description: "How protojsonx compares to standard protojson, JSON, and vtproto."
cover: "cover.jpg"
images: ["/posts/benchmarking-protojsonx/cover.jpg"]
featuredalt: ""
featuredpath: "date"
slug: "benchmarking-protojsonx"
type: "posts"
devtoSkip: true
draft: true
---

While modern Go frameworks like ConnectRPC handle browser-facing APIs natively, ProtoJSON remains a vital tool for debugging—particularly when frontend teams need human-readable logs, inspection tools, or mock JSON payloads to interact with Protobuf schemas.

The downside is performance. Go's official `protojson` package does a lot of work at runtime: walking descriptors, handling Protobuf-specific JSON mapping rules, resolving dynamic types, and allocating intermediate values. Compared with binary Protobuf, or even ordinary `encoding/json` on static Go structs, that overhead can be surprisingly large.

I built [`protojsonx`](https://github.com/sudorandom/protojsonx) to see how much of that cost could be removed without changing the Protobuf JSON format. The basic idea is to do more work up front, compile message layouts into faster lookup tables, and avoid repeated reflection-heavy traversal on every marshal or unmarshal call.

In this article, I’ll run a benchmark suite against `protojsonx` to see how it compares with standard `protojson`, raw JSON, and standard binary Protobuf (`proto` and reflection-free `vtproto`).

{{< github-repo repo="sudorandom/protojsonx" description="An experimental faster ProtoJSON encoder and decoder for Go." >}}

> **Warning:** `protojsonx` is highly experimental at this stage. It does pass the official Protobuf conformance tests, which gives me confidence that it follows the expected ProtoJSON behavior, but I would still treat it as early-stage software.

---

## What is `protojsonx`?

`protojsonx` is designed as a drop-in replacement for Go's official `google.golang.org/protobuf/encoding/protojson` library. To make that faster, it implements two key optimization strategies:

*   **Table-Driven Parser (Library Fallback)**: Rather than walking message descriptors dynamically via Go's reflection API on every call, `protojsonx` inspects them once and compiles them into flat, sequential layout tables. This allows it to marshal and unmarshal structures using fast, sequential offset arithmetic, avoiding per-call descriptor traversal for static messages.
*   **Statically Generated Plugin (`protoc-gen-go-protojsonx`)**: For the absolute maximum performance, `protojsonx` provides a protoc plugin that generates type-specific marshaling and unmarshaling methods directly. This completely bypasses runtime table lookups, tag parsing, and runtime type assertions.

### Operating Modes

To evaluate the performance of these strategies, I ran `protojsonx` in two distinct configurations:
1. **Library Mode**: The drop-in dynamic fallback using `protojsonx.Marshal(...)` / `protojsonx.Unmarshal(...)` on standard protobuf messages (which compiles and walks offset tables at runtime).
2. **Plugin Mode**: The static compiled path using `protojsonx.Marshal(...)` / `protojsonx.Unmarshal(...)` on messages compiled with `protoc-gen-go-protojsonx` (which automatically delegates to the generated type-specific methods).

---

## The Benchmark Setup

I used the benchmark suite from my previous article, running on Go 1.26 on an Apple M1 Pro. The configurations are:
* **Small:** A flat object with 4 fields (string ID, status boolean, age integer, score float).
* **Medium:** A nested user signup event containing actor object, string tags, and metadata map.
* **Large:** An array repeating the Medium object 100 times.

I grouped the results directly by data representation (**Static Message**), showing the serialization formats side-by-side.

---

## Marshaling Performance

Marshaling is the process of serializing Go values into bytes. The charts below group the formats (Binary, Standard protojson, and `protojsonx`) together for each data representation.

{{< tabs >}}
  {{< tab name="Small Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Binary (proto)",
      "Standard protojson",
      "protojsonx (Lib)",
      "protojsonx (Plugin)"
    ],
    "datasets": [
      {
        "label": "Small Marshal (ns/op)",
        "data": [82, 648, 146, 105],
        "backgroundColor": [
          "rgba(50, 205, 50, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)"
        ],
        "borderColor": [
          "rgba(50, 205, 50, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)"
        ],
        "borderWidth": 1
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Marshaling Performance (Small Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "display": false
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
<summary><b>Show complete data table</b></summary>

| Format / Config (Small Payload) | ns/op | Memory (B/op) | Allocations/op | Speed vs Standard protojson |
| :--- | :---: | :---: | :---: | :---: |
| **Concrete (vtproto)** | 25 ns | 32 B | 1 | - |
| **Concrete (proto)** | 82 ns | 32 B | 1 | - |
| **Concrete (JSON)** | 170 ns | 64 B | 1 | - |
| **Concrete (protojsonx - Plugin)** | **105 ns** | **64 B** | **1** | **6.2x faster** |
| **Concrete (protojsonx - Lib)** | **146 ns** | **64 B** | **1** | **4.4x faster** |
| **Concrete (protojson - standard)** | 648 ns | 512 B | 12 | 1.0x (Baseline) |

</details>
  {{< /tab >}}
  {{< tab name="Medium Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Binary (proto)",
      "Standard protojson",
      "protojsonx (Lib)",
      "protojsonx (Plugin)"
    ],
    "datasets": [
      {
        "label": "Medium Marshal (ns/op)",
        "data": [331, 2352, 367, 316],
        "backgroundColor": [
          "rgba(50, 205, 50, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)"
        ],
        "borderColor": [
          "rgba(50, 205, 50, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)"
        ],
        "borderWidth": 1
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Marshaling Performance (Medium Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "display": false
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
<summary><b>Show complete data table</b></summary>

| Format / Config (Medium Payload) | ns/op | Memory (B/op) | Allocations/op | Speed vs Standard protojson |
| :--- | :---: | :---: | :---: | :---: |
| **Concrete (vtproto)** | 115 ns | 176 B | 1 | - |
| **Concrete (proto)** | 331 ns | 176 B | 1 | - |
| **Concrete (JSON)** | 543 ns | 464 B | 2 | - |
| **Concrete (protojsonx - Plugin)** | **316 ns** | **320 B** | **1** | **7.4x faster** |
| **Concrete (protojsonx - Lib)** | **367 ns** | **320 B** | **1** | **6.4x faster** |
| **Concrete (protojson - standard)** | 2,352 ns | 1,721 B | 34 | 1.0x (Baseline) |

</details>
  {{< /tab >}}
  {{< tab name="Large Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Binary (proto)",
      "Standard protojson",
      "protojsonx (Lib)",
      "protojsonx (Plugin)"
    ],
    "datasets": [
      {
        "label": "Large Marshal (ns/op)",
        "data": [25865, 238838, 31854, 28456],
        "backgroundColor": [
          "rgba(50, 205, 50, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)"
        ],
        "borderColor": [
          "rgba(50, 205, 50, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)"
        ],
        "borderWidth": 1
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Marshaling Performance (Large Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "display": false
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
<summary><b>Show complete data table</b></summary>

| Format / Config (Large Payload) | ns/op | Memory (B/op) | Allocations/op | Speed vs Standard protojson |
| :--- | :---: | :---: | :---: | :---: |
| **Concrete (vtproto)** | 8,095 ns | 18,432 B | 1 | - |
| **Concrete (proto)** | 25,865 ns | 18,432 B | 1 | - |
| **Concrete (JSON)** | 43,933 ns | 32,829 B | 2 | - |
| **Concrete (protojsonx - Plugin)** | **28,456 ns** | **32,787 B** | **1** | **8.4x faster** |
| **Concrete (protojsonx - Lib)** | **31,854 ns** | **32,803 B** | **1** | **7.5x faster** |
| **Concrete (protojson - standard)** | 238,838 ns | 243,736 B | 2,728 | 1.0x (Baseline) |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Takeaways: Marshaling

*   **Descriptor compilation does most of the work**: Standard `protojson` relies on walking Protobuf reflection APIs on the fly, causing a stream of dynamic allocations. By pre-compiling message layout tables at startup, `protojsonx` avoids per-call descriptor traversal for static messages, completing static serialization in a fraction of the time with minimal allocations.
*   **Statically generated plugin pushes it further**: Code generation via `protoc-gen-go-protojsonx` completely bypasses runtime table lookups, dynamic type checks, and loop overhead, pushing marshaling speed to the absolute limit (e.g. Small marshaling goes from 146 ns/op with the dynamic table layout down to 105 ns/op with the plugin).
*   **Schema-guided JSON can beat generic JSON**: Because `protojsonx` is built on an optimized, single-allocation serialization pipeline, it outperforms Go's standard `encoding/json` library in these static-message benchmarks, proving that schema-guided serialization can beat generic JSON serialization for these schemas.

---

## Unmarshaling Performance

Unmarshaling parses incoming bytes back into Go values. In the benchmarks below, I present `protojsonx` in both its dynamic **Library Mode** and its statically compiled **Plugin Mode**.

{{< tabs >}}
  {{< tab name="Small Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Binary (proto)",
      "Standard protojson",
      "protojsonx (Lib)",
      "protojsonx (Plugin)"
    ],
    "datasets": [
      {
        "label": "Small Unmarshal (ns/op)",
        "data": [116, 948, 251, 138],
        "backgroundColor": [
          "rgba(50, 205, 50, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)"
        ],
        "borderColor": [
          "rgba(50, 205, 50, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)"
        ],
        "borderWidth": 1
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Unmarshaling Performance (Small Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "display": false
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
<summary><b>Show complete data table</b></summary>

| Format / Config (Small Payload) | ns/op | Memory (B/op) | Allocations/op | Speed vs Standard protojson |
| :--- | :---: | :---: | :---: | :---: |
| **Concrete (vtproto)** | 26 ns | 16 B | 1 | - |
| **Concrete (proto)** | 116 ns | 96 B | 2 | - |
| **Concrete (JSON)** | 777 ns | 280 B | 6 | - |
| **Concrete (protojsonx - Plugin)** | **138 ns** | **96 B** | **2** | **6.9x faster** |
| **Concrete (protojsonx - Lib)** | **251 ns** | **96 B** | **3** | **3.8x faster** |
| **Concrete (protojson - standard)** | 948 ns | 336 B | 14 | 1.0x (Baseline) |

</details>
  {{< /tab >}}
  {{< tab name="Medium Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Binary (proto)",
      "Standard protojson",
      "protojsonx (Lib)",
      "protojsonx (Plugin)"
    ],
    "datasets": [
      {
        "label": "Medium Unmarshal (ns/op)",
        "data": [577, 3760, 1085, 733],
        "backgroundColor": [
          "rgba(50, 205, 50, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)"
        ],
        "borderColor": [
          "rgba(50, 205, 50, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)"
        ],
        "borderWidth": 1
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Unmarshaling Performance (Medium Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "display": false
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
<summary><b>Show complete data table</b></summary>

| Format / Config (Medium Payload) | ns/op | Memory (B/op) | Allocations/op | Speed vs Standard protojson |
| :--- | :---: | :---: | :---: | :---: |
| **Concrete (vtproto)** | 327 ns | 432 B | 14 | - |
| **Concrete (proto)** | 577 ns | 560 B | 15 | - |
| **Concrete (JSON)** | 2,927 ns | 688 B | 19 | - |
| **Concrete (protojsonx - Plugin)** | **733 ns** | **528 B** | **14** | **5.1x faster** |
| **Concrete (protojsonx - Lib)** | **1,085 ns** | **576 B** | **16** | **3.5x faster** |
| **Concrete (protojson - standard)** | 3,760 ns | 1,304 B | 58 | 1.0x (Baseline) |

</details>
  {{< /tab >}}
  {{< tab name="Large Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Binary (proto)",
      "Standard protojson",
      "protojsonx (Lib)",
      "protojsonx (Plugin)"
    ],
    "datasets": [
      {
        "label": "Large Unmarshal (ns/op)",
        "data": [56842, 381304, 113750, 73070],
        "backgroundColor": [
          "rgba(50, 205, 50, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)"
        ],
        "borderColor": [
          "rgba(50, 205, 50, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)"
        ],
        "borderWidth": 1
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Unmarshaling Performance (Large Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "display": false
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
<summary><b>Show complete data table</b></summary>

| Format / Config (Large Payload) | ns/op | Memory (B/op) | Allocations/op | Speed vs Standard protojson |
| :--- | :---: | :---: | :---: | :---: |
| **Concrete (vtproto)** | 37,442 ns | 58,168 B | 1,508 | - |
| **Concrete (proto)** | 56,842 ns | 58,232 B | 1,509 | - |
| **Concrete (JSON)** | 275,105 ns | 70,584 B | 1,216 | - |
| **Concrete (protojsonx - Plugin)** | **73,070 ns** | **55,008 B** | **1,407** | **5.2x faster** |
| **Concrete (protojsonx - Lib)** | **113,750 ns** | **62,232 B** | **1,709** | **3.4x faster** |
| **Concrete (protojson - standard)** | 381,304 ns | 119,256 B | 5,713 | 1.0x (Baseline) |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Takeaways: Unmarshaling

*   **Statically generated plugin delivers massive speedups**: For unmarshaling, the plugin generated path (`protoc-gen-go-protojsonx`) bypasses dynamic parsing, tag parsing, and runtime table checks. It parses structures directly into concrete fields, achieving up to **5.2x - 6.9x speedup** on static payloads.
*   **Reduced allocation overhead**: By generating type-specific parsing routines, the plugin reduces intermediate allocations. For example, unmarshaling a Large payload drops from **5,713 allocations/op** with standard `protojson` down to **1,407 allocations/op** with the plugin (and 1,709 with the Library mode).
*   **The parsing bottleneck shifts to structural layout**: With reflection and metadata checks eliminated, the remaining processing cost is largely dictated by Go's representation of the data. Flat, contiguous static structures parse extremely quickly.

---

## How `protojsonx` Compares to Binary Formats

For static schemas, the rough performance shape looks like this:

1. **Official `protojson` (Slowest)**: Heavy runtime reflection and extensive heap allocations.
2. **Standard `encoding/json`**: Decent standard Go JSON serialization, but lacks Protobuf schema integration.
3. **`protojsonx` Library**: Pre-compiled offsets and memory reuse bring JSON speed close to binary.
4. **`protojsonx` Plugin**: Statically generated reflection-free code path, achieving maximum speed.
5. **Standard binary `proto`**: Compact binary Protobuf format using standard Go serialization.
6. **`vtproto` (Fastest)**: Statically generated, reflection-free optimized binary serialization.

For static-message marshaling, `protojsonx` (especially with the plugin) gets much closer to standard binary Protobuf than official `protojson`, and in these benchmarks it beats generic `encoding/json`.

---

## Methodology

To ensure these results are reproducible, here are the environment parameters and test details used for this benchmark run:

*   **Go Version**: `go version go1.26 darwin/arm64`
*   **Machine Details**: Apple M1 Pro (10-core CPU, 16GB unified memory, macOS)
*   **Target Library Version**: `github.com/sudorandom/protojsonx v0.0.5`
*   **Benchmark Source**: The benchmark code is available in the posts directory under [content/posts/2026/benchmarking-protojsonx/benchmarks/](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/benchmarking-protojsonx/benchmarks/).
*   **Test Execution Command**:
    ```bash
    go test -bench="." -benchmem -benchtime=1s ./...
    ```
*   **Execution Conditions**: Benchmarks were run on an otherwise idle machine. Results are the `ns/op`, `B/op`, and `allocs/op` reported by Go's benchmark runner. `GOMAXPROCS` was set to the default system allocation of 8.
*   **Microbenchmark Caveat**: As always with microbenchmarks, the exact numbers matter less than the shape of the results. Benchmark your own schemas before treating these results as production guidance.

---

## Recommendations

If you require high performance, follow these guidelines to pick the right pattern:

1. **Evaluate replacing `protojson` on hot paths**: If your service spends meaningful CPU time encoding or decoding ProtoJSON, `protojsonx` is worth benchmarking against your own schemas. **Use the `protoc-gen-go-protojsonx` plugin**: it delivers the absolute maximum performance with reflection-free statically generated paths. The dynamic library mode remains a great drop-in fallback if code generation is not available.
2. **Expose static contracts**: Avoid dynamic schema-less fields on hot paths. Even with the fastest JSON serializers, dynamic value trees impose a heavy allocation and pointer-chasing penalty.
3. **If you must support dynamic payloads on hot paths**: Use **Opaque JSON Packaging** rather than `Value`. For example, a `bytes raw_json = 1;` or `string raw_json = 1;` field preserves the dynamic payload and lets you bypass parser loops entirely on intermediate nodes.
4. **Prefer static schemas on hot paths**: `protojsonx` helps most when the schema is static. The more your data model becomes dynamic, the less a schema-aware serializer can help.
