---
title: "Speeding Up ProtoJSON: Benchmarking protojsonx"
date: "2026-06-30T15:20:00Z"
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
---

While modern Go frameworks like ConnectRPC handle browser-facing APIs natively, ProtoJSON remains a vital tool for debugging—particularly when frontend teams need human-readable logs, inspection tools, or mock JSON payloads to interact with Protobuf schemas.

The downside is performance. Go's official `protojson` package does a lot of work at runtime: walking descriptors, handling Protobuf-specific JSON mapping rules, resolving dynamic types, and allocating intermediate values. Compared with binary Protobuf, or even ordinary `encoding/json` on static Go structs, that overhead can be surprisingly large.

I built [`protojsonx`](https://github.com/sudorandom/protojsonx) to see how much of that cost could be removed without changing the Protobuf JSON format. The basic idea is to do more work up front, compile message layouts into faster lookup tables, and avoid repeated reflection-heavy traversal on every marshal or unmarshal call.

In this article, we’ll run a benchmark suite against `protojsonx` to see how it compares with standard `protojson`, raw JSON, dynamic Well-Known Types (`google.protobuf.Value`), and standard binary Protobuf (`proto` and reflection-free `vtproto`).

{{< github-repo repo="sudorandom/protojsonx" description="An experimental faster ProtoJSON encoder and decoder for Go." >}}

> **Warning:** `protojsonx` is highly experimental at this stage. It does pass the official Protobuf conformance tests, which gives me confidence that it follows the expected ProtoJSON behavior, but I would still treat it as early-stage software.

---

## What is `protojsonx`?

`protojsonx` is designed as a drop-in replacement for Go's official `google.golang.org/protobuf/encoding/protojson` library. To make that faster, it implements three key optimization strategies:

*   **Descriptor Pre-compilation**: Rather than walking message descriptors dynamically via Go's reflection API on every call, `protojsonx` inspects them once and compiles them into flat, sequential layout tables. This allows it to marshal and unmarshal structures using fast, sequential array iterations, avoiding per-call descriptor traversal for static messages.
*   **Zero-Copy Parsing**: During unmarshaling, standard parsers often allocate when materializing strings and raw byte slices from the input. When `ZeroCopy: true` is configured, `protojsonx` points Go string and byte slices directly to the original JSON input buffer. This avoids copying memory, reducing allocations for string-heavy payloads.
    *   *Warning: This introduces the risk of **memory pinning**. Because the unmarshaled string slices point directly into the input byte buffer, the entire original JSON buffer cannot be garbage collected as long as any of those string references remain active in memory. For long-lived cache entries or memory-sensitive storage, copying the parsed strings is still recommended.*
*   **Monotonic Bump Allocation**: When deserializing deeply nested objects, the Go runtime normally makes hundreds of small heap allocations for submessages, creating significant garbage collection (GC) pressure. `protojsonx` supports custom memory allocators like the built-in [`BumpAllocator`](https://github.com/sudorandom/protojsonx/blob/main/allocator.go). Instead of using Go runtime arenas, it pre-allocates memory chunks using standard byte slices (starting at 4KB). It calculates required alignment offsets using bitwise operations, clears the memory block, and uses `unsafe.Pointer` with `reflect.NewAt` to allocate zero values of submessages directly from the active chunk. Calling `alloc.Reset()` clears the offset index so the pre-allocated byte slices can be reused for subsequent operations, bypassing Go's heap allocator for submessages.

### Operating Modes

To evaluate the impact of these strategies, we run `protojsonx` in two distinct modes during unmarshaling:
1. **Default Mode**: A direct drop-in API path using `protojsonx.Unmarshal(...)`. This mode has no special input-buffer lifetime requirements.
2. **Optimized Mode**: Configured with `ZeroCopy: true` and the `BumpAllocator`. This mode reduces allocation pressure, especially for larger static messages, but it comes with buffer lifetime constraints and is not universally faster across every payload shape.

---

## The Benchmark Setup

We used the benchmark suite from our previous article, running on Go 1.26 on an Apple M1 Pro. The configurations are:
* **Small:** A flat object with 4 fields (string ID, status boolean, age integer, score float).
* **Medium:** A nested user signup event containing actor object, string tags, and metadata map.
* **Large:** An array repeating the Medium object 100 times.

We group the results directly by data representation (**Static Message**, **google.protobuf.Any**, and **google.protobuf.Value**), showing the serialization formats side-by-side.

### Payload Decoding in `Any` Benchmarks

There is one important caveat about how inner payload decoding is handled for `google.protobuf.Any` across the different test configurations:

* **Unmarshaling Benchmarks (Full End-to-End Decoding)**: For both binary and JSON unmarshaling benchmarks of `Any`, the code **does** decode the inner message by calling `anypb.UnmarshalTo`. This forces the payload to be fully deserialized into a concrete Go struct, allowing us to measure the complete end-to-end processing cost.
* **JSON Marshaling Benchmarks (Runtime Parsing)**: When marshaling `google.protobuf.Any` to JSON (`protojson` or `protojsonx`), the serializer has to decode the inner binary payload at runtime to represent its fields as human-readable JSON keys and values.
* **Binary Marshaling Benchmarks (Opaque Envelope)**: In the binary `google.protobuf.Any (proto)` marshal benchmark, the serializer does **not** decode or serialize the inner message at runtime. The inner message is already pre-packed into raw bytes inside `Any.Value`. The binary serializer only writes the envelope fields (`type_url` and the raw `value` byte slice). That is why binary `Any` marshaling is so fast in this benchmark.

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
      "Static Message",
      "google.protobuf.Any",
      "google.protobuf.Value"
    ],
    "datasets": [
      {
        "label": "Binary (proto)",
        "data": [82, 78, 1244],
        "backgroundColor": "rgba(50, 205, 50, 0.75)",
        "borderColor": "rgba(50, 205, 50, 1)",
        "borderWidth": 1
      },
      {
        "label": "Standard protojson",
        "data": [607, 1024, 1918],
        "backgroundColor": "rgba(186, 85, 211, 0.75)",
        "borderColor": "rgba(186, 85, 211, 1)",
        "borderWidth": 1
      },
      {
        "label": "protojsonx",
        "data": [140, 492, 1595],
        "backgroundColor": "rgba(0, 191, 255, 0.75)",
        "borderColor": "rgba(0, 191, 255, 1)",
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
        "labels": { "color": "#fff" }
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
| **Concrete (vtproto)** | 24 ns | 32 B | 1 | - |
| **Concrete (proto)** | 82 ns | 32 B | 1 | - |
| **Concrete (JSON)** | 166 ns | 64 B | 1 | - |
| **Concrete (protojsonx)** | **140 ns** | **64 B** | **1** | **4.3x faster** |
| **Concrete (protojson - standard)** | 607 ns | 512 B | 12 | 1.0x (Baseline) |
| **google.protobuf.Any (proto)** | 78 ns | 80 B | 1 | - |
| **google.protobuf.Any (protojsonx)** | **492 ns** | **208 B** | **3** | **2.1x faster** |
| **google.protobuf.Any (protojson - standard)** | 1,024 ns | 680 B | 15 | 1.0x (Baseline) |
| **google.protobuf.Value (proto)** | 1,244 ns | 208 B | 9 | - |
| **google.protobuf.Value (protojsonx)** | **1,595 ns** | **632 B** | **18** | **1.2x faster** |
| **google.protobuf.Value (protojson - standard)** | 1,918 ns | 688 B | 22 | 1.0x (Baseline) |

</details>
  {{< /tab >}}
  {{< tab name="Medium Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Static Message",
      "google.protobuf.Any",
      "google.protobuf.Value"
    ],
    "datasets": [
      {
        "label": "Binary (proto)",
        "data": [284, 105, 3822],
        "backgroundColor": "rgba(50, 205, 50, 0.75)",
        "borderColor": "rgba(50, 205, 50, 1)",
        "borderWidth": 1
      },
      {
        "label": "Standard protojson",
        "data": [2276, 3347, 6262],
        "backgroundColor": "rgba(186, 85, 211, 0.75)",
        "borderColor": "rgba(186, 85, 211, 1)",
        "borderWidth": 1
      },
      {
        "label": "protojsonx",
        "data": [351, 1182, 5226],
        "backgroundColor": "rgba(0, 191, 255, 0.75)",
        "borderColor": "rgba(0, 191, 255, 1)",
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
        "labels": { "color": "#fff" }
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
| **Concrete (vtproto)** | 98 ns | 176 B | 1 | - |
| **Concrete (proto)** | 284 ns | 176 B | 1 | - |
| **Concrete (JSON)** | 509 ns | 464 B | 2 | - |
| **Concrete (protojsonx)** | **351 ns** | **320 B** | **1** | **6.5x faster** |
| **Concrete (protojson - standard)** | 2,276 ns | 1,721 B | 34 | 1.0x (Baseline) |
| **google.protobuf.Any (proto)** | 105 ns | 224 B | 1 | - |
| **google.protobuf.Any (protojsonx)** | **1,182 ns** | **912 B** | **16** | **2.8x faster** |
| **google.protobuf.Any (protojson - standard)** | 3,347 ns | 2,546 B | 50 | 1.0x (Baseline) |
| **google.protobuf.Value (proto)** | 3,822 ns | 736 B | 25 | - |
| **google.protobuf.Value (protojsonx)** | **5,226 ns** | **2,168 B** | **59** | **1.2x faster** |
| **google.protobuf.Value (protojson - standard)** | 6,262 ns | 2,747 B | 70 | 1.0x (Baseline) |

</details>
  {{< /tab >}}
  {{< tab name="Large Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Static Message",
      "google.protobuf.Any",
      "google.protobuf.Value"
    ],
    "datasets": [
      {
        "label": "Binary (proto)",
        "data": [24195, 10571, 376401],
        "backgroundColor": "rgba(50, 205, 50, 0.75)",
        "borderColor": "rgba(50, 205, 50, 1)",
        "borderWidth": 1
      },
      {
        "label": "Standard protojson",
        "data": [225287, 324182, 622944],
        "backgroundColor": "rgba(186, 85, 211, 0.75)",
        "borderColor": "rgba(186, 85, 211, 1)",
        "borderWidth": 1
      },
      {
        "label": "protojsonx",
        "data": [30406, 117122, 515264],
        "backgroundColor": "rgba(0, 191, 255, 0.75)",
        "borderColor": "rgba(0, 191, 255, 1)",
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
        "labels": { "color": "#fff" }
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
| **Concrete (vtproto)** | 7,220 ns | 18,432 B | 1 | - |
| **Concrete (proto)** | 24,195 ns | 18,432 B | 1 | - |
| **Concrete (JSON)** | 41,684 ns | 32,822 B | 2 | - |
| **Concrete (protojsonx)** | **30,406 ns** | **32,787 B** | **1** | **7.4x faster** |
| **Concrete (protojson - standard)** | 225,287 ns | 243,730 B | 2,728 | 1.0x (Baseline) |
| **google.protobuf.Any (proto)** | 10,571 ns | 22,400 B | 100 | - |
| **google.protobuf.Any (protojsonx)** | **117,122 ns** | **91,258 B** | **1,600** | **2.8x faster** |
| **google.protobuf.Any (protojson - standard)** | 324,182 ns | 254,699 B | 5,001 | 1.0x (Baseline) |
| **google.protobuf.Value (proto)** | 376,401 ns | 79,361 B | 2,401 | - |
| **google.protobuf.Value (protojsonx)** | **515,264 ns** | **217,839 B** | **5,802** | **1.2x faster** |
| **google.protobuf.Value (protojson - standard)** | 626,973 ns | 320,165 B | 6,261 | 1.0x (Baseline) |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Takeaways: Marshaling

*   **Descriptor compilation does most of the work**: Standard `protojson` relies on walking Protobuf reflection APIs on the fly, causing a stream of dynamic allocations. By pre-compiling message layout tables at startup, `protojsonx` avoids per-call descriptor traversal for static messages, completing static serialization in a fraction of the time with minimal allocations.
*   **Schema-guided JSON can beat generic JSON**: Because `protojsonx` is built on an optimized, single-allocation serialization pipeline, it outperforms Go's standard `encoding/json` library in these static-message benchmarks, proving that schema-guided serialization can beat generic JSON serialization for these schemas.
*   **Polymorphism has a runtime cost**: While `protojsonx` significantly speeds up the serialization of dynamic formats like `Any` and `Value`, their performance remains bound by their internal Go structural overhead—specifically, the need to inspect types dynamically and map fields at runtime instead of relying on flat compiled layout tables.

---

## Unmarshaling Performance

Unmarshaling parses incoming bytes back into Go values. In the benchmarks below, we present `protojsonx` in both its drop-in **Default Mode** (no API changes) and its **Optimized Mode** (`ZeroCopy: true` + `BumpAllocator`).

{{< tabs >}}
  {{< tab name="Small Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Static Message",
      "google.protobuf.Any",
      "google.protobuf.Value"
    ],
    "datasets": [
      {
        "label": "Binary (proto)",
        "data": [111, 253, 1348],
        "backgroundColor": "rgba(50, 205, 50, 0.75)",
        "borderColor": "rgba(50, 205, 50, 1)",
        "borderWidth": 1
      },
      {
        "label": "Standard protojson",
        "data": [919, 2097, 2589],
        "backgroundColor": "rgba(186, 85, 211, 0.75)",
        "borderColor": "rgba(186, 85, 211, 1)",
        "borderWidth": 1
      },
      {
        "label": "protojsonx (Default)",
        "data": [241, 1298, 1799],
        "backgroundColor": "rgba(135, 206, 250, 0.75)",
        "borderColor": "rgba(135, 206, 250, 1)",
        "borderWidth": 1
      },
      {
        "label": "protojsonx (ZeroCopy + BumpAlloc)",
        "data": [239, 1292, 1808],
        "backgroundColor": "rgba(0, 191, 255, 0.75)",
        "borderColor": "rgba(0, 191, 255, 1)",
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
        "labels": { "color": "#fff" }
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
| **Concrete (vtproto)** | 24 ns | 16 B | 1 | - |
| **Concrete (proto)** | 111 ns | 96 B | 2 | - |
| **Concrete (JSON)** | 730 ns | 280 B | 6 | - |
| **Concrete (protojsonx - Default)** | **241 ns** | **96 B** | **3** | **3.8x faster** |
| **Concrete (protojsonx - ZeroCopy + BumpAlloc)** | **239 ns** | **82 B** | **2** | **3.8x faster** |
| **Concrete (protojson - standard)** | 919 ns | 336 B | 14 | 1.0x (Baseline) |
| **google.protobuf.Any (proto)** | 253 ns | 256 B | 5 | - |
| **google.protobuf.Any (protojsonx - Default)** | **1,298 ns** | **408 B** | **11** | **1.6x faster** |
| **google.protobuf.Any (protojsonx - ZeroCopy + BumpAlloc)** | **1,292 ns** | **392 B** | **10** | **1.6x faster** |
| **google.protobuf.Any (protojson - standard)** | 2,097 ns | 744 B | 31 | 1.0x (Baseline) |
| **google.protobuf.Value (proto)** | 1,348 ns | 832 B | 26 | - |
| **google.protobuf.Value (protojsonx - Default)** | **1,799 ns** | **976 B** | **34** | **1.4x faster** |
| **google.protobuf.Value (protojsonx - ZeroCopy + BumpAlloc)** | **1,808 ns** | **976 B** | **34** | **1.4x faster** |
| **google.protobuf.Value (protojson - standard)** | 2,589 ns | 1,256 B | 43 | 1.0x (Baseline) |

</details>
  {{< /tab >}}
  {{< tab name="Medium Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Static Message",
      "google.protobuf.Any",
      "google.protobuf.Value"
    ],
    "datasets": [
      {
        "label": "Binary (proto)",
        "data": [541, 715, 4583],
        "backgroundColor": "rgba(50, 205, 50, 0.75)",
        "borderColor": "rgba(50, 205, 50, 1)",
        "borderWidth": 1
      },
      {
        "label": "Standard protojson",
        "data": [3685, 6537, 8590],
        "backgroundColor": "rgba(186, 85, 211, 0.75)",
        "borderColor": "rgba(186, 85, 211, 1)",
        "borderWidth": 1
      },
      {
        "label": "protojsonx (Default)",
        "data": [1033, 3217, 6523],
        "backgroundColor": "rgba(135, 206, 250, 0.75)",
        "borderColor": "rgba(135, 206, 250, 1)",
        "borderWidth": 1
      },
      {
        "label": "protojsonx (ZeroCopy + BumpAlloc)",
        "data": [907, 3082, 6418],
        "backgroundColor": "rgba(0, 191, 255, 0.75)",
        "borderColor": "rgba(0, 191, 255, 1)",
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
        "labels": { "color": "#fff" }
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
| **Concrete (vtproto)** | 300 ns | 432 B | 14 | - |
| **Concrete (proto)** | 541 ns | 560 B | 15 | - |
| **Concrete (JSON)** | 2,836 ns | 688 B | 19 | - |
| **Concrete (protojsonx - Default)** | **1,033 ns** | **576 B** | **16** | **3.6x faster** |
| **Concrete (protojsonx - ZeroCopy + BumpAlloc)** | **907 ns** | **256 B** | **5** | **4.1x faster** |
| **Concrete (protojson - standard)** | 3,685 ns | 1,304 B | 58 | 1.0x (Baseline) |
| **google.protobuf.Any (proto)** | 715 ns | 864 B | 18 | - |
| **google.protobuf.Any (protojsonx - Default)** | **3,217 ns** | **1,496 B** | **37** | **2.0x faster** |
| **google.protobuf.Any (protojsonx - ZeroCopy + BumpAlloc)** | **3,082 ns** | **1,176 B** | **26** | **2.1x faster** |
| **google.protobuf.Any (protojson - standard)** | 6,537 ns | 2,576 B | 105 | 1.0x (Baseline) |
| **google.protobuf.Value (proto)** | 4,583 ns | 2,888 B | 90 | - |
| **google.protobuf.Value (protojsonx - Default)** | **6,523 ns** | **3,592 B** | **122** | **1.3x faster** |
| **google.protobuf.Value (protojsonx - ZeroCopy + BumpAlloc)** | **6,418 ns** | **3,592 B** | **122** | **1.3x faster** |
| **google.protobuf.Value (protojson - standard)** | 8,590 ns | 4,080 B | 145 | 1.0x (Baseline) |

</details>
  {{< /tab >}}
  {{< tab name="Large Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Static Message",
      "google.protobuf.Any",
      "google.protobuf.Value"
    ],
    "datasets": [
      {
        "label": "Binary (proto)",
        "data": [53352, 71266, 470248],
        "backgroundColor": "rgba(50, 205, 50, 0.75)",
        "borderColor": "rgba(50, 205, 50, 1)",
        "borderWidth": 1
      },
      {
        "label": "Standard protojson",
        "data": [368674, 654092, 873012],
        "backgroundColor": "rgba(186, 85, 211, 0.75)",
        "borderColor": "rgba(186, 85, 211, 1)",
        "borderWidth": 1
      },
      {
        "label": "protojsonx (Default)",
        "data": [108337, 321677, 670989],
        "backgroundColor": "rgba(135, 206, 250, 0.75)",
        "borderColor": "rgba(135, 206, 250, 1)",
        "borderWidth": 1
      },
      {
        "label": "protojsonx (ZeroCopy + BumpAlloc)",
        "data": [94208, 309077, 662143],
        "backgroundColor": "rgba(0, 191, 255, 0.75)",
        "borderColor": "rgba(0, 191, 255, 1)",
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
        "labels": { "color": "#fff" }
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
| **Concrete (vtproto)** | 35,597 ns | 58,168 B | 1,508 | - |
| **Concrete (proto)** | 53,352 ns | 58,232 B | 1,509 | - |
| **Concrete (JSON)** | 269,333 ns | 70,584 B | 1,216 | - |
| **Concrete (protojsonx - Default)** | **108,337 ns** | **62,232 B** | **1,709** | **3.4x faster** |
| **Concrete (protojsonx - ZeroCopy + BumpAlloc)** | **94,208 ns** | **17,434 B** | **509** | **3.9x faster** |
| **Concrete (protojson - standard)** | 368,674 ns | 119,256 B | 5,713 | 1.0x (Baseline) |
| **google.protobuf.Any (proto)** | 71,266 ns | 86,400 B | 1,800 | - |
| **google.protobuf.Any (protojsonx - Default)** | **321,677 ns** | **149,661 B** | **3,700** | **2.0x faster** |
| **google.protobuf.Any (protojsonx - ZeroCopy + BumpAlloc)** | **309,077 ns** | **117,648 B** | **2,600** | **2.1x faster** |
| **google.protobuf.Any (protojson - standard)** | 654,092 ns | 257,601 B | 10,500 | 1.0x (Baseline) |
| **google.protobuf.Value (proto)** | 470,248 ns | 291,185 B | 9,011 | - |
| **google.protobuf.Value (protojsonx - Default)** | **670,989 ns** | **363,957 B** | **12,312** | **1.3x faster** |
| **google.protobuf.Value (protojsonx - ZeroCopy + BumpAlloc)** | **662,143 ns** | **363,954 B** | **12,312** | **1.3x faster** |
| **google.protobuf.Value (protojson - standard)** | 873,012 ns | 395,330 B | 14,414 | 1.0x (Baseline) |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Takeaways: Unmarshaling

*   **Bump allocation helps most on static messages**: For large static payloads, the optimized mode cuts allocations substantially by reusing pre-allocated chunks instead of creating thousands of short-lived heap objects. That reduces GC pressure and improves throughput.
*   **Zero-copy helps when strings are part of the hot path**: When it is safe to tie decoded strings to the lifetime of the input buffer, zero-copy parsing can avoid extra string allocations. This is useful, but it is a trade-off, not a free lunch.
*   **The parsing bottleneck shifts to structural layout**: With reflection and memory allocation overhead minimized, the remaining processing cost is largely dictated by Go's representation of the data. Flat, contiguous static structures parse extremely quickly, while `Any` and `Value` get less benefit because their runtime structure still dominates the cost.

---

## Why is `google.protobuf.Value` with `protojsonx` Still So Slow?

One of the most interesting findings from these benchmarks is that while `protojsonx` speeds up static message parsing by **3.5x to 7x**, it only achieves a modest **1.2x to 1.4x improvement** on schema-less `google.protobuf.Value` payloads, even with the monotonic bump allocator enabled.

To understand why this is the case, we have to look at the structural difference in how Go represents these two types of data:

### 1. The Pointer Tree Problem

When you define a static Go struct (or a statically compiled Protobuf struct), the fields are stored in contiguous, flat blocks of memory. This allows serializers to stream writing and reading without chasing pointers around the heap.

In contrast, `google.protobuf.Value` represents a schema-less JSON tree. Because `Value` has to represent many possible JSON shapes, the Go representation becomes a tree of wrappers, oneofs, maps, slices, and pointers.

On a Large payload, this creates a fragmented graph of **over 12,000 pointer nodes**. Any JSON serializer—whether it is Go's standard `encoding/json` or `protojsonx`—must recursively traverse this massive tree, dereferencing thousands of pointers and chasing them through CPU caches. That pushes more time toward memory access and cache misses rather than straightforward parsing work.

### 2. Allocation vs. GC Reduction in the Bump Allocator

Even when allocation is cheaper, `google.protobuf.Value` still produces a large pointer-heavy object graph. The allocator can reduce some heap pressure, but it cannot make recursive maps, lists, oneof wrappers, and pointer chasing behave like a flat generated struct.

The allocation count improves from **14,414 down to 12,312**, but the remaining structure still has to be created and traversed. The allocator helps with one part of the problem; it does not change the shape of the data.

### 3. Loss of Pre-compiled Layout Tables

The main advantage of `protojsonx` is **pre-compiled descriptor tables**. For static messages, `protojsonx` analyzes the message structure once at startup and creates a flat lookup table. It knows exactly which byte offsets hold the string ID, the integer age, etc.

However, `google.protobuf.Value` has no static schema. Its content is completely arbitrary and only resolved at runtime. Therefore, `protojsonx` cannot use a pre-compiled layout table for the fields inside a `Value`. Instead, it must fall back to dynamic runtime inspections: checking if the value is a number, a string, or recursively walking a `map[string]*structpb.Value` at runtime.

Some of that cost is just the shape of the data structure, meaning `google.protobuf.Value` remains slow and allocation-heavy across all serializers.

---

## How `protojsonx` Compares to Binary Formats

For static schemas, the rough performance shape looks like this:

1. **Official `protojson` (Slowest)**: Heavy runtime reflection and extensive heap allocations.
2. **Standard `encoding/json`**: Decent standard Go JSON serialization, but lacks Protobuf schema integration.
3. **`protojsonx` (Fast)**: Pre-compiled offsets and memory reuse bring JSON speed close to binary.
4. **Standard binary `proto` (Faster)**: Compact binary Protobuf format using standard Go serialization.
5. **`vtproto` (Fastest)**: Statically generated, reflection-free optimized binary serialization.

For static-message marshaling, `protojsonx` gets much closer to standard binary Protobuf than official `protojson`, and in these benchmarks it beats generic `encoding/json`. For `Any` and `Value`, the gap remains larger because those types still carry runtime structural costs.

For unmarshaling, binary formats remain clearly faster. `protojsonx` mainly narrows the gap for static schemas, especially when allocation pressure is the dominant cost.

For static schemas, the performance trade-off becomes much smaller when choosing JSON over binary Protobuf for public-facing APIs.

---

## Methodology

To ensure these results are reproducible, here are the environment parameters and test details used for this benchmark run:

*   **Go Version**: `go version go1.26 darwin/arm64`
*   **Machine Details**: Apple M1 Pro (10-core CPU, 16GB unified memory, macOS)
*   **Target Library Version**: `github.com/sudorandom/protojsonx v0.0.4`
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

1. **Evaluate replacing `protojson` on hot paths**: If your service spends meaningful CPU time encoding or decoding ProtoJSON, `protojsonx` is worth benchmarking against your own schemas. It offers a drop-in replacement with significant speedups, plus further optimized modes if you can manage input-buffer lifetimes safely.
2. **Expose static contracts**: Avoid dynamic fields like `google.protobuf.Value` on hot paths. Even with the fastest JSON serializers, dynamic value trees impose a heavy allocation and pointer-chasing penalty.
3. **If you must support dynamic payloads on hot paths**: Use **Opaque JSON Packaging** rather than `Value`. For example, a `bytes raw_json = 1;` or `string raw_json = 1;` field preserves the dynamic payload and lets you bypass parser loops entirely on intermediate nodes, without forcing every service to materialize it as a `structpb.Value` tree.
4. **Prefer static schemas on hot paths**: `protojsonx` helps most when the schema is static. The more your data model becomes "JSON inside Protobuf," the less a schema-aware serializer can help.
