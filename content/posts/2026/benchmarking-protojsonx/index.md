---
title: "ProtoJSON Without the Reflection Tax: Benchmarking protojsonx"
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
---

In my [previous article](/posts/hidden-cost-of-google-protobuf-value/), I explored the performance bottlenecks of using dynamic JSON structures like `google.protobuf.Value` and official `protojson` in Go. The benchmark results were clear: reflection, pointer chasing, and runtime descriptor traversal add significant latency and memory allocation overhead.

ProtoJSON sits in an awkward place. It gives Protobuf-backed APIs a JSON representation that human operators, browsers, API gateways, and debugging tools can work with, but Go’s official `protojson` implementation pays heavily for this generality. To resolve field names, enum values, presence semantics, and message structure at runtime, it has to walk message descriptors using Protobuf reflection (`protoreflect`) and allocate intermediate values.

This is the right tradeoff for correctness and compatibility. But it made me wonder: *how much of this cost is fundamental to ProtoJSON, and how much comes from repeatedly resolving schema mappings at runtime?*

To explore this, I built [protojsonx](https://github.com/sudorandom/protojsonx), an experimental, mostly API-compatible replacement for Go's official `protojson` library. It implements two strategies: compiling descriptors into flat offset layout tables once at startup (**Runtime Table Mode**), and generating static, reflection-free parsing routines via a protoc plugin (**Generated Plugin Mode**).

In this post, we'll look at the benchmarks comparing standard `protojson`, raw struct-based JSON serializers (`encoding/json` and `json` v2), `protojsonx` (both in Runtime Table and Generated Plugin modes), and raw binary Protobuf (`proto` and the optimized reflection-free `vtproto`). The results show that by leveraging schema knowledge up-front, ProtoJSON performance can get surprisingly close to standard binary Protobuf speeds, and in these benchmarks it surpasses generic JSON throughput.

{{< github-repo repo="sudorandom/protojsonx" description="An experimental faster ProtoJSON encoder and decoder for Go." >}}

> **Warning:** `protojsonx` is highly experimental at this stage. It does pass the official Protobuf conformance tests, which gives me confidence that it follows the expected ProtoJSON behavior, but I would still treat it as early-stage software.

---

## What is protojsonx?

[protojsonx](https://github.com/sudorandom/protojsonx) is designed as a mostly API-compatible experimental replacement for Go's official `google.golang.org/protobuf/encoding/protojson` library. It supports core ProtoJSON behaviors like field names, enums, presence semantics, unknown-field handling, `json_name`, oneofs, maps, and standard marshaling/unmarshaling options. However, it does not yet cover every edge case or configuration option of the official library (such as dynamic `Any` resolving).

To make serialization faster, it implements two key optimization strategies:

*   **Runtime Table Mode**: This mode implements the startup compilation strategy. It builds flat offset tables at initialization, allowing it to marshal and unmarshal structures using fast, sequential offset arithmetic instead of per-call descriptor traversal.
*   **Generated Plugin Mode**: For the highest-performance path, `protojsonx` provides a protoc plugin (`protoc-gen-go-protojsonx`) that generates type-specific marshaling and unmarshaling methods directly. This bypasses runtime table lookups, repeated field-name matching, and runtime type assertions, removing enough overhead that JSON formatting and field access become much more prominent in the remaining cost.

---

## The Benchmark Setup

I used the benchmark suite from my previous article, running on Go 1.26 on an Apple M1 Pro. The configurations are:
* **Small:** A flat object with 4 fields (string ID, status boolean, age integer, score float).
* **Medium:** A nested user signup event containing actor object, string tags, and metadata map.
* **Large:** An array repeating the Medium object 100 times.

### The Rules of the Game

All tests use equivalent payload shapes and measure end-to-end marshal/unmarshal cost, including allocations. The generic JSON cases use plain Go structs with similar fields, while the protobuf cases use generated protobuf messages.

The JSON payloads use the same logical fields, types, and values, though the ProtoJSON and generic JSON implementations are not intended to be byte-for-byte semantic equivalents. ProtoJSON has additional semantic requirements around field names, presence, enums, well-known types, and numeric encoding. The useful comparison is not “which library implements the same thing,” but “how much does ProtoJSON cost compared with familiar JSON tooling?”

I also included `hyperpb` as another example of descriptor/layout-driven protobuf processing. Note that it is not a ProtoJSON library, so its numbers should be read as a binary protobuf reference point rather than a direct competitor.

---

## Marshaling Performance

Marshaling is the process of serializing Go values into bytes. The charts and tables below group the results by payload sizes.

{{< tabs >}}
  {{< tab name="Small Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "protojson",
      "encoding/json",
      "encoding/json/v2",
      "protojsonx (Runtime Tables)",
      "protojsonx (Generated Plugin)",
      "proto.Marshal",
      "vtproto",
      "hyperpb"
    ],
    "datasets": [
      {
        "label": "Small Marshal (ns/op)",
        "data": [648, 171, 276, 144, 106, 82, 24, 292],
        "backgroundColor": [
          "rgba(186, 85, 211, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 130, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(50, 205, 50, 0.75)",
          "rgba(0, 250, 154, 0.75)",
          "rgba(34, 139, 34, 0.75)"
        ],
        "borderColor": [
          "rgba(186, 85, 211, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 130, 0, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(50, 205, 50, 1)",
          "rgba(0, 250, 154, 1)",
          "rgba(34, 139, 34, 1)"
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

| Format / Serializer | ns/op | Memory (B/op) | Allocations/op | Speed vs protojson |
| :--- | :---: | :---: | :---: | :---: |
| **protojson** | 648 ns | 512 B | 12 | 1.0x (Baseline) |
| **encoding/json** | 171 ns | 64 B | 1 | - |
| **encoding/json/v2** | 276 ns | 112 B | 2 | - |
| **protojsonx (Runtime Tables)** | **144 ns** | **64 B** | **1** | **4.5x faster** |
| **protojsonx (Generated Plugin)** | **106 ns** | **64 B** | **1** | **6.1x faster** |
| **proto.Marshal** | 82 ns | 32 B | 1 | - |
| **vtproto** | 24 ns | 32 B | 1 | - |
| **hyperpb** | 292 ns | 144 B | 7 | - |

</details>
  {{< /tab >}}
  {{< tab name="Medium Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "protojson",
      "encoding/json",
      "encoding/json/v2",
      "protojsonx (Runtime Tables)",
      "protojsonx (Generated Plugin)",
      "proto.Marshal",
      "vtproto",
      "hyperpb"
    ],
    "datasets": [
      {
        "label": "Medium Marshal (ns/op)",
        "data": [2361, 517, 805, 379, 323, 292, 108, 1069],
        "backgroundColor": [
          "rgba(186, 85, 211, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 130, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(50, 205, 50, 0.75)",
          "rgba(0, 250, 154, 0.75)",
          "rgba(34, 139, 34, 0.75)"
        ],
        "borderColor": [
          "rgba(186, 85, 211, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 130, 0, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(50, 205, 50, 1)",
          "rgba(0, 250, 154, 1)",
          "rgba(34, 139, 34, 1)"
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

| Format / Serializer | ns/op | Memory (B/op) | Allocations/op | Speed vs protojson |
| :--- | :---: | :---: | :---: | :---: |
| **protojson** | 2,361 ns | 1,721 B | 34 | 1.0x (Baseline) |
| **encoding/json** | 517 ns | 464 B | 2 | - |
| **encoding/json/v2** | 805 ns | 608 B | 3 | - |
| **protojsonx (Runtime Tables)** | **379 ns** | **320 B** | **1** | **6.2x faster** |
| **protojsonx (Generated Plugin)** | **323 ns** | **320 B** | **1** | **7.3x faster** |
| **proto.Marshal** | 292 ns | 176 B | 1 | - |
| **vtproto** | 108 ns | 176 B | 1 | - |
| **hyperpb** | 1,069 ns | 744 B | 17 | - |

</details>
  {{< /tab >}}
  {{< tab name="Large Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "protojson",
      "encoding/json",
      "encoding/json/v2",
      "protojsonx (Runtime Tables)",
      "protojsonx (Generated Plugin)",
      "proto.Marshal",
      "vtproto",
      "hyperpb"
    ],
    "datasets": [
      {
        "label": "Large Marshal (ns/op)",
        "data": [237191, 41091, 61035, 32185, 27617, 24455, 7704, 105781],
        "backgroundColor": [
          "rgba(186, 85, 211, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 130, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(50, 205, 50, 0.75)",
          "rgba(0, 250, 154, 0.75)",
          "rgba(34, 139, 34, 0.75)"
        ],
        "borderColor": [
          "rgba(186, 85, 211, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 130, 0, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(50, 205, 50, 1)",
          "rgba(0, 250, 154, 1)",
          "rgba(34, 139, 34, 1)"
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

| Format / Serializer | ns/op | Memory (B/op) | Allocations/op | Speed vs protojson |
| :--- | :---: | :---: | :---: | :---: |
| **protojson** | 237,191 ns | 243,736 B | 2,728 | 1.0x (Baseline) |
| **encoding/json** | 41,091 ns | 32,829 B | 2 | - |
| **encoding/json/v2** | 61,035 ns | 32,868 B | 3 | - |
| **protojsonx (Runtime Tables)** | **32,185 ns** | **32,803 B** | **1** | **7.4x faster** |
| **protojsonx (Generated Plugin)** | **27,617 ns** | **32,787 B** | **1** | **8.6x faster** |
| **proto.Marshal** | 24,455 ns | 18,432 B | 1 | - |
| **vtproto** | 7,704 ns | 18,432 B | 1 | - |
| **hyperpb** | 105,781 ns | 107,216 B | 1,022 | - |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Takeaways: Marshaling

Most of the marshaling performance gain comes from compiling descriptors up-front. Standard `protojson` spends its cycles querying Go’s reflection and descriptor trees on the fly, creating a continuous stream of allocations. Compiling those layout mappings once at startup (**Runtime Table Mode**) allows the serialization process to bypass runtime lookup loops entirely, converting the fields to JSON via sequential offset math.

The Generated Plugin Mode takes this layout compile strategy to its logical conclusion. By emitting type-specific serialization code during `protoc` generation, it bypasses the runtime offsets table entirely. This removes enough lookup and assertion overhead that JSON formatting and field access become much more prominent in the remaining cost.

On a broader scale, these benchmarks show that schema-guided ProtoJSON can achieve JSON serialization speeds that easily surpass standard library JSON (`encoding/json` and the experimental v2 implementation) while remaining highly competitive with standard binary Protobuf (`proto.Marshal`). For context on other descriptor-driven options, `hyperpb` shows slower marshaling times here because its architecture is heavily optimized around unmarshaling and zero-copy offset decoding rather than fast serialization.

---

## Unmarshaling Performance

Unmarshaling parses incoming bytes back into Go values. In the benchmarks below, I present `protojsonx` in both its dynamic **Runtime Table Mode** and its statically compiled **Generated Plugin Mode**.

{{< tabs >}}
  {{< tab name="Small Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "protojson",
      "encoding/json",
      "encoding/json/v2",
      "protojsonx (Runtime Tables)",
      "protojsonx (Generated Plugin)",
      "proto.Unmarshal",
      "vtproto",
      "hyperpb"
    ],
    "datasets": [
      {
        "label": "Small Unmarshal (ns/op)",
        "data": [942, 780, 333, 255, 143, 117, 26, 366],
        "backgroundColor": [
          "rgba(186, 85, 211, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 130, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(50, 205, 50, 0.75)",
          "rgba(0, 250, 154, 0.75)",
          "rgba(34, 139, 34, 0.75)"
        ],
        "borderColor": [
          "rgba(186, 85, 211, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 130, 0, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(50, 205, 50, 1)",
          "rgba(0, 250, 154, 1)",
          "rgba(34, 139, 34, 1)"
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

| Format / Serializer | ns/op | Memory (B/op) | Allocations/op | Speed vs protojson |
| :--- | :---: | :---: | :---: | :---: |
| **protojson** | 942 ns | 336 B | 14 | 1.0x (Baseline) |
| **encoding/json** | 780 ns | 280 B | 6 | - |
| **encoding/json/v2** | 333 ns | 48 B | 1 | - |
| **protojsonx (Runtime Tables)** | **255 ns** | **96 B** | **3** | **3.7x faster** |
| **protojsonx (Generated Plugin)** | **143 ns** | **96 B** | **2** | **6.6x faster** |
| **proto.Unmarshal** | 117 ns | 96 B | 2 | - |
| **vtproto** | 26 ns | 16 B | 1 | - |
| **hyperpb** | 366 ns | 797 B | 4 | - |

</details>
  {{< /tab >}}
  {{< tab name="Medium Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "protojson",
      "encoding/json",
      "encoding/json/v2",
      "protojsonx (Runtime Tables)",
      "protojsonx (Generated Plugin)",
      "proto.Unmarshal",
      "vtproto",
      "hyperpb"
    ],
    "datasets": [
      {
        "label": "Medium Unmarshal (ns/op)",
        "data": [3730, 2898, 1121, 1162, 724, 570, 319, 629],
        "backgroundColor": [
          "rgba(186, 85, 211, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 130, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(50, 205, 50, 0.75)",
          "rgba(0, 250, 154, 0.75)",
          "rgba(34, 139, 34, 0.75)"
        ],
        "borderColor": [
          "rgba(186, 85, 211, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 130, 0, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(50, 205, 50, 1)",
          "rgba(0, 250, 154, 1)",
          "rgba(34, 139, 34, 1)"
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

| Format / Serializer | ns/op | Memory (B/op) | Allocations/op | Speed vs protojson |
| :--- | :---: | :---: | :---: | :---: |
| **protojson** | 3,730 ns | 1,304 B | 58 | 1.0x (Baseline) |
| **encoding/json** | 2,898 ns | 688 B | 19 | - |
| **encoding/json/v2** | 1,121 ns | 256 B | 4 | - |
| **protojsonx (Runtime Tables)** | **1,162 ns** | **576 B** | **16** | **3.2x faster** |
| **protojsonx (Generated Plugin)** | **724 ns** | **528 B** | **14** | **5.2x faster** |
| **proto.Unmarshal** | 570 ns | 560 B | 15 | - |
| **vtproto** | 319 ns | 432 B | 14 | - |
| **hyperpb** | 629 ns | 1,445 B | 5 | - |

</details>
  {{< /tab >}}
  {{< tab name="Large Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "protojson",
      "encoding/json",
      "encoding/json/v2",
      "protojsonx (Runtime Tables)",
      "protojsonx (Generated Plugin)",
      "proto.Unmarshal",
      "vtproto",
      "hyperpb"
    ],
    "datasets": [
      {
        "label": "Large Unmarshal (ns/op)",
        "data": [376027, 277076, 110644, 108963, 70317, 56362, 37173, 25021],
        "backgroundColor": [
          "rgba(186, 85, 211, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 130, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(50, 205, 50, 0.75)",
          "rgba(0, 250, 154, 0.75)",
          "rgba(34, 139, 34, 0.75)"
        ],
        "borderColor": [
          "rgba(186, 85, 211, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 130, 0, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(50, 205, 50, 1)",
          "rgba(0, 250, 154, 1)",
          "rgba(34, 139, 34, 1)"
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

| Format / Serializer | ns/op | Memory (B/op) | Allocations/op | Speed vs protojson |
| :--- | :---: | :---: | :---: | :---: |
| **protojson** | 376,027 ns | 119,256 B | 5,713 | 1.0x (Baseline) |
| **encoding/json** | 277,076 ns | 70,584 B | 1,216 | - |
| **encoding/json/v2** | 110,644 ns | 54,304 B | 309 | - |
| **protojsonx (Runtime Tables)** | **108,963 ns** | **62,232 B** | **1,709** | **3.5x faster** |
| **protojsonx (Generated Plugin)** | **70,317 ns** | **55,008 B** | **1,407** | **5.4x faster** |
| **proto.Unmarshal** | 56,362 ns | 58,232 B | 1,509 | - |
| **vtproto** | 37,173 ns | 58,168 B | 1,508 | - |
| **hyperpb** | 25,021 ns | 60,044 B | 12 | - |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Takeaways: Unmarshaling

Unmarshaling is where the Generated Plugin Mode shows its real strength. If you’ve ever looked at Go’s standard `protojson` decoder, it spends a massive amount of time matching string keys against descriptors, allocating intermediate maps, and resolving types at runtime. Bypassing all of that by compiling direct field assignments in the generator yields a massive speedup, with 5.2x to 6.6x faster throughput on static payloads. It also reduces heap churn substantially; unmarshaling the Large payload drops from over 5,700 allocations down to just 1,407.

Generalizing the results, the Generated Plugin Mode demonstrates that ProtoJSON unmarshaling can be significantly faster than standard library JSON and `encoding/json/v2`, while performing within striking distance of standard binary Protobuf (`proto.Unmarshal`).

That said, unmarshaling still has a wider performance gap relative to binary Protobuf than marshaling does. This is a fundamental constraint of the JSON format itself. A binary decoder can skip fields using length prefixes and parse integer types with very little overhead. A JSON decoder, by contrast, is forced to parse string layouts, handle token boundaries, and instantiate nested objects and map fields on the fly. But even with these structural constraints, the reflection-free generated code narrows the gap significantly, turning what used to be a massive bottleneck into a much smaller and more predictable cost.

---

## How protojsonx Compares to Generic JSON & Binary Formats

The benchmarks show that the basic idea works: **by using schema-guided code generation, we can achieve better performance with ProtoJSON than Go's standard library `encoding/json` package.**

The other interesting result is that Go's experimental **V2 JSON implementation (`go-json-experiment/json`) is shaping up to be extremely performant.** Under unmarshaling, V2 JSON achieves significant latency improvements and allocator reductions over V1 (e.g. unmarshaling the Large payload drops from 277,076 ns/op down to 110,644 ns/op).

While V2 JSON gets close to `protojsonx` Runtime Table Mode, the statically generated `protojsonx` Generated Plugin Mode was the fastest JSON option in these benchmarks.

The overall performance landscape for static schemas looks like this:

| Format / Serializer | Performance Tier | Characteristics |
| :--- | :--- | :--- |
| **Official `protojson`** | Slowest JSON | Walks descriptor trees dynamically via reflection on every call |
| **`encoding/json` (V1)** | General JSON | General-purpose reflection-based standard library parser |
| **`protojsonx` Runtime Table Mode** | Fast JSON | Compiles descriptors once at startup into sequential offset tables |
| **`encoding/json/v2`** | Fast JSON | Redesigned standard parser with optimized parsing and lower heap allocations |
| **`protojsonx` Generated Plugin Mode** | Fastest JSON | Statically generated type-specific parsing routines (no reflection) |
| **Standard binary `proto`** | Fast Binary | Compact binary Protobuf using the standard Go protobuf runtime |
| **`vtproto`** | Fastest Binary | Statically generated type-specific binary routines (no reflection) |

For static-message marshaling, `protojsonx` (especially with the plugin) gets much closer to standard binary Protobuf than official `protojson`, and in these benchmarks it beats generic `encoding/json` and V2 JSON.

---

## Methodology

To ensure these results are reproducible, here are the environment parameters and test details used for this benchmark run:

*   **Go Version**: `go version go1.26.3 darwin/arm64`
*   **Machine Details**: Apple M1 Pro (10-core CPU, 16GB unified memory, macOS)
*   **Target Library Version**: `github.com/sudorandom/protojsonx@v0.0.6`
*   **Benchmark Commit**: `b8fff78c`
*   **Benchmark Source**: The benchmark code is available in the [benchmarks/](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/benchmarking-protojsonx/benchmarks/) folder.
*   **Test Execution Command**:
    ```bash
    go test -bench="." -benchmem -benchtime=5s -count=5 ./...
    ```

All benchmarks were run on an otherwise idle machine using a multi-run sequence to ensure statistical stability. All runs were performed on an AC-powered, thermally-settled machine to prevent thermal throttling or low-power state interference. The reported metrics (`ns/op`, `B/op`, and `allocs/op`) represent the averages returned by Go's benchmark runner across five independent iterations of 5-second runs. The JSON payloads use the same logical fields, types, and values, though the ProtoJSON and generic JSON implementations are not intended to be byte-for-byte semantic equivalents.

I left `GOMAXPROCS` at Go’s default for this machine, which was 8.

As always with microbenchmarks, the exact numbers matter less than the overall shape of the results. You should benchmark your own schemas and payloads under production-realistic environments before drawing final design conclusions.

---

## Try it Out & Give Feedback!

`protojsonx` is still highly experimental and early-stage software. It passes the official Protobuf conformance tests, which gives me confidence that the core serialization and parsing behavior is on the right track, but I would not treat it as boring infrastructure yet.

If you are serving JSON APIs backed by Protobuf and `protojson` is showing up in your profiles, I’d love for you to try it against your own schemas.

I’m especially interested in results from real production-shaped messages: large repeated fields, maps, oneofs, well-known types, custom `json_name` usage, and other cases that are more interesting than tiny benchmark fixtures.

You can find the code and instructions on GitHub:
* **GitHub Repository**: [sudorandom/protojsonx](https://github.com/sudorandom/protojsonx)

Please file issues or share benchmark results there. The more weird schemas, the better.
