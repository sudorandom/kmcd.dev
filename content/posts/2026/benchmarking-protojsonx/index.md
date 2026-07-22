---
title: "Beating Go's encoding/json with Schema-Guided ProtoJSON"
date: "2026-07-21T15:20:00Z"
categories: ["article"]
tags: ["protobuf", "go", "performance", "json", "protojsonx"]
description: "I built protojsonx to measure how much of Go’s ProtoJSON overhead comes from runtime reflection."
cover: "cover.jpg"
images: ["/posts/benchmarking-protojsonx/cover.jpg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
slug: "benchmarking-protojsonx"
type: "posts"
devtoSkip: true
---

In my [previous article](/posts/hidden-cost-of-google-protobuf-value/), benchmark results showed that Go’s official `protojson` package was surprisingly slow. Execution profiles consistently highlighted the same bottlenecks: reflection, pointer chasing, and descriptor traversal accounted for much of the latency and memory allocation.

ProtoJSON must preserve Protobuf semantics, including custom field names, enums, presence rules, and message structure, while producing standard JSON. The official implementation uses `protoreflect` to dynamically walk descriptors and inspect message structures on every call.

That raised a concrete question: *how much of this cost is inherent to JSON formatting, and how much comes from repeatedly resolving schema mappings at runtime?*

To answer this, I built [protojsonx](https://github.com/sudorandom/protojsonx). Moving schema resolution out of the hot path brings ProtoJSON marshaling performance close to binary protobuf and ahead of both Go's standard `encoding/json` and `encoding/json/v2` in these benchmarks.

{{< github-repo repo="sudorandom/protojsonx" description="An experimental faster ProtoJSON encoder and decoder for Go." >}}

---

## Moving Schema Work Out of the Hot Path

The official implementation discovers how to map each protobuf field to JSON while processing the message. `protojsonx` resolves that mapping once and stores the result as a precomputed schema layout table.

It uses that precomputed schema information in two ways:

* **Runtime Table Mode:** Builds flat schema layout tables once at startup initialization. This allows it to marshal and unmarshal messages using direct offset arithmetic instead of per-call descriptor traversal.
* **Generated Plugin Mode:** Provides a `protoc` plugin (`protoc-gen-go-protojsonx`) that bakes type-specific marshaling and unmarshaling methods directly into generated `.pb.go` code, bypassing the runtime lookup layer entirely.

`protojsonx` is designed as a mostly API-compatible experimental alternative to Go's official `google.golang.org/protobuf/encoding/protojson` package. It handles common rules including custom `json_name`, enums, field presence, unknown fields, oneofs, maps, and standard options (though experimental features like dynamic `Any` resolver callbacks are not yet supported).

> **Warning:** `protojsonx` is experimental. It passes the official Protobuf conformance suite, but it has not yet seen enough production use for me to recommend it as a drop-in replacement.

---

## Benchmark Setup

These benchmarks do not compare identical semantics. They answer a practical question: when an application needs ProtoJSON-compatible output, how expensive is that compared with Go’s general-purpose JSON packages?

### Payloads and Sizes

I used the benchmark suite from my previous article, running on Go 1.26 on an Apple M1 Pro across three payload shapes:

* **Small:** A flat object with 4 fields (string ID, status boolean, age integer, score float).
* **Medium:** A nested user signup event containing an actor object, string tags, and a metadata map.
* **Large:** An array repeating the Medium object 100 times.

Serialization output size matters, especially when comparing JSON and binary protobuf:

| Payload | ProtoJSON bytes | generic JSON bytes | binary proto bytes |
| :--- | ---: | ---: | ---: |
| Small | 55 B | 55 B | 25 B |
| Medium | 293 B | 291 B | 162 B |
| Large | 29,412 B | 29,201 B | 16,500 B |

The ProtoJSON and generic JSON sizes are nearly identical. Binary protobuf is significantly smaller, so binary numbers serve as a compact wire-format baseline rather than a direct format equivalent.

### Compared Implementations

All tests measure end-to-end marshal and unmarshal execution times along with allocations. 

* The generic JSON cases (`encoding/json` and `encoding/json/v2` from `github.com/go-json-experiment/json`) use native Go structs with equivalent fields.
* The Protobuf cases use standard generated messages.
* I also included `hyperpb` (a descriptor/layout-driven dynamic protobuf parser built around read-oriented offset decoding) and `hyperpb.Shared` (where the benchmark reuses an arena buffer). While `hyperpb` is not a ProtoJSON library, its numbers provide context on raw binary parsing overhead.

<details>
<summary><b>Show environment details and test command</b></summary>

* **Go Version**: `go version go1.26.3 darwin/arm64`
* **Machine Details**: Apple M1 Pro (10-core CPU, 16GB unified memory, macOS), `GOMAXPROCS=8`
* **Target Library Version**: `github.com/sudorandom/protojsonx@v0.0.6`
* **Benchmark Commit**: `b8fff78c`
* **Benchmark Source**: Available in [benchmarks/](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/benchmarking-protojsonx/benchmarks/)
* **Test Execution Command**:
    ```bash
    go test -bench=. -benchmem -benchtime=5s -count=5 > results.txt
    ```

The tables report arithmetic means for `ns/op`, `B/op`, and `allocs/op` computed from the raw five-run output.

</details>

---

## Marshaling Performance

Marshaling starts with an already constructed Go message, so the encoder mainly needs to walk its fields and write JSON bytes.

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

### Interpreting Marshaling Results

Precomputing schema layout tables eliminates most of official `protojson`'s marshaling overhead. Both `protojsonx` modes reduce marshaling to one heap allocation in these benchmarks (64 B for small, 320 B for medium, and ~32 KB for large).

Across all payload sizes, runtime table mode gets surprisingly close to the generated plugin mode. The difference is 40 ns for the small message (142 ns vs 102 ns), and about 11% for the large case (31.1 µs vs 27.6 µs). With descriptor traversal gone, the remaining gap likely comes from interpreting the table at runtime instead of executing generated field-specific code. Projects with many generated message types will pay for that extra speed through additional generated code and a larger binary.

One result unrelated to `protojsonx` also stood out: `encoding/json/v2` was consistently slower than v1 during marshaling (for example, 62 µs vs 41 µs on large payloads). That may reflect additional state tracking and the costs of its more flexible implementation, though the benchmark does not isolate the cause.

On the large payload, generated `protojsonx` took 27.6 µs, compared with 25.1 µs for `proto.Marshal`. I did not expect JSON encoding to get that close. For this payload, once descriptor traversal was removed, writing field names and formatted values added surprisingly little time over binary encoding.

---

## Unmarshaling Performance

Marshaling showed that precomputed schema information eliminates most of the official implementation's overhead. Decoding is a tougher test because avoiding reflection does not remove JSON tokenization, string parsing, or message construction.

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

### Interpreting Unmarshaling Results

The generated decoder pulls further ahead than I expected. Runtime tables are almost twice as slow on the small payload (267 ns vs 136 ns) and remain about 34% slower on the large one (111.1 µs vs 73.3 µs). When parsing JSON dynamically, runtime table mode looks up field mappings in a table for every incoming key, whereas generated code emits direct message-specific key matching and field assignments.

The allocation count initially looks disappointing. Generated `protojsonx` still performs 1,407 allocations on the large payload. In this benchmark, decoding must allocate the nested messages, slices, strings, and maps that make up the result. Many of the remaining allocations therefore belong to constructing the output message graph rather than resolving its schema.

`encoding/json/v2` performed much better than v1 during decoding, nearly matching runtime-table `protojsonx` on the large payload (108.8 µs vs 111.1 µs). Generated `protojsonx` remained faster (73.3 µs), which is consistent with its ability to emit message-specific field matching and assignment code without generic reflection overhead.

---

## Conclusion

These benchmarks suggest that much of Go’s official `protojson` cost comes from resolving schemas at runtime rather than from JSON formatting alone. Precomputed tables recover most of the marshaling performance, while generated code has a larger advantage during decoding, where field matching happens for every JSON key.

That does not make JSON equivalent to binary protobuf. Tokenization, number parsing, and constructing the destination message still cost time and allocations. But the results show that ProtoJSON can be substantially faster than Go’s current general-purpose implementation.

---

## What I Need Tested Next

If you are serving JSON APIs backed by Protobuf and `protojson` is showing up in your profiles, run the benchmarks against your own schemas and payloads before choosing an implementation.

I’m especially interested in benchmark results from real production schemas: large repeated fields, maps, oneofs, well-known types, custom `json_name` usage, and edge cases beyond standard benchmark structs.

Check out the project on GitHub: [sudorandom/protojsonx](https://github.com/sudorandom/protojsonx).

The more weird schemas, the better.
