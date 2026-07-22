---
title: "Beating Go's encoding/json with Schema-Guided ProtoJSON"
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

In my [previous article](/posts/hidden-cost-of-google-protobuf-value/), I explored the performance bottlenecks of using dynamic JSON structures like `google.protobuf.Value` and official `protojson` in Go. The benchmark results pointed at the same culprit over and over: reflection, pointer chasing, and descriptor traversal were showing up as real latency and allocation costs. The benchmark results also pointed out that protojson was consistently *really slow*.

ProtoJSON sits in an awkward place. It gives Protobuf-backed APIs a JSON representation that human operators, browsers, API gateways, and debugging tools can work with, but Go’s official `protojson` implementation pays heavily for these abilities. To resolve field names, enum values, presence semantics, and message structure at runtime, it has to walk message descriptors using Protobuf reflection (`protoreflect`) and allocate intermediate values.

This is the right tradeoff for correctness and compatibility. But it made me wonder: *how much of this cost is fundamental to ProtoJSON, and how much comes from repeatedly resolving schema mappings at runtime?*

So I built [protojsonx](https://github.com/sudorandom/protojsonx), partly as an experiment and partly because I wanted to know whether ProtoJSON was inherently slow or whether Go’s implementation was just lacking optimizations. This library implements two strategies: compiling descriptors into flat offset layout tables once at startup (**Runtime Table Mode**), and generating static, reflection-free parsing routines via a protoc plugin (**Generated Plugin Mode**).

In this post, I'll walk through benchmark results comparing standard `protojson`, standard struct-based JSON (`encoding/json` and `encoding/json/v2` currently living at [`github.com/go-json-experiment/json`](https://github.com/go-json-experiment/json)), `protojsonx` in both modes, and raw binary Protobuf (`proto` and `vtproto`). Moving reflection out of the hot path ends up getting ProtoJSON performance much closer to binary protobuf and easily out-performs `encoding/json` and `encoding/json/v2`.

{{< github-repo repo="sudorandom/protojsonx" description="An experimental faster ProtoJSON encoder and decoder for Go." >}}

> **Warning:** `protojsonx` is highly experimental at this stage. It does pass the official Protobuf conformance tests, which gives me confidence that it follows the expected ProtoJSON behavior, but I would still treat it as early-stage software.

---

## What is protojsonx?

[protojsonx](https://github.com/sudorandom/protojsonx) is designed as a mostly API-compatible experimental replacement for Go's official `google.golang.org/protobuf/encoding/protojson` library. It supports core ProtoJSON behaviors like field names, enums, presence semantics, unknown-field handling, `json_name`, oneofs, maps, and standard marshaling/unmarshaling options. However, it does not yet cover every edge case or configuration option of the official library (such as dynamic `Any` resolving).

To make serialization faster, it implements two key optimization strategies:

*   **Runtime Table Mode**: This mode implements the startup compilation strategy. It builds flat offset tables at initialization, allowing it to marshal and unmarshal structures using fast, sequential offset arithmetic instead of per-call descriptor traversal.
*   **Generated Plugin Mode**: For the highest-performance path, `protojsonx` provides a protoc plugin (`protoc-gen-go-protojsonx`) that generates type-specific marshaling and unmarshaling methods directly. At that point, a lot of the runtime bookkeeping disappears. The remaining cost is less about “what field is this?” and more about the unavoidable work of reading or writing JSON. This strategy also inceases the binary size, which may or may not be appropriate.

---

## The Benchmark Setup

I used the benchmark suite from my previous article, running on Go 1.26 on an Apple M1 Pro. The configurations are:
* **Small:** A flat object with 4 fields (string ID, status boolean, age integer, score float).
* **Medium:** A nested user signup event containing actor object, string tags, and metadata map.
* **Large:** An array repeating the Medium object 100 times.

### Methodology Notes

All tests use equivalent payload shapes and measure end-to-end marshal/unmarshal cost, including allocations. The generic JSON cases use plain Go structs with similar fields, while the protobuf cases use generated protobuf messages.

This is not a perfect apples-to-apples comparison. ProtoJSON has extra rules around field names, presence, enums, well-known types, and numeric encoding. The point is narrower: if you already need ProtoJSON-shaped output, how much does that cost compared with the JSON tools Go developers normally reach for? These benchmarks compare the cost of serving similar application-shaped JSON payloads, not identical semantics.

I also included `hyperpb`, a descriptor/layout-driven dynamic protobuf parser built around read-oriented offset decoding, including `hyperpb.Shared` where the benchmark can reuse its arena. It is not a ProtoJSON library, so its numbers should be read as a binary protobuf reference point rather than a direct competitor.

### Payload Sizes

Serialization output size matters, especially when comparing JSON and binary protobuf. These are the payload sizes produced by the benchmark payloads:

| Payload | ProtoJSON bytes | generic JSON bytes | binary proto bytes |
| :--- | ---: | ---: | ---: |
| Small | 55 B | 55 B | 25 B |
| Medium | 293 B | 291 B | 162 B |
| Large | 29,412 B | 29,201 B | 16,500 B |

The ProtoJSON and generic JSON sizes are close, but not byte-identical. Binary protobuf is much smaller for these payloads, so the binary rows should be read as a compact binary reference point rather than a JSON-format comparison.

**Overall, `protojsonx` generated code consistently beats both `encoding/json` and `encoding/json/v2`, while runtime table mode beats `encoding/json` and remains competitive with `encoding/json/v2`.** In fact, for marshaling, generated `protojsonx` comes surprisingly close to the speed of binary protobuf parsing. On average across payloads, generated code is about 6x to 8x faster at marshaling and 5x to 7x faster at unmarshaling than official `protojson`, while outperforming standard `encoding/json` by roughly 1.5x to 2x on marshal and 4x to 5x on unmarshal.

---

## Marshaling Performance

Marshaling is the happier path here. The encoder already has typed Go values in memory, so the main question is how quickly each implementation can walk those values and write JSON bytes.

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
      "hyperpb + Shared"
    ],
    "datasets": [
      {
        "label": "Small Marshal (ns/op)",
        "data": [
          612,
          183,
          289,
          142,
          102,
          83,
          25,
          296
        ],
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
        "ticks": {
          "color": "#fff"
        }
      },
      "y": {
        "ticks": {
          "color": "#fff"
        }
      }
    }
  }
}
{{< /chart >}}

<details>
<summary><b>Show complete data table</b></summary>

| Format / Serializer | ns/op | Memory (B/op) | Allocations/op | Speed vs protojson |
| :--- | :---: | :---: | :---: | :---: |
| **protojson** | 612 ns | 512 B | 12 | 1.0x (Baseline) |
| **encoding/json** | 183 ns | 64 B | 1 | 3.3x faster |
| **encoding/json/v2** | 289 ns | 112 B | 2 | 2.1x faster |
| **protojsonx (Runtime Tables)** | **142 ns** | **64 B** | **1** | **4.3x faster** |
| **protojsonx (Generated Plugin)** | **102 ns** | **64 B** | **1** | **6.0x faster** |
| **proto.Marshal** | 83 ns | 32 B | 1 | 7.4x faster |
| **vtproto** | 25 ns | 32 B | 1 | 24.5x faster |
| **hyperpb + Shared** | 296 ns | 144 B | 7 | 2.1x faster |

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
      "hyperpb + Shared"
    ],
    "datasets": [
      {
        "label": "Medium Marshal (ns/op)",
        "data": [
          2227,
          521,
          803,
          404,
          318,
          285,
          103,
          1062
        ],
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
        "ticks": {
          "color": "#fff"
        }
      },
      "y": {
        "ticks": {
          "color": "#fff"
        }
      }
    }
  }
}
{{< /chart >}}

<details>
<summary><b>Show complete data table</b></summary>

| Format / Serializer | ns/op | Memory (B/op) | Allocations/op | Speed vs protojson |
| :--- | :---: | :---: | :---: | :---: |
| **protojson** | 2,227 ns | 1,722 B | 34 | 1.0x (Baseline) |
| **encoding/json** | 521 ns | 464 B | 2 | 4.3x faster |
| **encoding/json/v2** | 803 ns | 608 B | 3 | 2.8x faster |
| **protojsonx (Runtime Tables)** | **404 ns** | **320 B** | **1** | **5.5x faster** |
| **protojsonx (Generated Plugin)** | **318 ns** | **320 B** | **1** | **7.0x faster** |
| **proto.Marshal** | 285 ns | 176 B | 1 | 7.8x faster |
| **vtproto** | 103 ns | 176 B | 1 | 21.6x faster |
| **hyperpb + Shared** | 1,062 ns | 744 B | 17 | 2.1x faster |

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
      "hyperpb + Shared"
    ],
    "datasets": [
      {
        "label": "Large Marshal (ns/op)",
        "data": [
          223838,
          41250,
          62036,
          31134,
          27601,
          25131,
          7575,
          103495
        ],
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
        "ticks": {
          "color": "#fff"
        }
      },
      "y": {
        "ticks": {
          "color": "#fff"
        }
      }
    }
  }
}
{{< /chart >}}

<details>
<summary><b>Show complete data table</b></summary>

| Format / Serializer | ns/op | Memory (B/op) | Allocations/op | Speed vs protojson |
| :--- | :---: | :---: | :---: | :---: |
| **protojson** | 223,838 ns | 243,749 B | 2728 | 1.0x (Baseline) |
| **encoding/json** | 41,250 ns | 32,823 B | 2 | 5.4x faster |
| **encoding/json/v2** | 62,036 ns | 32,856 B | 3 | 3.6x faster |
| **protojsonx (Runtime Tables)** | **31,134 ns** | **32,797 B** | **1** | **7.2x faster** |
| **protojsonx (Generated Plugin)** | **27,601 ns** | **32,784 B** | **1** | **8.1x faster** |
| **proto.Marshal** | 25,131 ns | 18,432 B | 1 | 8.9x faster |
| **vtproto** | 7,575 ns | 18,432 B | 1 | 29.5x faster |
| **hyperpb + Shared** | 103,495 ns | 107,216 B | 1022 | 2.2x faster |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Takeaways: Marshaling

Most of the marshaling performance gain comes from compiling descriptors up front. Standard `protojson` spends its time querying reflection and descriptor trees on every call, creating a steady stream of allocations. Compiling those layout mappings once at startup (**Runtime Table Mode**) bypasses runtime descriptor lookups entirely, converting fields to JSON with sequential offset math.

The generated plugin takes this a step further by writing field access code directly into the generated `.pb.go` files, removing the generic lookup layer altogether.

The trade-off is typical for generated code: in a repository with hundreds of message types, generating specialized JSON routines will increase binary size.

For these benchmark payloads, schema-guided ProtoJSON marshaling beats both `encoding/json` and `encoding/json/v2`, with the generated encoder landing right next to standard binary `proto.Marshal` (27.6 µs vs 25.1 µs on the large payload).

---

## Unmarshaling Performance

Unmarshaling is the harder side of the benchmark. The decoder has to turn strings, tokens, object keys, maps, and repeated fields back into typed Go values. The tables below show both the runtime-table path and the generated path.

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
      "hyperpb",
      "hyperpb + Shared"
    ],
    "datasets": [
      {
        "label": "Small Unmarshal (ns/op)",
        "data": [
          934,
          752,
          339,
          267,
          136,
          114,
          25,
          360,
          125
        ],
        "backgroundColor": [
          "rgba(186, 85, 211, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 130, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(50, 205, 50, 0.75)",
          "rgba(0, 250, 154, 0.75)",
          "rgba(34, 139, 34, 0.75)",
          "rgba(34, 139, 34, 0.45)"
        ],
        "borderColor": [
          "rgba(186, 85, 211, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 130, 0, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(50, 205, 50, 1)",
          "rgba(0, 250, 154, 1)",
          "rgba(34, 139, 34, 1)",
          "rgba(34, 139, 34, 0.8)"
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
        "ticks": {
          "color": "#fff"
        }
      },
      "y": {
        "ticks": {
          "color": "#fff"
        }
      }
    }
  }
}
{{< /chart >}}

<details>
<summary><b>Show complete data table</b></summary>

| Format / Serializer | ns/op | Memory (B/op) | Allocations/op | Speed vs protojson |
| :--- | :---: | :---: | :---: | :---: |
| **protojson** | 934 ns | 336 B | 14 | 1.0x (Baseline) |
| **encoding/json** | 752 ns | 280 B | 6 | 1.2x faster |
| **encoding/json/v2** | 339 ns | 48 B | 1 | 2.8x faster |
| **protojsonx (Runtime Tables)** | **267 ns** | **96 B** | **3** | **3.5x faster** |
| **protojsonx (Generated Plugin)** | **136 ns** | **96 B** | **2** | **6.9x faster** |
| **proto.Unmarshal** | 114 ns | 96 B | 2 | 8.2x faster |
| **vtproto** | 25 ns | 16 B | 1 | 37.4x faster |
| **hyperpb** | 360 ns | 798 B | 4 | 2.6x faster |
| **hyperpb + Shared** | 125 ns | 65 B | 1 | 7.5x faster |

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
      "hyperpb",
      "hyperpb + Shared"
    ],
    "datasets": [
      {
        "label": "Medium Unmarshal (ns/op)",
        "data": [
          3703,
          2937,
          1133,
          1108,
          720,
          571,
          322,
          638,
          290
        ],
        "backgroundColor": [
          "rgba(186, 85, 211, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 130, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(50, 205, 50, 0.75)",
          "rgba(0, 250, 154, 0.75)",
          "rgba(34, 139, 34, 0.75)",
          "rgba(34, 139, 34, 0.45)"
        ],
        "borderColor": [
          "rgba(186, 85, 211, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 130, 0, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(50, 205, 50, 1)",
          "rgba(0, 250, 154, 1)",
          "rgba(34, 139, 34, 1)",
          "rgba(34, 139, 34, 0.8)"
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
        "ticks": {
          "color": "#fff"
        }
      },
      "y": {
        "ticks": {
          "color": "#fff"
        }
      }
    }
  }
}
{{< /chart >}}

<details>
<summary><b>Show complete data table</b></summary>

| Format / Serializer | ns/op | Memory (B/op) | Allocations/op | Speed vs protojson |
| :--- | :---: | :---: | :---: | :---: |
| **protojson** | 3,703 ns | 1,304 B | 58 | 1.0x (Baseline) |
| **encoding/json** | 2,937 ns | 688 B | 19 | 1.3x faster |
| **encoding/json/v2** | 1,133 ns | 256 B | 4 | 3.3x faster |
| **protojsonx (Runtime Tables)** | **1,108 ns** | **576 B** | **16** | **3.3x faster** |
| **protojsonx (Generated Plugin)** | **720 ns** | **528 B** | **14** | **5.1x faster** |
| **proto.Unmarshal** | 571 ns | 560 B | 15 | 6.5x faster |
| **vtproto** | 322 ns | 432 B | 14 | 11.5x faster |
| **hyperpb** | 638 ns | 1,446 B | 5 | 5.8x faster |
| **hyperpb + Shared** | 290 ns | 357 B | 1 | 12.8x faster |

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
      "hyperpb",
      "hyperpb + Shared"
    ],
    "datasets": [
      {
        "label": "Large Unmarshal (ns/op)",
        "data": [
          377892,
          272103,
          108811,
          111141,
          73389,
          54923,
          36002,
          24835,
          18163
        ],
        "backgroundColor": [
          "rgba(186, 85, 211, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 130, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(50, 205, 50, 0.75)",
          "rgba(0, 250, 154, 0.75)",
          "rgba(34, 139, 34, 0.75)",
          "rgba(34, 139, 34, 0.45)"
        ],
        "borderColor": [
          "rgba(186, 85, 211, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 130, 0, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(50, 205, 50, 1)",
          "rgba(0, 250, 154, 1)",
          "rgba(34, 139, 34, 1)",
          "rgba(34, 139, 34, 0.8)"
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
        "ticks": {
          "color": "#fff"
        }
      },
      "y": {
        "ticks": {
          "color": "#fff"
        }
      }
    }
  }
}
{{< /chart >}}

<details>
<summary><b>Show complete data table</b></summary>

| Format / Serializer | ns/op | Memory (B/op) | Allocations/op | Speed vs protojson |
| :--- | :---: | :---: | :---: | :---: |
| **protojson** | 377,892 ns | 119,256 B | 5713 | 1.0x (Baseline) |
| **encoding/json** | 272,103 ns | 70,584 B | 1216 | 1.4x faster |
| **encoding/json/v2** | 108,811 ns | 54,305 B | 309 | 3.5x faster |
| **protojsonx (Runtime Tables)** | **111,141 ns** | **62,232 B** | **1709** | **3.4x faster** |
| **protojsonx (Generated Plugin)** | **73,389 ns** | **55,008 B** | **1407** | **5.1x faster** |
| **proto.Unmarshal** | 54,923 ns | 58,232 B | 1509 | 6.9x faster |
| **vtproto** | 36,002 ns | 58,168 B | 1508 | 10.5x faster |
| **hyperpb** | 24,835 ns | 60,053 B | 12 | 15.2x faster |
| **hyperpb + Shared** | 18,163 ns | 21,863 B | 1 | 20.8x faster |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Takeaways: Unmarshaling

Decoding is where generated code really pays off. Standard `protojson` spends significant CPU time matching string keys against descriptors, allocating temporary maps, and resolving types at runtime. Generating direct field assignments cuts out that overhead—running 5x to 7x faster than official `protojson` on these payloads. Heap churn also drops dramatically; unmarshaling the large payload requires only ~1,400 allocations compared to over 5,700 with `protojson`.

The generated JSON decoder beats both `encoding/json` and `encoding/json/v2` across all test payloads. While JSON parsing is fundamentally constrained by being a text format (requiring string tokenization and boundary checks), eliminating runtime reflection makes ProtoJSON performance competitive enough for hot paths where it previously wasn't.

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
    go test -bench=. -benchmem -benchtime=5s -count=5 > results.txt
    ```

The command writes the five raw runs for each benchmark to `results.txt`; the article tables report arithmetic means for `ns/op`, `B/op`, and `allocs/op` computed from those raw rows. For stricter statistical comparison, I would increase the run count and summarize with `benchstat`.

I left `GOMAXPROCS` at Go’s default for this machine, which was 8.

As always with microbenchmarks, the exact numbers matter less than the overall shape of the results. You should benchmark your own schemas and payloads under production-realistic environments before drawing final design conclusions.

---

## Try it Out & Give Feedback!

`protojsonx` is still highly experimental software. It passes the official Protobuf conformance tests, which gives me confidence in its core parsing and serialization logic, but don't run it in critical production paths without thorough testing.

If you are serving JSON APIs backed by Protobuf and `protojson` is showing up in your profiles, I’d love for you to try it against your own schemas.

I’m especially interested in results from real production-shaped messages: large repeated fields, maps, oneofs, well-known types, custom `json_name` usage, and other cases that are more interesting than simple benchmark payloads.

You can find the code and instructions on GitHub:
* **GitHub Repository**: [sudorandom/protojsonx](https://github.com/sudorandom/protojsonx)

Please file issues or share benchmark results there. The more weird schemas, the better.
