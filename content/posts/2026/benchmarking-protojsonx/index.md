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

In my [previous article](/posts/hidden-cost-of-google-protobuf-value/), I explored the performance bottlenecks of using dynamic JSON structures like `google.protobuf.Value` and official `protojson` in Go. The benchmark results pointed at the same culprit over and over: reflection, pointer chasing, and descriptor traversal were showing up as real latency and allocation costs.

ProtoJSON sits in an awkward place. It gives Protobuf-backed APIs a JSON representation that human operators, browsers, API gateways, and debugging tools can work with, but Go’s official `protojson` implementation pays heavily for this generality. To resolve field names, enum values, presence semantics, and message structure at runtime, it has to walk message descriptors using Protobuf reflection (`protoreflect`) and allocate intermediate values.

This is the right tradeoff for correctness and compatibility. But it made me wonder: *how much of this cost is fundamental to ProtoJSON, and how much comes from repeatedly resolving schema mappings at runtime?*

So I built [protojsonx](https://github.com/sudorandom/protojsonx), partly as an experiment and partly because I wanted to know whether ProtoJSON was inherently slow or whether Go’s implementation was paying too much runtime bookkeeping cost. It implements two strategies: compiling descriptors into flat offset layout tables once at startup (**Runtime Table Mode**), and generating static, reflection-free parsing routines via a protoc plugin (**Generated Plugin Mode**).

In this post, we'll look at the benchmarks comparing standard `protojson`, raw struct-based JSON serializers (`encoding/json` and `encoding/json/v2` currently living at [`github.com/go-json-experiment/json`](https://github.com/go-json-experiment/json)), `protojsonx` in both modes, and raw binary Protobuf (`proto` and the optimized reflection-free `vtproto`). The interesting part is that moving schema work out of the hot path gets ProtoJSON much closer to ordinary JSON and binary protobuf than I expected.

{{< github-repo repo="sudorandom/protojsonx" description="An experimental faster ProtoJSON encoder and decoder for Go." >}}

> **Warning:** `protojsonx` is highly experimental at this stage. It does pass the official Protobuf conformance tests, which gives me confidence that it follows the expected ProtoJSON behavior, but I would still treat it as early-stage software.

---

## What is protojsonx?

[protojsonx](https://github.com/sudorandom/protojsonx) is designed as a mostly API-compatible experimental replacement for Go's official `google.golang.org/protobuf/encoding/protojson` library. It supports core ProtoJSON behaviors like field names, enums, presence semantics, unknown-field handling, `json_name`, oneofs, maps, and standard marshaling/unmarshaling options. However, it does not yet cover every edge case or configuration option of the official library (such as dynamic `Any` resolving).

To make serialization faster, it implements two key optimization strategies:

*   **Runtime Table Mode**: This mode implements the startup compilation strategy. It builds flat offset tables at initialization, allowing it to marshal and unmarshal structures using fast, sequential offset arithmetic instead of per-call descriptor traversal.
*   **Generated Plugin Mode**: For the highest-performance path, `protojsonx` provides a protoc plugin (`protoc-gen-go-protojsonx`) that generates type-specific marshaling and unmarshaling methods directly. At that point, a lot of the runtime bookkeeping disappears. The remaining cost is less about “what field is this?” and more about the unavoidable work of reading or writing JSON.

---

## The Benchmark Setup

I used the benchmark suite from my previous article, running on Go 1.26 on an Apple M1 Pro. The configurations are:
* **Small:** A flat object with 4 fields (string ID, status boolean, age integer, score float).
* **Medium:** A nested user signup event containing actor object, string tags, and metadata map.
* **Large:** An array repeating the Medium object 100 times.

### The Rules of the Game

All tests use equivalent payload shapes and measure end-to-end marshal/unmarshal cost, including allocations. The generic JSON cases use plain Go structs with similar fields, while the protobuf cases use generated protobuf messages.

This is not a perfect apples-to-apples comparison. ProtoJSON has extra rules around field names, presence, enums, well-known types, and numeric encoding. The point is narrower: if you already need ProtoJSON-shaped output, how much does that cost compared with the JSON tools Go developers normally reach for? These benchmarks compare the cost of serving similar application-shaped JSON payloads, not identical semantics.

I also included `hyperpb`, a descriptor/layout-driven dynamic protobuf parser built around read-oriented offset decoding, including `hyperpb.Shared` where the benchmark can reuse its arena. It is not a ProtoJSON library, so its numbers should be read as a binary protobuf reference point rather than a direct competitor.

### Payload Sizes

Serialization output size matters, especially when comparing JSON and binary protobuf. These are the payload sizes produced by the benchmark fixtures:

| Payload | ProtoJSON bytes | generic JSON bytes | binary proto bytes |
| :--- | ---: | ---: | ---: |
| Small | 55 B | 55 B | 25 B |
| Medium | 293 B | 291 B | 162 B |
| Large | 29,412 B | 29,201 B | 16,500 B |

The ProtoJSON and generic JSON sizes are close, but not byte-identical. Binary protobuf is much smaller for these payloads, so the binary rows should be read as a compact binary reference point rather than a JSON-format comparison.

### Headline Results

Before the full tables, here is the short version for the fastest JSON path in each group:

| Operation | Best JSON option | Compared with official protojson | Compared with encoding/json |
| :--- | :--- | ---: | ---: |
| Small marshal | protojsonx generated | 6.0x faster | 1.8x faster |
| Medium marshal | protojsonx generated | 7.0x faster | 1.6x faster |
| Large marshal | protojsonx generated | 8.1x faster | 1.5x faster |
| Small unmarshal | protojsonx generated | 6.9x faster | 5.5x faster |
| Medium unmarshal | protojsonx generated | 5.1x faster | 4.1x faster |
| Large unmarshal | protojsonx generated | 5.1x faster | 3.7x faster |

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
| **encoding/json** | 183 ns | 64 B | 1 | - |
| **encoding/json/v2** | 289 ns | 112 B | 2 | - |
| **protojsonx (Runtime Tables)** | **142 ns** | **64 B** | **1** | **4.3x faster** |
| **protojsonx (Generated Plugin)** | **102 ns** | **64 B** | **1** | **6.0x faster** |
| **proto.Marshal** | 83 ns | 32 B | 1 | - |
| **vtproto** | 25 ns | 32 B | 1 | - |
| **hyperpb + Shared** | 296 ns | 144 B | 7 | - |

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
| **encoding/json** | 521 ns | 464 B | 2 | - |
| **encoding/json/v2** | 803 ns | 608 B | 3 | - |
| **protojsonx (Runtime Tables)** | **404 ns** | **320 B** | **1** | **5.5x faster** |
| **protojsonx (Generated Plugin)** | **318 ns** | **320 B** | **1** | **7.0x faster** |
| **proto.Marshal** | 285 ns | 176 B | 1 | - |
| **vtproto** | 103 ns | 176 B | 1 | - |
| **hyperpb + Shared** | 1,062 ns | 744 B | 17 | - |

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
| **encoding/json** | 41,250 ns | 32,823 B | 2 | - |
| **encoding/json/v2** | 62,036 ns | 32,856 B | 3 | - |
| **protojsonx (Runtime Tables)** | **31,134 ns** | **32,797 B** | **1** | **7.2x faster** |
| **protojsonx (Generated Plugin)** | **27,601 ns** | **32,784 B** | **1** | **8.1x faster** |
| **proto.Marshal** | 25,131 ns | 18,432 B | 1 | - |
| **vtproto** | 7,575 ns | 18,432 B | 1 | - |
| **hyperpb + Shared** | 103,495 ns | 107,216 B | 1022 | - |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Takeaways: Marshaling

Most of the marshaling performance gain comes from compiling descriptors up-front. Standard `protojson` spends its cycles querying Go’s reflection and descriptor trees on the fly, creating a continuous stream of allocations. Compiling those layout mappings once at startup (**Runtime Table Mode**) allows the serialization process to bypass runtime lookup loops entirely, converting the fields to JSON via sequential offset math.

The generated plugin goes one step further. Instead of building layout tables at startup, it writes the field access code directly into the generated Go file. That removes another layer of lookup and assertion work from the hot path.

The tradeoff is the usual one for generated code: in a large schema repository, generating specialized JSON routines for hundreds of message types can increase compiled binary size.

For these fixtures, schema-guided ProtoJSON marshaling beats both `encoding/json` and `encoding/json/v2`, and the generated encoder lands close to standard binary `proto.Marshal`. `hyperpb + Shared` is slower on marshal here, which fits its design: it is much more interesting as a read-oriented parser than as a serializer.

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
| **encoding/json** | 752 ns | 280 B | 6 | - |
| **encoding/json/v2** | 339 ns | 48 B | 1 | - |
| **protojsonx (Runtime Tables)** | **267 ns** | **96 B** | **3** | **3.5x faster** |
| **protojsonx (Generated Plugin)** | **136 ns** | **96 B** | **2** | **6.9x faster** |
| **proto.Unmarshal** | 114 ns | 96 B | 2 | - |
| **vtproto** | 25 ns | 16 B | 1 | - |
| **hyperpb** | 360 ns | 798 B | 4 | - |
| **hyperpb + Shared** | 125 ns | 65 B | 1 | - |

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
| **encoding/json** | 2,937 ns | 688 B | 19 | - |
| **encoding/json/v2** | 1,133 ns | 256 B | 4 | - |
| **protojsonx (Runtime Tables)** | **1,108 ns** | **576 B** | **16** | **3.3x faster** |
| **protojsonx (Generated Plugin)** | **720 ns** | **528 B** | **14** | **5.1x faster** |
| **proto.Unmarshal** | 571 ns | 560 B | 15 | - |
| **vtproto** | 322 ns | 432 B | 14 | - |
| **hyperpb** | 638 ns | 1,446 B | 5 | - |
| **hyperpb + Shared** | 290 ns | 357 B | 1 | - |

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
| **encoding/json** | 272,103 ns | 70,584 B | 1216 | - |
| **encoding/json/v2** | 108,811 ns | 54,305 B | 309 | - |
| **protojsonx (Runtime Tables)** | **111,141 ns** | **62,232 B** | **1709** | **3.4x faster** |
| **protojsonx (Generated Plugin)** | **73,389 ns** | **55,008 B** | **1407** | **5.1x faster** |
| **proto.Unmarshal** | 54,923 ns | 58,232 B | 1509 | - |
| **vtproto** | 36,002 ns | 58,168 B | 1508 | - |
| **hyperpb** | 24,835 ns | 60,053 B | 12 | - |
| **hyperpb + Shared** | 18,163 ns | 21,863 B | 1 | - |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Takeaways: Unmarshaling

Decode is where the generated code earns its keep. If you’ve ever looked at Go’s standard `protojson` decoder, it spends a lot of time matching string keys against descriptors, allocating intermediate maps, and resolving types at runtime. Compiling direct field assignments into the generated decoder cuts out a lot of that work: 5.1x to 6.9x faster than official `protojson` on these static payloads. It also reduces heap churn substantially; unmarshaling the Large payload drops from over 5,700 allocations down to just 1,407.

The interesting part is that the generated JSON decoder beats both `encoding/json` and `encoding/json/v2` in these fixtures. It still does not catch binary protobuf, but it gets close enough that ProtoJSON stops looking like an automatic performance disaster.

That said, unmarshaling still has a wider performance gap relative to binary Protobuf than marshaling does. This is a fundamental constraint of the JSON format itself. A binary decoder can skip fields using length prefixes and parse integer types with very little overhead. A JSON decoder, by contrast, is forced to parse string layouts, handle token boundaries, and instantiate nested objects and map fields on the fly. But even with these structural constraints, the reflection-free generated code narrows the gap significantly, turning what used to be a large bottleneck into a much smaller and more predictable cost.

---

## How protojsonx Compares to Generic JSON & Binary Formats

The basic idea held up: **moving schema work out of the hot path can make ProtoJSON faster than Go's standard library `encoding/json` package for these fixtures.**

The **`encoding/json/v2` package also looks very strong.** Under unmarshaling, it cuts a lot of latency and allocation overhead compared with `encoding/json` (e.g. unmarshaling the Large payload drops from 272,103 ns/op down to 108,811 ns/op).

The experimental `encoding/json/v2` package gets close to `protojsonx` Runtime Table Mode, but the generated `protojsonx` path was still the fastest JSON option in these benchmarks.

A rough summary of the benchmark results looks like this:

| Format / Serializer | Role in these benchmarks | Notes |
| :--- | :--- | :--- |
| **Official `protojson`** | Slowest JSON path | Most general and most dynamic |
| **`encoding/json`** | Standard JSON baseline | General-purpose reflection-based JSON |
| **`encoding/json/v2`** | Faster generic JSON baseline | Official v2 prototype; much better on unmarshal |
| **`protojsonx` Runtime Table Mode** | Fast dynamic ProtoJSON | Descriptor work moved to startup |
| **`protojsonx` Generated Plugin Mode** | Fastest JSON path here | Type-specific generated code |
| **Standard binary `proto`** | Binary protobuf baseline | Compact protobuf wire format |
| **`vtproto`** | Fast generated binary baseline | Strongest generated marshal path here |
| **`hyperpb + Shared`** | Binary decode reference | Very fast reusable-storage decode path |

For static-message marshaling, `protojsonx` with the plugin gets much closer to standard binary Protobuf than official `protojson`, and in these benchmarks it beats generic `encoding/json` and `encoding/json/v2`.

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

All benchmarks were run on an otherwise idle machine using a multi-run sequence to reduce noise. All runs were performed on an AC-powered, thermally-settled machine to prevent thermal throttling or low-power state interference. The command writes the five raw runs for each benchmark to `results.txt`; the article tables report arithmetic means for `ns/op`, `B/op`, and `allocs/op` computed from those raw rows. For stricter statistical comparison, I would increase the run count and summarize with `benchstat`.

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
