---
title: "google.protobuf.Value considered harmful?"
date: "2026-06-15T10:00:00Z"
categories: ["article"]
tags: ["protobuf", "go", "performance", "json", "software-architecture"]
description: "The hidden performance cost of dynamic Protobuf in Go."
cover: "cover.svg"
images: ["/posts/protobuf-benchmarks/cover.svg"]
featuredalt: ""
featuredpath: "date"
slug: "protobuf-benchmarks"
type: "posts"
devtoSkip: true
draft: false
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
*   [`google.protobuf.Value`](https://protobuf.dev/reference/protobuf/google.protobuf/#value) represents a dynamically typed wrapper that can hold any JSON-compatible value: null, a number (encoded as a double-precision float), a string, a boolean, a nested `Struct` (representing a JSON object), or a list of values (`ListValue`, representing a JSON array).
*   [`google.protobuf.Struct`](https://protobuf.dev/reference/protobuf/google.protobuf/#struct) represents a structured data value consisting of fields that map to dynamically typed values. In practice, it **behaves like a JSON object**, mapping keys (strings) to `Value` messages:
    ```protobuf
    message Struct {
      map<string, Value> fields = 1;
    }
    ```
    In Go, this translates to a key-value map (`map[string]any`) or a raw JavaScript object where the keys are strings and the values are dynamically typed.

Together, they allow Protobuf messages to carry arbitrary, unstructured JSON-like payloads without declaring a strict schema beforehand. This is a common architectural pattern, and it successfully solves a real developer pain point: handling highly dynamic data. Since Protobuf is famous for being fast and compact, one would intuitively think that wrapping dynamic payloads in these well-known types would still be more efficient than standard JSON. I decided to test that assumption, and **I could not have been more wrong**.

---

## 1. The Benchmark Setup

I built a Go benchmark comparing standard JSON against various Protobuf strategies across three payload sizes:
* **Small:** A flat object with 4 fields (string ID, status boolean, age integer, score float).
* **Medium:** A nested user signup event containing an actor object, string tags, and a metadata map.
* **Large:** An array repeating the Medium object 100 times.

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
| **Concrete (JSONProto)** | JSON | Serializes statically generated Protobuf messages into JSON format using Go's official [`protojson`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson) encoder. |
| **google.protobuf.Value (JSONProto)** | JSON | Serializes dynamic [`google.protobuf.Value`](https://protobuf.dev/reference/protobuf/google.protobuf/#value) payloads into JSON format using Go's official [`protojson`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson) encoder. |
| **google.protobuf.Any (JSONProto)** | JSON | Serializes polymorphic [`google.protobuf.Any`](https://protobuf.dev/reference/protobuf/google.protobuf/#any) wrappers into JSON format using Go's official [`protojson`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson) encoder. |

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

**Note on methodology:** These benchmarks isolate the marshaling and unmarshaling steps using prebuilt structures. They do not include network transit times or the initial conversion cost of translating native Go types into `structpb.NewStruct()`, which would add even more overhead to the dynamic Protobuf numbers.

---

## 2. Benchmark Results

The benchmarks were executed under Go 1.26 on an Apple M1 Pro.

### Wire Size

First, I compared the serialized payload size of each configuration. Since Protobuf is widely recognized as a highly compact binary protocol, developers often assume that even unstructured dynamic payloads utilizing `google.protobuf.Value` will naturally be smaller on the wire than standard JSON. Measuring serialized byte sizes is a straightforward test that yields definitive, objective results.

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
      "Concrete (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)",
      "google.protobuf.Any (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Small Payload (Bytes)",
        "data": [25, 74, 74, 55, 55, 55, 111],
        "backgroundColor": [
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

| Format / Config (Small Payload) | Serialized Size | % of JSON (lower is better) |
| :--- | :---: | :---: |
| **Concrete (JSON)** | 55 B | 100.0% (Baseline) |
| **Concrete (proto)** / **Concrete (vtproto)** | 25 B | **45.5%** |
| **google.protobuf.Any (proto)** | 74 B | 134.5% |
| **google.protobuf.Value (proto)** | 74 B | 134.5% |
| **Concrete (JSONProto)** | 55 B | 100.0% |
| **google.protobuf.Value (JSONProto)** | 55 B | 100.0% |
| **google.protobuf.Any (JSONProto)** | 111 B | 201.8% |
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
      "Concrete (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)",
      "google.protobuf.Any (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Medium Payload (Bytes)",
        "data": [162, 328, 212, 291, 293, 291, 349],
        "backgroundColor": [
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

| Format / Config (Medium Payload) | Serialized Size | % of JSON (lower is better) |
| :--- | :---: | :---: |
| **Concrete (JSON)** | 291 B | 100.0% (Baseline) |
| **Concrete (proto)** / **Concrete (vtproto)** | 162 B | **55.7%** |
| **google.protobuf.Any (proto)** | 212 B | **72.9%** |
| **google.protobuf.Value (proto)** | 328 B | 112.7% |
| **Concrete (JSONProto)** | 293 B | 100.7% |
| **google.protobuf.Value (JSONProto)** | 291 B | 100.0% |
| **google.protobuf.Any (JSONProto)** | 349 B | 119.9% |
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
      "Concrete (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)",
      "google.protobuf.Any (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Large Payload (Bytes)",
        "data": [16500, 33104, 21200, 29201, 29412, 29201, 34900],
        "backgroundColor": [
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

| Format / Config (Large Payload) | Serialized Size | % of JSON (lower is better) |
| :--- | :---: | :---: |
| **Concrete (JSON)** | 29,201 B | 100.0% (Baseline) |
| **Concrete (proto)** / **Concrete (vtproto)** | 16,500 B | **56.5%** |
| **google.protobuf.Any (proto)** | 21,200 B | **72.6%** |
| **google.protobuf.Value (proto)** | 33,104 B | 113.4% |
| **Concrete (JSONProto)** | 29,412 B | 100.7% |
| **google.protobuf.Value (JSONProto)** | 29,201 B | 100.0% |
| **google.protobuf.Any (JSONProto)** | 34,900 B | 119.5% |
  {{< /tab >}}
{{< /tabs >}}

While static binary structures are highly compact, dynamic Protobuf schemas drop schema optimization completely. When representing arbitrary object structures, `google.protobuf.Struct` is defined under the hood as `map<string, Value> fields = 1;`.

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

Let's look at the layout of serializing the simple entry `"age": 30` in compact JSON vs. dynamic Protobuf:

### Wire Format Layout Comparison (`"age": 30`)

#### 1. Compact JSON (8 bytes)

For text-based formats it's pretty easy to see where the 'space' is being taken up.

For this subset of JSON, containing a singe key/value pair for age, we have this JSON
```text
"age":30
```
When you count 'structual' characters (commas and quotes), you get 3 'extra' bytes for each key/value pair. It's maybe 4 bytes if you consider commas between elements.

#### 2. Dynamic Protobuf (Protoscope Representation: 18 bytes)

If we describe the dynamic Protobuf binary payload using [Protoscope](https://github.com/protocolbuffers/protoscope) (language for representing raw Protobuf wire formats), the structure and tags become immediately clear:

```protoscope
1: {           # Struct.fields (Map Entry sub-message, tag 1, length-delimited)
  1: "age"     # MapEntry.key (string key, tag 1, length-delimited)
  2: {         # MapEntry.value (Value sub-message, tag 2, length-delimited)
    2: 30.0    # Value.number_value (double float, tag 2, 64-bit)
  }
}
```

- Each set of braces `{}` represents a length-delimited sub-message, which compiles to a field tag followed by a length prefix byte on the wire.
- The prefix tags `1:` and `2:` represent the field numbers.
- `30.0` compiles to field tag 2 (wire type 1, 64-bit) followed by its fixed 8-byte double-precision float value.

To see where the 18 bytes come from, we can break down the serialized dynamic Protobuf payload byte by byte:

| Bytes | Hex Value | Field / Component | Description | Size |
| :---: | :--- | :--- | :--- | :---: |
| **1 – 2** | `0x0A 0x10` | `Struct.fields` Tag & Length | Map entry (tag 1, wire type 2, length 16) | 2 B |
| **3 – 4** | `0x0A 0x03` | `MapEntry.key` Tag & Length | Key field (tag 1, wire type 2, length 3) | 2 B |
| **5 – 7** | `0x61 0x67 0x65` | `MapEntry.key` Value | The string `"age"` | 3 B |
| **8 – 9** | `0x12 0x09` | `MapEntry.value` Tag & Length | Value wrapper (tag 2, wire type 2, length 9) | 2 B |
| **10** | `0x11` | `Value.number_value` Tag | Double float field (tag 2, wire type 1) | 1 B |
| **11 – 18** | `0x00 0x00 ... 0x40` | `Value.number_value` Value | Float64 value `30.0` (little-endian) | 8 B |
| **Total** | | | | **18 B** |

Let's analyze why this byte layout is so inefficient:

1. **Double Nesting and Field Tag Overhead**: In compact JSON, a single isolated field's framing overhead is only 3 bytes (quotes and colon). To be completely fair, a full JSON payload also incurs a small fixed overhead of 2 bytes for the outer curly braces `{}` and 1 byte per additional field for commas `,` (though this is a flat, amortized cost rather than a per-field nested multiplier). In dynamic Protobuf, the nested structure requires **7 bytes of structural framing metadata** (tag and length headers for each layer), plus **3 bytes** for the field name itself.

2. **No Field Name Compression**: One of Protobuf's largest size advantages usually comes from discarding human-readable field names (like `"age"`) and replacing them with compact, 1-byte numeric tags. However, because `google.protobuf.Struct` is unstructured, it must serialize the actual field name string `"age"` on the wire (represented by the `"age"` key value in the Protoscope code). This completely forfeits the field-name compression benefit that makes static Protobuf so compact.

3. **No Varint Compression for Numbers**: Standard Protobuf is highly efficient because it serializes integers using variable-length Varints (e.g., the number `30` takes only 1 byte). However, to remain JSON-compatible, `google.protobuf.Value` stores all numbers as double-precision floating-point numbers (`double`). This completely disables Varint compression:

##### 1. Compact JSON (2 bytes / 16 bits)
```text
┌─────────────────────────┬─────────────────────────┐
│       '3' (0x33)        │       '0' (0x30)        │
│        00110011         │        00110000         │
├─────────────────────────┼─────────────────────────┤
│         1 byte          │         1 byte          │
└─────────────────────────┴─────────────────────────┘
```

##### 2. Static Protobuf with Varint (1 byte / 8 bits)
```text
┌───────────────────────────────────────────────────┐
│                 Varint 30 (0x1E)                  │
│                     00011110                      │
├───────────────────────────────────────────────────┤
│                      1 byte                       │
└───────────────────────────────────────────────────┘
```

##### 3. Dynamic Protobuf (8 bytes / 64 bits)
```text
┌───────────────────────────────────────────────┐
│       8-Byte Float (30.0 double)              │
│  Hex (Little-Endian): 00 00 00 00 00 00 3E 40 │
│      00000000 00000000 00000000 00000000      |
|      00000000 00000000 00111110 01000000      │
├───────────────────────────────────────────────┤
│                  8 bytes                      │
└───────────────────────────────────────────────┘
```

Instead of 2 bytes in JSON or a single byte in native Protobuf, dynamic Protobuf forces even a simple integer to occupy **8 bytes** (double float value) on the wire.

As a result of this multiple-nesting structure and fixed-size floats, dynamic Protobuf payloads end up **significantly larger** on the wire than compact JSON.

### Processing Throughput
Building and parsing schema-less Protobuf trees involves significant pointer-wrapping overhead, resulting in higher CPU usage and frequent heap allocations. Standard concrete Protobuf marshals almost instantly, and PlanetScale's reflection-free generator `Concrete (vtproto)` is the absolute fastest.

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
      "google.protobuf.Value (proto)",
      "Concrete (JSON)",
      "Concrete (JSONv2)",
      "Map (JSONv2)",
      "Map (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Small Payload (ns/op)",
        "data": [29.74, 103.8, 281.6, 2144, 210.8, 347.1, 926.7, 703.6, 750.3, 3064],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
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

| Benchmark (Small Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **Concrete (vtproto)** | **29.7 ns** | **32 B** | **1** |
| **Concrete (proto)** | 103.8 ns | 32 B | 1 |
| **Concrete (JSON)** | 210.8 ns | 64 B | 1 |
| **google.protobuf.Any (proto)** | 281.6 ns | 240 B | 4 |
| **Concrete (JSONv2)** | 347.1 ns | 112 B | 2 |
| **Map (JSON)** | 703.6 ns | 352 B | 10 |
| **Concrete (JSONProto)** | 750.3 ns | 512 B | 12 |
| **Map (JSONv2)** | 926.7 ns | 151 B | 9 |
| **google.protobuf.Value (proto)** | 2,144.0 ns | 879 B | 22 |
| **google.protobuf.Value (JSONProto)** | 3,064.0 ns | 1,364 B | 35 |
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
      "google.protobuf.Value (proto)",
      "Concrete (JSON)",
      "Concrete (JSONv2)",
      "Map (JSONv2)",
      "Map (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Medium Payload (ns/op)",
        "data": [122.0, 359.2, 580.3, 6835, 634.3, 988.5, 1811, 2273, 2732, 10063],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
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

| Benchmark (Medium Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **Concrete (vtproto)** | **122.0 ns** | **176 B** | **1** |
| **Concrete (proto)** | 359.2 ns | 176 B | 1 |
| **google.protobuf.Any (proto)** | 580.3 ns | 528 B | 4 |
| **Concrete (JSON)** | 634.3 ns | 464 B | 2 |
| **Concrete (JSONv2)** | 988.5 ns | 608 B | 3 |
| **Map (JSONv2)** | 1,811.0 ns | 456 B | 12 |
| **Map (JSON)** | 2,273.0 ns | 1,200 B | 28 |
| **Concrete (JSONProto)** | 2,732.0 ns | 1,722 B | 34 |
| **google.protobuf.Value (proto)** | 6,835.0 ns | 2,959 B | 68 |
| **google.protobuf.Value (JSONProto)** | 10,063.0 ns | 4,977 B | 113 |
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
      "google.protobuf.Value (proto)",
      "Concrete (JSON)",
      "Concrete (JSONv2)",
      "Map (JSONv2)",
      "Map (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Large Payload (ns/op)",
        "data": [8683, 30846, 57640, 675890, 50244, 76923, 106473, 223140, 274812, 988378],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
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

| Benchmark (Large Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **Concrete (vtproto)** | **8,683 ns** | **18,432 B** | **1** |
| **Concrete (proto)** | 30,846 ns | 18,432 B | 1 |
| **Concrete (JSON)** | 50,244 ns | 32,823 B | 2 |
| **google.protobuf.Any (proto)** | 57,640 ns | 52,800 B | 400 |
| **Concrete (JSONv2)** | 76,923 ns | 32,847 B | 3 |
| **Map (JSONv2)** | 106,473 ns | 35,275 B | 303 |
| **Map (JSON)** | 223,140 ns | 120,897 B | 2,702 |
| **Concrete (JSONProto)** | 274,812 ns | 243,742 B | 2,728 |
| **google.protobuf.Value (proto)** | 675,890 ns | 302,770 B | 6,706 |
| **google.protobuf.Value (JSONProto)** | 988,378 ns | 543,738 B | 10,565 |
  {{< /tab >}}
{{< /tabs >}}

For a medium payload, standard static Protobuf is 19x faster than dynamic binary `Value` serialization. When evaluating unmarshalling, the gap widens further:

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
      "google.protobuf.Value (proto)",
      "Concrete (JSONv2)",
      "Map (JSONv2)",
      "Concrete (JSON)",
      "Map (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Small Payload (ns/op)",
        "data": [31.57, 141.2, 315.2, 1715, 423.6, 961.1, 941.6, 1336, 1157, 3192],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
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

| Benchmark (Small Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **Concrete (vtproto)** | **31.6 ns** | **16 B** | **1** |
| **Concrete (proto)** | 141.2 ns | 96 B | 2 |
| **google.protobuf.Any (proto)** | 315.2 ns | 256 B | 5 |
| **Concrete (JSONv2)** | 423.6 ns | 48 B | 1 |
| **Concrete (JSON)** | 941.6 ns | 280 B | 6 |
| **Map (JSONv2)** | 961.1 ns | 408 B | 8 |
| **Concrete (JSONProto)** | 1,157.0 ns | 336 B | 14 |
| **Map (JSON)** | 1,336.0 ns | 648 B | 20 |
| **google.protobuf.Value (proto)** | 1,715.0 ns | 832 B | 26 |
| **google.protobuf.Value (JSONProto)** | 3,192.0 ns | 1,256 B | 43 |
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
      "google.protobuf.Value (proto)",
      "Concrete (JSONv2)",
      "Map (JSONv2)",
      "Concrete (JSON)",
      "Map (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Medium Payload (ns/op)",
        "data": [369.4, 673.5, 877.4, 5772, 1396, 2581, 3645, 4148, 4645, 10903],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
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

| Benchmark (Medium Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **Concrete (vtproto)** | **369.4 ns** | **432 B** | **14** |
| **Concrete (proto)** | 673.5 ns | 560 B | 15 |
| **google.protobuf.Any (proto)** | 877.4 ns | 864 B | 18 |
| **Concrete (JSONv2)** | 1,396.0 ns | 256 B | 4 |
| **Map (JSONv2)** | 2,581.0 ns | 1,392 B | 30 |
| **Concrete (JSON)** | 3,645.0 ns | 688 B | 19 |
| **Map (JSON)** | 4,148.0 ns | 1,856 B | 54 |
| **Concrete (JSONProto)** | 4,645.0 ns | 1,304 B | 58 |
| **google.protobuf.Value (proto)** | 5,772.0 ns | 2,888 B | 90 |
| **google.protobuf.Value (JSONProto)** | 10,903.0 ns | 4,080 B | 145 |
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
      "google.protobuf.Value (proto)",
      "Concrete (JSONv2)",
      "Map (JSONv2)",
      "Concrete (JSON)",
      "Map (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Large Payload (ns/op)",
        "data": [42896, 66722, 88219, 588602, 136392, 221455, 345604, 337503, 471031, 1104782],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(186, 85, 211, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(255, 165, 0, 1)",
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

| Benchmark (Large Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **Concrete (vtproto)** | **42,896 ns** | **58,168 B** | **1,508** |
| **Concrete (proto)** | 66,722 ns | 58,232 B | 1,509 |
| **google.protobuf.Any (proto)** | 88,219 ns | 86,400 B | 1,800 |
| **Concrete (JSONv2)** | 136,392 ns | 54,303 B | 309 |
| **Map (JSONv2)** | 221,455 ns | 144,647 B | 3,309 |
| **Map (JSON)** | 337,503 ns | 162,296 B | 4,313 |
| **Concrete (JSON)** | 345,604 ns | 70,584 B | 1,216 |
| **Concrete (JSONProto)** | 471,031 ns | 119,256 B | 5,713 |
| **google.protobuf.Value (proto)** | 588,602 ns | 291,171 B | 9,011 |
| **google.protobuf.Value (JSONProto)** | 1,104,782 ns | 395,330 B | 14,414 |
  {{< /tab >}}
{{< /tabs >}}

Dynamic binary parsing takes **5,772.0 ns** and requires **90 allocations**, compared to just **673.5 ns** and **15 allocations** for standard static Protobuf.

---

## 3. The Root Cause: Wire Overhead and Heap Allocations

The performance drop comes down to two specific architectural factors:

1. **Wire Format Overhead:** Statically compiled Protobuf omits field names entirely, sending only numeric field tags. A dynamic `Value` field has no static schema. To represent a simple key-value pair like `{"age": 30}`, Protobuf must serialize a `MapEntry` message containing the string key `"age"` (which takes 5 bytes on the wire including its field tag and length prefix), the field tag of the selected type inside the `Value` message (1 byte), and an 8-byte double precision float. With all the nesting tags and length prefixes included, the dynamic Protobuf footprint for this single pair reaches **18 bytes**, compared to just **8 bytes** in compact JSON (`"age":30`).
2. **Heap Allocations in Go:** In Go, representing dynamic, polymorphic variants requires nested pointers and interfaces. Every map item inside a `structpb.Struct` maps to a distinct `*structpb.Value` pointer containing an interface value. Parsing a large payload into this structural tree demands **over 9,000 individual heap allocations**, introducing substantial garbage collection pressure.

---

## 4. High-Performance Alternatives

Importantly, this problem is not inherent to runtime protobuf parsing itself. The real bottleneck is schema-less JSON-style polymorphism layered onto protobuf through `Struct` and `Value`.

If your system requires runtime schema flexibility, avoid `google.protobuf.Struct` for high-throughput paths and leverage these specific optimizations depending on your runtime requirements:

### Polymorphism: Use `google.protobuf.Any`
When data conforms to a known set of pre-compiled schemas, wrap the fields in an `Any` message. It records a clean `type_url` string alongside raw compiled binary bytes.
* **Pros:** Highly compact (212 bytes for a medium payload) and fast. Processing is roughly 11x faster than using generic values.
* **Cons:** Requires compile-time schema awareness for all incoming types.

### Runtime Schema Discovery: Use Buf's `hyperpb`
For pipelines that handle dynamic descriptors entirely at runtime (like schema registries or event gateways), Go's native `dynamicpb` is notoriously slow. Buf's `hyperpb` fixes this by compiling a message descriptor into dedicated Table-Driven Parser bytecode at application startup.

To evaluate dynamic runtime parsing options, I compared the following variants:

| Variant | Format | Description |
| :--- | :---: | :--- |
| **dynamicpb** | Protobuf | Evaluates dynamic descriptor compilation and reflection-based Protobuf handling at runtime using Go's standard [`dynamicpb`](https://pkg.go.dev/google.golang.org/protobuf/types/dynamicpb) package. |
| **hyperpb** | Protobuf | Evaluates dynamic descriptor parsing and serialization at runtime using Buf's table-driven [`hyperpb`](https://github.com/bufbuild/hyperpb) library. |
| **hyperpb + Shared** | Protobuf | Evaluates dynamic parsing using Buf's [`hyperpb`](https://github.com/bufbuild/hyperpb) paired with a thread-local, pre-allocated memory arena to eliminate runtime heap allocations. |

By combining this bytecode engine with a thread-local `hyperpb.Shared` arena pool, you can eliminate request-time heap churn:

```go
shared := new(hyperpb.Shared) // Instantiated once per goroutine

for _, payload := range incoming {
    msg := shared.NewMessage(mType) // Reuses the underlying memory arena
    _ = proto.Unmarshal(payload, msg)
    
    route(msg) // Read-only access pipeline
    shared.Free() // Recycles the arena back to the pool
}
```

On a large payload, `hyperpb + Shared` processes requests in **22,074 ns** with exactly **1 heap allocation**, outperforming even build-time generated static Protobuf code (**66,722 ns**, **1,509 allocations**).

{{< tabs >}}
  {{< tab name="Small Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Map (JSON)",
      "Map (JSONv2)",
      "dynamicpb",
      "hyperpb",
      "hyperpb + Shared",
      "Concrete (proto)",
      "Concrete (vtproto)"
    ],
    "datasets": [{
      "label": "ns/op",
      "data": [1336, 961.1, 762.3, 381.3, 150.7, 141.2, 31.57],
      "backgroundColor": [
        "rgba(255, 165, 0, 0.75)",
        "rgba(255, 165, 0, 0.75)",
        "rgba(0, 191, 255, 0.75)",
        "rgba(0, 191, 255, 0.75)",
        "rgba(0, 191, 255, 0.75)",
        "rgba(148, 163, 184, 0.75)",
        "rgba(148, 163, 184, 0.75)"
      ],
      "borderColor": [
        "rgba(255, 165, 0, 1)",
        "rgba(255, 165, 0, 1)",
        "rgba(0, 191, 255, 1)",
        "rgba(0, 191, 255, 1)",
        "rgba(0, 191, 255, 1)",
        "rgba(148, 163, 184, 1)",
        "rgba(148, 163, 184, 1)"
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
          { "text": "proto", "color": "rgba(0, 191, 255, 0.75)" },
          { "text": "json", "color": "rgba(255, 165, 0, 0.75)" },
          { "text": "baseline", "color": "rgba(148, 163, 184, 0.75)" }
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

| Benchmark (Small Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **Concrete (vtproto)** | **31.6 ns** | **16 B** | **1** |
| **Concrete (proto)** | 141.2 ns | 96 B | 2 |
| **hyperpb + Shared** | 150.7 ns | 64 B | 1 |
| **hyperpb** | 381.3 ns | 799 B | 4 |
| **dynamicpb** | 762.3 ns | 616 B | 11 |
| **Map (JSONv2)** | 961.1 ns | 408 B | 8 |
| **Map (JSON)** | 1,336.0 ns | 648 B | 20 |
  {{< /tab >}}
  {{< tab name="Medium Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Map (JSON)",
      "dynamicpb",
      "Map (JSONv2)",
      "hyperpb",
      "Concrete (proto)",
      "Concrete (vtproto)",
      "hyperpb + Shared"
    ],
    "datasets": [{
      "label": "ns/op",
      "data": [4148, 2930, 2581, 697.1, 673.5, 369.4, 350.2],
      "backgroundColor": [
        "rgba(255, 165, 0, 0.75)",
        "rgba(0, 191, 255, 0.75)",
        "rgba(255, 165, 0, 0.75)",
        "rgba(0, 191, 255, 0.75)",
        "rgba(148, 163, 184, 0.75)",
        "rgba(148, 163, 184, 0.75)",
        "rgba(0, 191, 255, 0.75)"
      ],
      "borderColor": [
        "rgba(255, 165, 0, 1)",
        "rgba(0, 191, 255, 1)",
        "rgba(255, 165, 0, 1)",
        "rgba(0, 191, 255, 1)",
        "rgba(148, 163, 184, 1)",
        "rgba(148, 163, 184, 1)",
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
        "text": "Dynamic Parsing Performance (Medium Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "proto", "color": "rgba(0, 191, 255, 0.75)" },
          { "text": "json", "color": "rgba(255, 165, 0, 0.75)" },
          { "text": "baseline", "color": "rgba(148, 163, 184, 0.75)" }
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

| Benchmark (Medium Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **hyperpb + Shared** | **350.2 ns** | **357 B** | **1** |
| **Concrete (vtproto)** | 369.4 ns | 432 B | 14 |
| **Concrete (proto)** | 673.5 ns | 560 B | 15 |
| **hyperpb** | 697.1 ns | 1,446 B | 5 |
| **Map (JSONv2)** | 2,581.0 ns | 1,392 B | 30 |
| **dynamicpb** | 2,930.0 ns | 2,072 B | 43 |
| **Map (JSON)** | 4,148.0 ns | 1,856 B | 54 |
  {{< /tab >}}
  {{< tab name="Large Payload" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Map (JSON)",
      "dynamicpb",
      "Map (JSONv2)",
      "Concrete (proto)",
      "Concrete (vtproto)",
      "hyperpb",
      "hyperpb + Shared"
    ],
    "datasets": [{
      "label": "ns/op",
      "data": [337503, 298918, 221455, 66722, 42896, 29197, 22074],
      "backgroundColor": [
        "rgba(255, 165, 0, 0.75)",
        "rgba(0, 191, 255, 0.75)",
        "rgba(255, 165, 0, 0.75)",
        "rgba(148, 163, 184, 0.75)",
        "rgba(148, 163, 184, 0.75)",
        "rgba(0, 191, 255, 0.75)",
        "rgba(0, 191, 255, 0.75)"
      ],
      "borderColor": [
        "rgba(255, 165, 0, 1)",
        "rgba(0, 191, 255, 1)",
        "rgba(255, 165, 0, 1)",
        "rgba(148, 163, 184, 1)",
        "rgba(148, 163, 184, 1)",
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
        "text": "Dynamic Parsing Performance (Large Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" },
        "customLegend": [
          { "text": "proto", "color": "rgba(0, 191, 255, 0.75)" },
          { "text": "json", "color": "rgba(255, 165, 0, 0.75)" },
          { "text": "baseline", "color": "rgba(148, 163, 184, 0.75)" }
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

| Benchmark (Large Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **hyperpb + Shared** | **22,074 ns** | **21,838 B** | **1** |
| **hyperpb** | 29,197 ns | 59,999 B | 12 |
| **Concrete (vtproto)** | 42,896 ns | 58,168 B | 1,508 |
| **Concrete (proto)** | 66,722 ns | 58,232 B | 1,509 |
| **Map (JSONv2)** | 221,455 ns | 144,647 B | 3,309 |
| **dynamicpb** | 298,918 ns | 205,753 B | 4,117 |
| **Map (JSON)** | 337,503 ns | 162,296 B | 4,313 |
  {{< /tab >}}
{{< /tabs >}}

---

## 5. When google.protobuf.Struct Is Still Reasonable

`Struct` remains useful and entirely appropriate for:

* low-throughput administrative APIs
* debugging endpoints
* rapidly evolving schemas
* plugin metadata
* cross-language extensibility layers

The performance problems only emerge when these dynamic trees sit directly on hot-path production traffic.

---

## 6. Recommendations

1. **Stable Schemas:** Commit to first-class, statically typed fields whenever possible. It's worth it.
2. **Flat Attributes:** If your metadata is strictly flat key-value strings (like HTTP headers or tags), use a native `map<string, string>`. It converts cleanly to a native Go map without pointer wrapping.
3. **Opaque JSON Packaging:** If the payload is complex, nested, and truly arbitrary, bypass the Protobuf wrapper completely. Store the raw data as an opaque `string` or `bytes` field directly in the message template:

```protobuf
message UserEvent {
  string event_id = 1;
  int64 timestamp = 2;
  string raw_metadata_json = 3; // Avoids structural parsing overhead during transit
}
```

This lets your edge nodes route the packet instantly without parsing overhead. Downstream consumer services can then extract and decode the payload cleanly into native Go structures using optimized JSON parsers only when necessary.


