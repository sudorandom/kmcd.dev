---
title: "The Hidden Cost of google.protobuf.Value"
date: "2026-06-15T10:00:00Z"
categories: ["article"]
tags: ["protobuf", "go", "performance", "json", "software-architecture"]
description: "The hidden performance cost of dynamic Protobuf in Go."
cover: "cover.svg"
images: ["/posts/hidden-cost-of-google-protobuf-value/cover.svg"]
featuredalt: ""
featuredpath: "date"
slug: "hidden-cost-of-google-protobuf-value"
type: "posts"
devtoSkip: true
---

Migrating legacy JSON APIs to gRPC frequently stumbles over a common anti-pattern: unstructured, dynamic JSON fields (such as `metadata` or `extra_properties`) mapped directly into Protobuf using [`google.protobuf.Value`](https://protobuf.dev/reference/protobuf/google.protobuf/#value) or [`google.protobuf.Struct`](https://protobuf.dev/reference/protobuf/google.protobuf/#struct).

These belong to Protobuf's **Well-Known Types (WKTs)**, a library of standardized, common message schemas defined by Google (such as `Timestamp`, `Duration`, and `Any`) that ship out-of-the-box with the protobuf compiler to provide consistent representation for reusable data structures across different languages.

But what actually are these specific dynamic types under the hood?

If we look at their official definitions, `google.protobuf.Value` is defined as a `oneof` containing the possible JSON-compatible data types:

```protobuf
message Value {
  oneof kind {
    NullValue null_value = 1;
    double number_value = 2;
    string string_value = 3;
    bool bool_value = 4;
    Struct struct_value = 5;
    ListValue list_value = 6;
  }
}
```

This structure leads to the following well-known types:
* [`google.protobuf.Value`](https://protobuf.dev/reference/protobuf/google.protobuf/#value) represents a dynamically typed wrapper that can hold any JSON-compatible value: null, a number (encoded as a double-precision float), a string, a boolean, a nested `Struct` (representing a JSON object), or a list of values (`ListValue`, representing a JSON array).
* [`google.protobuf.Struct`](https://protobuf.dev/reference/protobuf/google.protobuf/#struct) represents a structured data value consisting of fields that map to dynamically typed values. In practice, it **behaves like a JSON object**, mapping keys (strings) to `Value` messages:
    ```protobuf
    message Struct {
      map<string, Value> fields = 1;
    }
    ```
    In Go, this translates to a key-value map (`map[string]any`) or a raw JavaScript object where the keys are strings and the values are dynamically typed.

Together, they allow Protobuf messages to carry arbitrary, unstructured JSON-like payloads without declaring a strict schema beforehand. This is a common architectural pattern, and it successfully solves a real developer pain point: handling highly dynamic data. Since Protobuf is famous for being fast and compact, one would intuitively think that wrapping dynamic payloads in these well-known types would still be more efficient than standard JSON.

I decided to test that assumption, and the results were the opposite of what I expected. Most of Protobuf's performance advantage comes from ahead-of-time schema knowledge. Statically compiled schemas are the primary optimization lever that allows Protocol Buffers to achieve speed and compactness. The moment you remove schema information and adopt unstructured types like `google.protobuf.Value` or `google.protobuf.Struct`, you give up many of the optimizations that make Protobuf fast and compact.

## The Structural Cost of Dynamic Data

Before looking at the benchmark numbers, it helps to understand why `google.protobuf.Value` is structurally expensive on the wire compared to compact JSON. When representing arbitrary object structures, `google.protobuf.Struct` is defined under the hood as `map<string, Value> fields = 1;`.

In the Protobuf wire format, maps are not native primitives. Instead, they are represented as a repeated list of auto-generated key-value message entries. The map field is equivalent to:

```protobuf
message MapEntry {
  string key = 1;
  google.protobuf.Value value = 2;
}

// Inside google.protobuf.Struct:
repeated MapEntry fields = 1;
```

This explains why every key-value entry is serialized as a nested sub-message (`MapEntry`) containing two inner fields (`key` and `value`), each with its own tag and length overhead.

For tiny dynamic payloads, the structural metadata overhead of `Struct` can exceed the payload itself. While in practice payloads are enclosed in full objects and messages, comparing the raw serialization layout of a single dynamic key-value entry like `"age": 30` illustrates this overhead:

### Wire Format Layout Comparison

#### 1. Compact JSON (10 bytes)

For this conceptual representation of JSON, containing a single key/value pair for age wrapped in enclosing object braces, we have:

```json
{"age":30}
```

This payload is exactly **10 bytes** in size:
- 2 bytes for the enclosing object braces `{}`
- 5 bytes for the key `"age"` (including quotes)
- 1 byte for the colon separator `:`
- 2 bytes for the numeric characters `"30"`

#### 2. Dynamic Protobuf (Protoscope Representation)

If we describe the dynamic Protobuf binary payload using [Protoscope](https://github.com/protocolbuffers/protoscope) (the language for representing raw Protobuf wire formats) to visualize the structure and tags:

```protoscope
1: {           # Struct.fields map entry header        -> 2 Bytes
  1: "age"     # MapEntry.key (tag/len + "age" string) -> 5 Bytes
  2: {         # MapEntry.value Value wrapper header   -> 2 Bytes
    2: 30.0    # Value.number_value double field       -> 9 Bytes
  }
}
```

- Each set of braces `{}` represents a length-delimited sub-message, which compiles to a field tag followed by a length prefix byte on the wire.
- The prefix tags `1:` and `2:` represent the field numbers.
- `30.0` compiles to field tag 2 (wire type 1, 64-bit) followed by its fixed 8-byte double-precision float value.

Summing these up (2B + 5B + 2B + 9B) results in a total payload size of exactly 18 bytes for this specific encoding.

So with this context, we can now say *why* `google.protobuf.Value` is inefficient:

1. **Double Nesting and Header Overhead:** In compact JSON, the valid payload `{"age":30}` consumes 10 bytes total. In dynamic Protobuf, the nested schema-less structure requires 18 bytes. The nested header metadata alone (the field tags and length prefixes for each layer) consumes **7 bytes**, which is nearly as large as the entire JSON payload before any key or value content is even written.

2. **No Field Name Compression**: One of Protobuf's largest size advantages usually comes from discarding human-readable field names (like `"age"`) and replacing them with compact, 1-byte numeric tags. However, because `google.protobuf.Struct` is unstructured, it must serialize the actual field name string `"age"` on the wire. This completely forfeits the field-name compression benefit that makes static Protobuf so compact.

3. **No Varint Compression for Numbers**: Instead of benefiting from Protobuf's specialized integer encodings, all numeric values are represented through a double-precision floating-point field. Small integers suffer most because Protobuf normally compresses them using varints. Large values may see less dramatic differences because JSON must serialize every digit as text. To visualize the wire layout comparison for representing the small number `30`:

```text
Compact JSON ("30"):
00110011 00110000 -> 2 bytes

Static Protobuf (Varint 30):
00011110 -> 1 byte

Dynamic Protobuf (30.0 double-precision float):
00000000 00000000 00000000 00000000 
00000000 00000000 00111110 01000000 -> 8 bytes (little-endian)
```

As a result of this multiple-nesting structure and fixed-size floats, dynamic Protobuf payloads often end up larger on the wire than compact JSON. However, this result is heavily workload-dependent. The biggest penalties come from having many fields, many keys, small numeric values, and deeply nested structures. A payload consisting mostly of a single massive string will not exhibit the same structural overhead.

### Wire Inefficiency vs. Runtime Inefficiency

Dynamic Protobuf hurts in two completely different ways that affect different engineering decisions:

1. **Wire Inefficiency:** The serialized payload becomes larger than many developers expect. This is caused by human-readable field names being serialized repeatedly, nested map entry encoding, double-precision floating-point storage for all numbers, and the loss of varint integer compression. Bandwidth-sensitive systems, databases, or event brokers care heavily about this.
2. **Runtime Inefficiency:** The runtime representation in Go becomes allocation-heavy and expensive to parse. This is caused by Go's allocation behavior, interface-heavy and pointer-heavy structures in the standard `structpb` package, Go's reflection model, and tree-shaped decoding. CPU-bound services that deserialize payloads frequently care heavily about this.

## The Benchmark Setup

I built a Go benchmark comparing standard JSON against various Protobuf strategies across three payload sizes. The complete benchmark suite and Go test code are available in the [sudorandom/kmcd.dev](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/google-protobuf-value-considered-harmful/benchmarks) repository on GitHub.

The payload configurations are:
* **Small:** A flat object with 4 fields (string ID, status boolean, age integer, score float).
* **Medium:** A nested user signup event containing an actor object, string tags, and a metadata map.
* **Large:** An array repeating the Medium object 100 times.

This benchmark focuses specifically on the tradeoff between schema-less Protobuf WKTs and common Go JSON implementations rather than surveying all binary serialization formats (like MessagePack, CBOR, BSON, Avro, or FlatBuffers).

### Benchmark Variants

To evaluate performance across different serialization models, I compared the following variants:

| Variant | Format | Description |
| :--- | :---: | :--- |
| **Concrete (JSON)** | JSON | Serializes a standard concrete Go struct using Go's standard [`encoding/json`](https://pkg.go.dev/encoding/json) library (JSON v1). |
| **Concrete (JSONv2)** | JSON | Serializes a standard concrete Go struct using the experimental, higher-performance [`github.com/go-json-experiment/json`](https://pkg.go.dev/github.com/go-json-experiment/json) library (JSON v2). |
| **Map (JSON)** | JSON | Serializes a generic, schema-less Go map (`map[string]any`) using the standard [`encoding/json`](https://pkg.go.dev/encoding/json) library. |
| **Map (JSONv2)** | JSON | Serializes a generic, schema-less Go map (`map[string]any`) using the experimental [`github.com/go-json-experiment/json`](https://pkg.go.dev/github.com/go-json-experiment/json) library. |
| **Concrete (proto)** | Protobuf | Serializes statically generated Protobuf messages using Go's official [`google.golang.org/protobuf/proto`](https://pkg.go.dev/google.golang.org/protobuf/proto) library. |
| **Concrete (vtproto)** | Protobuf | Serializes statically generated Protobuf messages using PlanetScale's optimized, reflection-free [`vtproto`](https://github.com/planetscale/vtproto) generator. |
| **google.protobuf.Any (proto)** | Protobuf | Serializes static Protobuf messages wrapped in a dynamic, polymorphic Well-Known Type [`google.protobuf.Any`](https://protobuf.dev/reference/protobuf/google.protobuf/#any). |
| **google.protobuf.Value (proto)** | Protobuf | Serializes schema-less dynamic payloads using the Well-Known Type [`google.protobuf.Value`](https://protobuf.dev/reference/protobuf/google.protobuf/#value) ([`structpb`](https://pkg.go.dev/google.golang.org/protobuf/types/known/structpb)). |
| **Protobuf + JSON** | Protobuf | Bypasses the dynamic WKT wrapper by storing the raw serialized JSON string directly inside an opaque Protobuf string/bytes field (Opaque JSON Packaging). |
| **Concrete (JSONProto)** | JSON | Serializes statically generated Protobuf messages into JSON format using Go's official [`protojson`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson) encoder. |
| **google.protobuf.Value (JSONProto)** | JSON | Serializes dynamic [`google.protobuf.Value`](https://protobuf.dev/reference/protobuf/google.protobuf/#value) payloads into JSON format using Go's official [`protojson`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson) encoder. |
| **google.protobuf.Any (JSONProto)** | JSON | Serializes polymorphic [`google.protobuf.Any`](https://protobuf.dev/reference/protobuf/google.protobuf/#any) wrappers into JSON format using Go's official [`protojson`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson) encoder. |

**A Note on `Any` vs `Value`:**
It is worth noting that `Any` and `Value` solve different problems. `Any` is not a schema-less alternative; it is a schema-dispatch mechanism. `Any` assumes a schema exists and the consumer knows it, while `Value` assumes no schema exists at all. The comparison is useful because teams often reach for `Value` when their actual requirement is polymorphism rather than truly schema-less data.

**A Note on Payload Decoding in Benchmarks:**
To ensure a fair, apples-to-apples performance comparison, the unmarshaling benchmarks for all variants fully deserialize both the outer envelope and the inner dynamic/polymorphic payloads:
* In the **`google.protobuf.Any`** benchmark, the payload is not left as raw bytes; it is fully unpacked into a concrete statically-compiled Go struct using `anypb.UnmarshalTo()`.
* In the **`Protobuf + JSON`** benchmark, the inner JSON string is not left unparsed; it is fully deserialized into a concrete Go struct using standard `json.Unmarshal()`.
* In the **`google.protobuf.Value`** benchmark, the payload is fully parsed into a tree of Go objects representing the JSON-like data.

Importantly, in real-world applications, both `google.protobuf.Any` and opaque `Protobuf + JSON` packaging allow you to bypass this inner parsing entirely on intermediate routing nodes (deferred/lazy parsing). This is a significant architectural advantage if intermediate services only need to forward or store the payload without inspecting it. However, to maintain a level playing field and measure the actual parsing cost, these benchmarks force full decoding.

To handle arbitrary data, the dynamic Protobuf configurations rely on standard `structpb` definitions:

```protobuf
syntax = "proto3";
package event;

import "google/protobuf/struct.proto";

message EventEnvelope {
  string id = 1;
  int64 timestamp = 2;
  google.protobuf.Value payload = 3; // Dynamic payload field
}
```

### Benchmark Disclaimer and Workload Caveats

As with all performance testing, microbenchmarks should be taken with a grain of salt. Actual performance will vary depending on your specific hardware, operating system, compiler version, garbage collection settings, and payload structure. Workload shape and configuration matter enormously. Map-heavy workloads are particularly pathological for `google.protobuf.Struct` due to Go's map lookup and insertion overhead. Deeply nested objects increase allocation and traversal costs substantially, whereas flat structures suffer less. Furthermore, implementation details like hot-path struct reuse or memory pooling (such as a thread-local arena pool solution using dynamic parser bytecode) can dramatically change outcomes in production, shifting the bottleneck back toward wire serialization and parsing logic.

Microbenchmarks represent synthetic workloads and may not perfectly translate to the performance profile of a complex, production system. You should always run benchmarks under your own representative workloads before making significant architectural decisions.

This article focuses specifically on Go's protobuf implementation and the `structpb` runtime model. Other languages may exhibit different allocation and parsing characteristics, though the wire-format overhead discussed here remains universal.

Additionally, dynamic Protobuf is not always a poor choice. For admin panels, configuration APIs, low-volume integrations, or systems where schema flexibility matters more than throughput, `google.protobuf.Value` remains a perfectly reasonable choice. It is only when these structures sit directly on high-throughput hot paths that performance problems emerge.

---

## Benchmark Results

The benchmarks were executed under Go 1.26 on an Apple M1 Pro.

### Wire Size

First, I compared the serialized payload size of each configuration. Since Protobuf is widely recognized as a highly compact binary protocol, developers often assume that even unstructured dynamic payloads utilizing `google.protobuf.Value` will naturally be smaller on the wire than standard JSON. Measuring serialized byte sizes is a straightforward test that yields definitive, objective results.

*Note: These measurements reflect raw serialized payload size before transport-level compression. Systems using gzip or zstd may observe different relative wire sizes. Repeated JSON keys compress extremely well, though Protobuf map entries and repeated keys also benefit heavily from transport compression.*

{{< tabs >}}
  {{< tab name="Small Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Concrete (proto)",
      "google.protobuf.Value (proto)",
      "google.protobuf.Any (proto)",
      "Protobuf + JSON",
      "Concrete (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)",
      "google.protobuf.Any (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Small Payload (Bytes)",
        "data": [25, 74, 74, 57, 55, 55, 55, 111],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(186, 85, 211, 1)"
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
        "text": "Serialized Data Size (Small Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "proto", "color": "rgba(0, 191, 255, 0.75)" },
          { "text": "json", "color": "rgba(255, 165, 0, 0.75)" },
          { "text": "jsonproto", "color": "rgba(186, 85, 211, 0.75)" }
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

| Format / Config (Small Payload) | Serialized Size | % of JSON (lower is better) |
| :--- | :---: | :---: |
| **Concrete (JSON)** | 55 B | 100.0% (Baseline) |
| **Concrete (proto)** / **Concrete (vtproto)** | 25 B | **45.5%** |
| **Protobuf + JSON** | 57 B | 103.6% |
| **google.protobuf.Any (proto)** | 74 B | 134.5% |
| **google.protobuf.Value (proto)** | 74 B | 134.5% |
| **Concrete (JSONProto)** | 55 B | 100.0% |
| **google.protobuf.Value (JSONProto)** | 55 B | 100.0% |
| **google.protobuf.Any (JSONProto)** | 111 B | 201.8% |

</details>
  {{< /tab >}}
  {{< tab name="Medium Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Concrete (proto)",
      "google.protobuf.Value (proto)",
      "google.protobuf.Any (proto)",
      "Protobuf + JSON",
      "Concrete (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)",
      "google.protobuf.Any (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Medium Payload (Bytes)",
        "data": [162, 328, 212, 294, 291, 293, 291, 349],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(186, 85, 211, 1)"
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
        "text": "Serialized Data Size (Medium Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "proto", "color": "rgba(0, 191, 255, 0.75)" },
          { "text": "json", "color": "rgba(255, 165, 0, 0.75)" },
          { "text": "jsonproto", "color": "rgba(186, 85, 211, 0.75)" }
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

| Format / Config (Medium Payload) | Serialized Size | % of JSON (lower is better) |
| :--- | :---: | :---: |
| **Concrete (JSON)** | 291 B | 100.0% (Baseline) |
| **Concrete (proto)** / **Concrete (vtproto)** | 162 B | **55.7%** |
| **google.protobuf.Any (proto)** | 212 B | **72.9%** |
| **Protobuf + JSON** | 294 B | 101.0% |
| **google.protobuf.Value (proto)** | 328 B | 112.7% |
| **Concrete (JSONProto)** | 293 B | 100.7% |
| **google.protobuf.Value (JSONProto)** | 291 B | 100.0% |
| **google.protobuf.Any (JSONProto)** | 349 B | 119.9% |

</details>
  {{< /tab >}}
  {{< tab name="Large Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Concrete (proto)",
      "google.protobuf.Value (proto)",
      "google.protobuf.Any (proto)",
      "Protobuf + JSON",
      "Concrete (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)",
      "google.protobuf.Any (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Large Payload (Bytes)",
        "data": [16500, 33104, 21200, 29205, 29201, 29412, 29201, 34900],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(186, 85, 211, 1)"
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
        "text": "Serialized Data Size (Large Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "proto", "color": "rgba(0, 191, 255, 0.75)" },
          { "text": "json", "color": "rgba(255, 165, 0, 0.75)" },
          { "text": "jsonproto", "color": "rgba(186, 85, 211, 0.75)" }
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

| Format / Config (Large Payload) | Serialized Size | % of JSON (lower is better) |
| :--- | :---: | :---: |
| **Concrete (JSON)** | 29,201 B | 100.0% (Baseline) |
| **Concrete (proto)** / **Concrete (vtproto)** | 16,500 B | **56.5%** |
| **google.protobuf.Any (proto)** | 21200 B | **72.6%** |
| **Protobuf + JSON** | 29,205 B | 100.0% |
| **google.protobuf.Value (proto)** | 33,104 B | 113.4% |
| **Concrete (JSONProto)** | 29,412 B | 100.7% |
| **google.protobuf.Value (JSONProto)** | 29,201 B | 100.0% |
| **google.protobuf.Any (JSONProto)** | 34,900 B | 119.5% |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Processing Throughput
Building and parsing schema-less Protobuf trees involves significant pointer-wrapping overhead, resulting in higher CPU usage and frequent heap allocations. Standard concrete Protobuf marshals almost instantly, and PlanetScale's reflection-free generator `Concrete (vtproto)` is the absolute fastest. Much of `vtproto`'s advantage comes from eliminating reflection and generating specialized straight-line serialization code ahead of time.

{{< tabs >}}
  {{< tab name="Small Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Concrete (vtproto)",
      "Concrete (proto)",
      "Concrete (JSON)",
      "google.protobuf.Any (proto)",
      "Protobuf + JSON",
      "Concrete (JSONv2)",
      "Map (JSON)",
      "Concrete (JSONProto)",
      "Map (JSONv2)",
      "google.protobuf.Value (proto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Small Payload (ns/op)",
        "data": [31, 104, 210, 287, 365, 365, 706, 761, 950, 2163, 3017],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(186, 85, 211, 1)"
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
        "text": "Marshalling Performance (Small Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "proto", "color": "rgba(0, 191, 255, 0.75)" },
          { "text": "json", "color": "rgba(255, 165, 0, 0.75)" },
          { "text": "jsonproto", "color": "rgba(186, 85, 211, 0.75)" }
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
| **Concrete (vtproto)** | **31 ns** | **32 B** | **1** |
| **Concrete (proto)** | 104 ns | 32 B | 1 |
| **Concrete (JSON)** | 210 ns | 64 B | 1 |
| **google.protobuf.Any (proto)** | 287 ns | 240 B | 4 |
| **Protobuf + JSON** | 365 ns | 256 B | 4 |
| **Concrete (JSONv2)** | 365 ns | 112 B | 2 |
| **Map (JSON)** | 706 ns | 352 B | 10 |
| **Concrete (JSONProto)** | 761 ns | 512 B | 12 |
| **Map (JSONv2)** | 950 ns | 151 B | 9 |
| **google.protobuf.Value (proto)** | 2,163 ns | 879 B | 22 |
| **google.protobuf.Value (JSONProto)** | 3,017 ns | 1,364 B | 35 |

</details>
  {{< /tab >}}
  {{< tab name="Medium Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Concrete (vtproto)",
      "Concrete (proto)",
      "google.protobuf.Any (proto)",
      "Concrete (JSON)",
      "Protobuf + JSON",
      "Concrete (JSONv2)",
      "Map (JSONv2)",
      "Map (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (proto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Medium Payload (ns/op)",
        "data": [129, 366, 597, 656, 855, 1002, 1852, 2273, 2739, 6854, 10177],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(186, 85, 211, 1)"
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
        "text": "Marshalling Performance (Medium Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "proto", "color": "rgba(0, 191, 255, 0.75)" },
          { "text": "json", "color": "rgba(255, 165, 0, 0.75)" },
          { "text": "jsonproto", "color": "rgba(186, 85, 211, 0.75)" }
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
| **Concrete (vtproto)** | **129 ns** | **176 B** | **1** |
| **Concrete (proto)** | 366 ns | 176 B | 1 |
| **google.protobuf.Any (proto)** | 597 ns | 528 B | 4 |
| **Concrete (JSON)** | 656 ns | 464 B | 2 |
| **Protobuf + JSON** | 855 ns | 1,024 B | 4 |
| **Concrete (JSONv2)** | 1,002 ns | 608 B | 3 |
| **Map (JSONv2)** | 1,852 ns | 456 B | 12 |
| **Map (JSON)** | 2,273 ns | 1,200 B | 28 |
| **Concrete (JSONProto)** | 2,739 ns | 1,722 B | 34 |
| **google.protobuf.Value (proto)** | 6,854 ns | 2,959 B | 68 |
| **google.protobuf.Value (JSONProto)** | 10,177 ns | 4,977 B | 113 |

</details>
  {{< /tab >}}
  {{< tab name="Large Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Concrete (vtproto)",
      "Concrete (proto)",
      "Concrete (JSON)",
      "google.protobuf.Any (proto)",
      "Protobuf + JSON",
      "Concrete (JSONv2)",
      "Map (JSONv2)",
      "Map (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (proto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Large Payload (ns/op)",
        "data": [9065, 31060, 50746, 59813, 61323, 76945, 107436, 236591, 279812, 680700, 988728],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(186, 85, 211, 1)"
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
        "text": "Marshalling Performance (Large Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "proto", "color": "rgba(0, 191, 255, 0.75)" },
          { "text": "json", "color": "rgba(255, 165, 0, 0.75)" },
          { "text": "jsonproto", "color": "rgba(186, 85, 211, 0.75)" }
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
| **Concrete (vtproto)** | **9,065 ns** | **18,432 B** | **1** |
| **Concrete (proto)** | 31,060 ns | 18,432 B | 1 |
| **Concrete (JSON)** | 50,746 ns | 32,831 B | 2 |
| **google.protobuf.Any (proto)** | 59,813 ns | 52,800 B | 400 |
| **Protobuf + JSON** | 61,323 ns | 98,928 B | 4 |
| **Concrete (JSONv2)** | 76,945 ns | 32,837 B | 3 |
| **Map (JSONv2)** | 107,436 ns | 35,275 B | 303 |
| **Map (JSON)** | 236,591 ns | 120,886 B | 2,702 |
| **Concrete (JSONProto)** | 279,812 ns | 243,749 B | 2,728 |
| **google.protobuf.Value (proto)** | 680,700 ns | 302,768 B | 6,706 |
| **google.protobuf.Value (JSONProto)** | 988,728 ns | 543,744 B | 10,566 |

</details>
  {{< /tab >}}
{{< /tabs >}}

The most surprising finding here is not that `Value` is slower than static Protobuf. Everyone expects that. The headline-worthy result is this: in these benchmarks, dynamic Protobuf frequently loses to plain JSON as well.

For a medium payload, standard static Protobuf is 19x faster than dynamic binary `Value` serialization, but standard JSON is over 10x faster than `Value`. When evaluating unmarshalling, the gap widens further:

{{< tabs >}}
  {{< tab name="Small Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Concrete (vtproto)",
      "Concrete (proto)",
      "google.protobuf.Any (proto)",
      "Concrete (JSONv2)",
      "Concrete (JSON)",
      "Map (JSONv2)",
      "Protobuf + JSON",
      "Concrete (JSONProto)",
      "Map (JSON)",
      "google.protobuf.Value (proto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Small Payload (ns/op)",
        "data": [32, 141, 315, 430, 936, 964, 1130, 1161, 1345, 1715, 3220],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(186, 85, 211, 1)"
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
        "text": "Unmarshalling Performance (Small Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "proto", "color": "rgba(0, 191, 255, 0.75)" },
          { "text": "json", "color": "rgba(255, 165, 0, 0.75)" },
          { "text": "jsonproto", "color": "rgba(186, 85, 211, 0.75)" }
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
| **Concrete (vtproto)** | **32 ns** | **16 B** | **1** |
| **Concrete (proto)** | 141 ns | 96 B | 2 |
| **google.protobuf.Any (proto)** | 315 ns | 256 B | 5 |
| **Concrete (JSONv2)** | 430 ns | 48 B | 1 |
| **Concrete (JSON)** | 936 ns | 280 B | 6 |
| **Map (JSONv2)** | 964 ns | 408 B | 8 |
| **Protobuf + JSON** | 1,130 ns | 472 B | 9 |
| **Concrete (JSONProto)** | 1,161 ns | 336 B | 14 |
| **Map (JSON)** | 1,345 ns | 648 B | 20 |
| **google.protobuf.Value (proto)** | 1,715 ns | 832 B | 26 |
| **google.protobuf.Value (JSONProto)** | 3,220 ns | 1,256 B | 43 |

</details>
  {{< /tab >}}
  {{< tab name="Medium Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Concrete (vtproto)",
      "Concrete (proto)",
      "google.protobuf.Any (proto)",
      "Concrete (JSONv2)",
      "Map (JSONv2)",
      "Concrete (JSON)",
      "Protobuf + JSON",
      "Map (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (proto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Medium Payload (ns/op)",
        "data": [383, 690, 906, 1410, 2595, 3659, 3970, 4177, 4659, 5686, 10819],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(186, 85, 211, 1)"
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
        "text": "Unmarshalling Performance (Medium Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "proto", "color": "rgba(0, 191, 255, 0.75)" },
          { "text": "json", "color": "rgba(255, 165, 0, 0.75)" },
          { "text": "jsonproto", "color": "rgba(186, 85, 211, 0.75)" }
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
| **Concrete (vtproto)** | **383 ns** | **432 B** | **14** |
| **Concrete (proto)** | 690 ns | 560 B | 15 |
| **google.protobuf.Any (proto)** | 906 ns | 864 B | 18 |
| **Concrete (JSONv2)** | 1,410 ns | 256 B | 4 |
| **Map (JSONv2)** | 2,595 ns | 1,392 B | 30 |
| **Concrete (JSON)** | 3,659 ns | 688 B | 19 |
| **Protobuf + JSON** | 3,970 ns | 1,392 B | 22 |
| **Map (JSON)** | 4,177 ns | 1,856 B | 54 |
| **Concrete (JSONProto)** | 4,659 ns | 1,304 B | 58 |
| **google.protobuf.Value (proto)** | 5,686 ns | 2,888 B | 90 |
| **google.protobuf.Value (JSONProto)** | 10,819 ns | 4,080 B | 145 |

</details>
  {{< /tab >}}
  {{< tab name="Large Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Concrete (vtproto)",
      "Concrete (proto)",
      "google.protobuf.Any (proto)",
      "Concrete (JSONv2)",
      "Map (JSONv2)",
      "Map (JSON)",
      "Concrete (JSON)",
      "Protobuf + JSON",
      "Concrete (JSONProto)",
      "google.protobuf.Value (proto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Large Payload (ns/op)",
        "data": [43925, 67402, 90635, 137095, 219971, 337811, 345927, 358199, 473163, 585077, 1150365],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(186, 85, 211, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(186, 85, 211, 1)"
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
        "text": "Unmarshalling Performance (Large Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "proto", "color": "rgba(0, 191, 255, 0.75)" },
          { "text": "json", "color": "rgba(255, 165, 0, 0.75)" },
          { "text": "jsonproto", "color": "rgba(186, 85, 211, 0.75)" }
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
| **Concrete (vtproto)** | **43,925 ns** | **58,168 B** | **1,508** |
| **Concrete (proto)** | 67,402 ns | 58,232 B | 1,509 |
| **google.protobuf.Any (proto)** | 90,635 ns | 86,400 B | 1,800 |
| **Concrete (JSONv2)** | 137,095 ns | 54,303 B | 309 |
| **Map (JSONv2)** | 219,971 ns | 144,699 B | 3,309 |
| **Map (JSON)** | 337,811 ns | 162,297 B | 4,313 |
| **Concrete (JSON)** | 345,927 ns | 70,584 B | 1,216 |
| **Protobuf + JSON** | 358,199 ns | 136,184 B | 1,219 |
| **Concrete (JSONProto)** | 473,163 ns | 119,256 B | 5,713 |
| **google.protobuf.Value (proto)** | 585,077 ns | 291,106 B | 9,011 |
| **google.protobuf.Value (JSONProto)** | 1,150,365 ns | 395,331 B | 14,414 |

</details>
  {{< /tab >}}
{{< /tabs >}}

Dynamic binary parsing takes **5,772 ns** and requires **90 allocations**, compared to just **674 ns** and **15 allocations** for standard static Protobuf.

### Appendix: The Hidden Cost of Construction
Note that these benchmarks isolate the marshaling and unmarshaling steps using prebuilt structures. They do not include the initial conversion cost of translating native Go types (like `map[string]any`) into `structpb.NewStruct()`. In real systems, building this fragmented graph of heap-allocated objects before serialization even begins can be surprisingly expensive, adding even more overhead to the dynamic Protobuf numbers.

## The Root Cause: Allocations and Pointer Chasing

With the wire and processing numbers in hand, we can look at the mechanical reasons behind the overhead.

1. **Go Runtime Allocations:** Because Go is statically typed, representing a polymorphic JSON-like tree requires nesting interfaces and pointers. Deserializing a dynamic `Value` payload requires Go's runtime to allocate a unique `*structpb.Value` pointer for every single map key, list item, and value in the tree. Every node in the tree is represented as a separate protobuf message (`Value`, `Struct`, or `ListValue`), requiring recursive traversal during serialization and deserialization. On a large payload, this creates over 9,000 individual heap allocations, putting immense pressure on Go's garbage collector and memory allocator. Furthermore, `structpb` relies heavily on interface boxing. Pushing values through these abstraction layers prevents the compiler from applying the aggressive optimizations it normally uses for generated static structs.
2. **Cache Locality and Memory Layout:** At the systems level, statically generated Protobuf messages compile to flat, cache-friendly Go structs representing contiguous (or near-contiguous) memory blocks. In contrast, `structpb.Value` constructs a highly fragmented graph of heap-allocated objects connected by pointers. Traversing this tree-shaped structure causes frequent pointer chasing, which hurts cache locality, increases L1/L2 cache misses, and hinders branch prediction.

## High-Performance Alternatives

Importantly, this does not mean runtime protobuf parsing itself is inherently slow. Rather, the real bottleneck is schema-less, JSON-style polymorphism layered onto Protobuf through `Struct` and `Value`.

If your system requires runtime schema flexibility, avoid `google.protobuf.Struct` for high-throughput paths and leverage these specific optimizations depending on your runtime requirements:

### Polymorphism: Use `google.protobuf.Any`
When data conforms to a known set of pre-compiled schemas, wrap the fields in an `Any` message. It records a clean `type_url` string alongside raw compiled binary bytes.
* **Pros:** Highly compact (212 bytes for a medium payload) and fast. Processing is roughly 11x faster than using generic values.
* **Cons:** Requires compile-time schema awareness for all incoming types. Crucially, the consuming service must have the exact generated Go types compiled and registered in its binary to cleanly unpack the message (e.g., via `anypb.UnmarshalTo()`). If the global protobuf registry lacks the specific type matching the incoming `type_url`, unmarshaling will fail. This strict coupling highlights why `Any` is a schema-dispatch mechanism, not a drop-in replacement for fully unstructured JSON. Additionally, `Any` carries the wire overhead of serializing the `type_url` string (e.g., `type.googleapis.com/package.Message`), which adds a few dozen bytes depending on your package name length. This explains the size increases for `Any` visible in the benchmark charts compared to native static Protobuf.

## Recommendations

To help choose the right design pattern for your dynamic payloads, you can follow this decision flow:

{{< d2 max-height="100%" >}}
direction: down

style: {
  fill: transparent
}

Start: "Data Design Decision" {
  shape: oval
}

Q_Schema: "Fixed schema?" {
  shape: diamond
}

Stable: "Statically typed\nProtobuf fields"

Q_Flat: "Flat key-value\nstrings?" {
  shape: diamond
}

Flat: "map<string, string>"

Q_Opaque: "Opaque to\nmiddle layer?" {
  shape: diamond
}

Any: "google.protobuf.Any"

Q_Perf: "Care about\nperformance?" {
  shape: diamond
}

JSON: "Embedded JSON\nstring/bytes"

Value: "google.protobuf.Value"

Start -> Q_Schema

Q_Schema -> Stable: "Yes"
Q_Schema -> Q_Flat: "No"

Q_Flat -> Flat: "Yes"
Q_Flat -> Q_Opaque: "No"

Q_Opaque -> Any: "Yes"
Q_Opaque -> Q_Perf: "No"

Q_Perf -> JSON: "Yes"
Q_Perf -> Value: "No"
{{< /d2 >}}

### Model your actual data in Protobuf

Statically defining your schemas is always the ideal path. It yields the best performance, full compile-time type safety, and clear API contracts. Commit to first-class, statically typed fields whenever possible. It's worth it.

### Use a native `map<string, string>` for flat attributes

If your metadata is strictly flat key-value strings (like HTTP headers or tags), use a native `map<string, string>`. It converts cleanly to a native Go map without pointer wrapping or parsing overhead.

### Use `google.protobuf.Any` for middle-layer opacity

If you have opaque data that you don't want intermediate routing nodes to parse, wrap the payload in a `google.protobuf.Any` message. This allows middle layers to forward or store packets without deserialization, while downstream consumers can decode the payload cleanly using pre-compiled schemas.

### Pack raw JSON into strings (Opaque JSON Packaging)

If you must support unstructured data on a high-throughput, latency-sensitive hot path, storing the dynamic data as raw JSON directly inside a standard `string` or `bytes` protobuf field is often the best choice.

Let's be honest: this feels gross. Wrapping raw, stringified JSON inside a Protobuf binary envelope violates schema purity, makes API documentation messy, and is generally a dirty hack.

But if you are on a high-throughput, latency-sensitive hot path, the numbers don't care about architectural aesthetics. Bypassing dynamic WKT parsing in favor of opaque JSON packaging saves a lot of CPU cycles and heap allocations. Among the approaches evaluated here, opaque JSON packaging provides the best performance/flexibility tradeoff for fully unstructured payloads. You may not like it, but this might be what peak performance looks like:

```protobuf
message EventEnvelope {
  string event_json = 1;
  int64 timestamp = 2;
}
```

### Use `google.protobuf.Value` for low-throughput dynamic data

If you just want a quick, standardized way to represent arbitrary JSON-like structures in Protobuf and your throughput/latency budgets aren't tight, using the built-in Well-Known Types is completely fine and requires the least custom logic.

## Are You Actually Solving a Dynamic Data Problem?

In practice, many `google.protobuf.Value` fields are not truly dynamic. They are often legacy JSON blobs carried forward during API migrations. Before reaching for `Value`, ask whether the payload has a finite, documented structure. If it does, a normal protobuf message is usually the better long-term design. A dynamic field is often a temporary migration artifact rather than a genuine domain requirement. If the field's shape is stable enough to document, it is usually stable enough to model as a statically typed protobuf message.

## Conclusion

Static protobuf delivers its benefits because the schema is known ahead of time. `google.protobuf.Value` intentionally gives up that information in exchange for flexibility. In Go, that tradeoff can be surprisingly expensive in both wire size and runtime cost. If your payload has a schema, model it. If it doesn't and performance matters, opaque JSON may outperform dynamic protobuf despite being less elegant.
