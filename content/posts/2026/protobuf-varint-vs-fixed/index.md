---
title: "The CPU Cost of Protobuf Varints in Go"
date: "2026-08-04T10:00:00Z"
categories: ["article"]
tags: ["protobuf", "go", "performance", "software-architecture"]
description: "Do fixed-size integers serialize faster than varints? We benchmark the CPU overhead of continuation-bit parsing using Go, vtprotobuf, and hyperpb."
slug: "protobuf-varint-vs-fixed"
cover: "cover.svg"
images: ["/posts/protobuf-varint-vs-fixed/cover.svg"]
type: "posts"
devtoSkip: true
---

When you define an integer field in a Protocol Buffers schema, `int64` is a common default. Varint encoding compresses small numbers into a byte or two, keeping network payloads lean.

However, that compression comes at the cost of CPU cycles. To read or write a varint, the CPU must process the value byte by byte, checking continuation bits and shifting payloads. When a field sits in a high-throughput backend service, CPU efficiency often matters far more than saving a few wire bytes.

Protobuf also provides fixed-size integers (`fixed32`, `fixed64`, `sfixed32`, `sfixed64`), which use a constant-width, little-endian format. While taking more bytes for small values, their CPU path is dramatically simpler.

In Go benchmarks across standard `google.golang.org/protobuf`, PlanetScale `vtprotobuf`, and `hyperpb`, fixed-size integers prove up to 4.5x faster to encode and decode for packed 64-bit arrays—especially when values are large or negative. Here is how wire formats, CPU overhead, and runtime implementations interact in practice.

## How the Wire Formats Actually Differ

Protobuf integer types divide into three encoding groups:

1. Standard varints: `int32`, `int64`, `uint32`, and `uint64`
2. ZigZag varints: `sint32` and `sint64`
3. Fixed-size integers: `fixed32`, `fixed64`, `sfixed32`, and `sfixed64`

### Standard Varints (`int32` / `int64`)

Varints use protobuf's [Base 128 Varint](https://protobuf.dev/programming-guides/encoding/#varints) format. Each byte reserves its MSB as a continuation flag, leaving 7 bits for payload:

* Small numbers (`< 128`) fit in 1 byte.
* Larger numbers require up to 10 bytes for 64-bit integers.

```go
for v >= 1<<7 {
    buf[idx] = byte(v&0x7f | 0x80)
    v >>= 7
    idx++
}
buf[idx] = byte(v)
```

The decoder reverses this bit by bit. While negligible for scalars, this loop adds noticeable overhead over millions of elements in hot paths.

Negative numbers are particularly penalizing: two's-complement representation sets bit 63. Because Base 128 packs only 7 bits per byte, negative integers encoded as standard `int32`/`int64` force the maximum 10-byte encoding every single time—maximizing both wire size and CPU decoding cycles simultaneously.

### ZigZag Varints (`sint32` / `sint64`)

ZigZag encoding solves this penalty by mapping signed integers to unsigned values (`0 -> 0`, `-1 -> 1`, `1 -> 2`, `-2 -> 3`), keeping small absolute values small on the wire.

However, ZigZag only solves payload bloat, not CPU cost. The parser still runs the varint continuation loop for every byte.

### Fixed-Size Integers (`fixed` / `sfixed`)

Fixed-size integers skip small-value compression entirely:

* `fixed32` / `sfixed32`: 4 bytes
* `fixed64` / `sfixed64`: 8 bytes

Represented as raw little-endian values, the parser reads them directly without continuation checks or bit assembly. In Go schemas, `fixed32`/`fixed64` map to `uint32`/`uint64`, while `sfixed32`/`sfixed64` map to `int32`/`int64`.

## The Benchmark Setup

To measure the practical difference in Go, I set up a test module with schemas containing packed repeated integer fields. Each test message holds 1,000 elements.

I benchmarked three value distributions:

1. **Small Positive**: integers in the range `[0, 99]`
2. **Large Positive**: integers in the range `[2^50, 2^50 + 999]`
3. **Negative**: integers in the range `[-100, -1]`

The tests evaluate three Go parsing implementations:

1. The standard `google.golang.org/protobuf` runtime using `proto.Marshal` and `proto.Unmarshal`
2. Generated marshal and unmarshal code from PlanetScale's [`vtprotobuf`](https://github.com/planetscale/vtprotobuf) plugin
3. Descriptor-based dynamic parsing using [`hyperpb`](https://github.com/bufbuild/hyperpb) with a reusable `hyperpb.Shared` memory arena

Note that `hyperpb` is not a direct drop-in replacement for standard struct unmarshaling. It evaluates how a specialized dynamic parser with zero-allocation memory arenas handles the wire formats, highlighting how parser architecture interacts with payload size.

All tests ran on an Apple M1 Pro (`darwin/arm64`) using Go 1.26.3. Averages represent 5 independent runs of 5 seconds each:

```sh
go test -bench=. -benchmem -benchtime=5s -count=5 > results.txt
```

Because these benchmark messages use packed repeated fields, each serialized payload consists of a single field tag, a length prefix, and the concatenated binary values. This structure amortizes the tag overhead across all 1,000 elements, isolating the actual cost of the integer serialization.

### Wire Size Comparison

Before examining CPU timing, look at the serialized payload sizes for 1,000 integers:

| Integer Type                 | Small Positive | Large Positive |   Negative  |
| :--------------------------- | :------------: | :------------: | :---------: |
| **`int64` (Varint)**         |   **1,003 B**  |     8,003 B    |   10,003 B  |
| **`sint64` (ZigZag Varint)** |     1,363 B    |     8,003 B    | **1,363 B** |
| **`sfixed64` (Fixed-Size)**  |     8,003 B    |   **8,003 B**  |   8,003 B   |

{{% tip-box %}}
ZigZag (`sint64`) is slightly larger than plain `int64` for small positive numbers because the bitwise mapping shifts positive values upward. Numbers above 63 cross into 2-byte varint territory sooner. For negative numbers, however, ZigZag reduces payload size by over 86%.
{{% /tip-box %}}

The size trade-off is substantial. For small positive integers, varints are an order of magnitude smaller than fixed-size integers. For large numbers, the byte-saving advantage disappears entirely since a 64-bit varint at `2^50` requires 8 bytes anyway. For negative numbers, plain `int64` expands to 10 bytes per value, making it both larger and more complex to parse than `sfixed64`.

## Benchmark Results

### Marshaling

Serialization benchmarks measure the cost of converting Go structs into protobuf wire data.

{{< tabs >}}
{{< tab name="Small Positive" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "sint64 (ZigZag)",
      "int64 (Varint)",
      "sint64 (ZigZag) + vtproto",
      "int64 (Varint) + vtproto",
      "sfixed64 (Fixed) + vtproto",
      "sfixed64 (Fixed)"
    ],
    "datasets": [
      {
        "label": "ns/op",
        "data": [
          4504,
          3743,
          3525,
          2851,
          2214,
          1768
        ],
        "backgroundColor": [
          "rgba(75, 192, 192, 0.85)",
          "rgba(54, 162, 235, 0.85)",
          "rgba(75, 192, 192, 0.45)",
          "rgba(54, 162, 235, 0.45)",
          "rgba(153, 102, 255, 0.45)",
          "rgba(153, 102, 255, 0.85)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Marshal 64-bit (Small Positive): lower is better",
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
<summary><b>Show data table</b></summary>

| Benchmark (1000 Small Positives) |     ns/op    | Memory (B/op) | Allocations/op |
| :------------------------------- | :----------: | :-----------: | :------------: |
| **`sfixed64 (Fixed)`**           | **1,768 ns** |    8,192 B    |        1       |
| **`sfixed64 (Fixed) + vtproto`** | 2,214 ns     |    8,192 B    |        1       |
| **`int64 (Varint) + vtproto`**   | 2,851 ns     |    1,024 B    |        1       |
| **`sint64 (ZigZag) + vtproto`**  | 3,525 ns     |    1,408 B    |        1       |
| **`int64 (Varint)`**             | 3,743 ns     |    1,024 B    |        1       |
| **`sint64 (ZigZag)`**            | 4,504 ns     |    1,408 B    |        1       |

</details>
{{< /tab >}}
{{< tab name="Large Positive" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "sint64 (ZigZag)",
      "int64 (Varint)",
      "sint64 (ZigZag) + vtproto",
      "int64 (Varint) + vtproto",
      "sfixed64 (Fixed) + vtproto",
      "sfixed64 (Fixed)"
    ],
    "datasets": [
      {
        "label": "ns/op",
        "data": [
          7487,
          7014,
          7036,
          6725,
          2224,
          1727
        ],
        "backgroundColor": [
          "rgba(75, 192, 192, 0.85)",
          "rgba(54, 162, 235, 0.85)",
          "rgba(75, 192, 192, 0.45)",
          "rgba(54, 162, 235, 0.45)",
          "rgba(153, 102, 255, 0.45)",
          "rgba(153, 102, 255, 0.85)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Marshal 64-bit (Large Positive): lower is better",
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
<summary><b>Show data table</b></summary>

| Benchmark (1000 Large Positives) |     ns/op    | Memory (B/op) | Allocations/op |
| :------------------------------- | :----------: | :-----------: | :------------: |
| **`sfixed64 (Fixed)`**           | **1,727 ns** |    8,192 B    |        1       |
| **`sfixed64 (Fixed) + vtproto`** | 2,224 ns     |    8,192 B    |        1       |
| **`int64 (Varint) + vtproto`**   | 6,725 ns     |    8,192 B    |        1       |
| **`int64 (Varint)`**             | 7,014 ns     |    8,192 B    |        1       |
| **`sint64 (ZigZag) + vtproto`**  | 7,036 ns     |    8,192 B    |        1       |
| **`sint64 (ZigZag)`**            | 7,487 ns     |    8,192 B    |        1       |

</details>
{{< /tab >}}
{{< tab name="Negative" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "int64 (Varint)",
      "int64 (Varint) + vtproto",
      "sint64 (ZigZag)",
      "sint64 (ZigZag) + vtproto",
      "sfixed64 (Fixed) + vtproto",
      "sfixed64 (Fixed)"
    ],
    "datasets": [
      {
        "label": "ns/op",
        "data": [
          7609,
          7345,
          4488,
          3478,
          2175,
          1716
        ],
        "backgroundColor": [
          "rgba(54, 162, 235, 0.85)",
          "rgba(54, 162, 235, 0.45)",
          "rgba(75, 192, 192, 0.85)",
          "rgba(75, 192, 192, 0.45)",
          "rgba(153, 102, 255, 0.45)",
          "rgba(153, 102, 255, 0.85)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Marshal 64-bit (Negative): lower is better",
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
<summary><b>Show data table</b></summary>

| Benchmark (1000 Negatives)       |     ns/op    | Memory (B/op) | Allocations/op |
| :------------------------------- | :----------: | :-----------: | :------------: |
| **`sfixed64 (Fixed)`**           | **1,716 ns** |    8,192 B    |        1       |
| **`sfixed64 (Fixed) + vtproto`** | 2,175 ns     |    8,192 B    |        1       |
| **`sint64 (ZigZag) + vtproto`**  | 3,478 ns     |    1,408 B    |        1       |
| **`sint64 (ZigZag)`**            | 4,488 ns     |    1,408 B    |        1       |
| **`int64 (Varint) + vtproto`**   | 7,345 ns     |    10,240 B   |        1       |
| **`int64 (Varint)`**             | 7,609 ns     |    10,240 B   |        1       |

</details>
{{< /tab >}}
{{< /tabs >}}

### Unmarshaling

Deserialization benchmarks measure the CPU cost of parsing wire data back into allocated Go structs.

{{< tabs >}}
{{< tab name="Small Positive" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "sint64 (ZigZag) + vtproto",
      "int64 (Varint) + vtproto",
      "sfixed64 (Fixed) + vtproto",
      "sint64 (ZigZag)",
      "int64 (Varint)",
      "sfixed64 (Fixed)",
      "sint64 (ZigZag) + hyperpb Shared",
      "sfixed64 (Fixed) + hyperpb Shared",
      "int64 (Varint) + hyperpb Shared"
    ],
    "datasets": [
      {
        "label": "ns/op",
        "data": [
          3603,
          3759,
          3156,
          2974,
          2538,
          2366,
          1828,
          1430,
          478
        ],
        "backgroundColor": [
          "rgba(75, 192, 192, 0.45)",
          "rgba(54, 162, 235, 0.45)",
          "rgba(153, 102, 255, 0.45)",
          "rgba(75, 192, 192, 0.85)",
          "rgba(54, 162, 235, 0.85)",
          "rgba(153, 102, 255, 0.85)",
          "rgba(75, 192, 192, 0.30)",
          "rgba(153, 102, 255, 0.30)",
          "rgba(54, 162, 235, 0.30)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Unmarshal 64-bit (Small Positive): lower is better",
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
<summary><b>Show data table</b></summary>

| Benchmark (1000 Small Positives)        |     ns/op    | Memory (B/op) | Allocations/op |
| :-------------------------------------- | :----------: | :-----------: | :------------: |
| **`int64 (Varint) + hyperpb Shared`**   | **478 ns**   |    1,560 B    |        1       |
| **`sfixed64 (Fixed) + hyperpb Shared`** | 1,430 ns     |    10,419 B   |        1       |
| **`sint64 (ZigZag) + hyperpb Shared`**  | 1,828 ns     |    2,066 B    |        1       |
| **`sfixed64 (Fixed)`**                  | 2,366 ns     |    8,256 B    |        2       |
| **`int64 (Varint)`**                    | 2,538 ns     |    8,256 B    |        2       |
| **`sint64 (ZigZag)`**                   | 2,974 ns     |    8,256 B    |        2       |
| **`sfixed64 (Fixed) + vtproto`**        | 3,156 ns     |    8,192 B    |        1       |
| **`sint64 (ZigZag) + vtproto`**         | 3,603 ns     |    8,192 B    |        1       |
| **`int64 (Varint) + vtproto`**          | 3,759 ns     |    8,192 B    |        1       |

</details>
{{< /tab >}}
{{< tab name="Large Positive" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "int64 (Varint) + vtproto",
      "sint64 (ZigZag) + vtproto",
      "sint64 (ZigZag)",
      "int64 (Varint)",
      "sint64 (ZigZag) + hyperpb Shared",
      "int64 (Varint) + hyperpb Shared",
      "sfixed64 (Fixed) + vtproto",
      "sfixed64 (Fixed)",
      "sfixed64 (Fixed) + hyperpb Shared"
    ],
    "datasets": [
      {
        "label": "ns/op",
        "data": [
          11800,
          9201,
          8466,
          8383,
          6526,
          6249,
          3247,
          2505,
          1465
        ],
        "backgroundColor": [
          "rgba(54, 162, 235, 0.45)",
          "rgba(75, 192, 192, 0.45)",
          "rgba(75, 192, 192, 0.85)",
          "rgba(54, 162, 235, 0.85)",
          "rgba(75, 192, 192, 0.30)",
          "rgba(54, 162, 235, 0.30)",
          "rgba(153, 102, 255, 0.45)",
          "rgba(153, 102, 255, 0.85)",
          "rgba(153, 102, 255, 0.30)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Unmarshal 64-bit (Large Positive): lower is better",
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
<summary><b>Show data table</b></summary>

| Benchmark (1000 Large Positives)        |     ns/op    | Memory (B/op) | Allocations/op |
| :-------------------------------------- | :----------: | :-----------: | :------------: |
| **`sfixed64 (Fixed) + hyperpb Shared`** | **1,465 ns** |    10,416 B   |        1       |
| **`sfixed64 (Fixed)`**                  | 2,505 ns     |    8,256 B    |        2       |
| **`sfixed64 (Fixed) + vtproto`**        | 3,247 ns     |    8,192 B    |        1       |
| **`int64 (Varint) + hyperpb Shared`**   | 6,249 ns     |    10,302 B   |        1       |
| **`sint64 (ZigZag) + hyperpb Shared`**  | 6,526 ns     |    10,302 B   |        1       |
| **`int64 (Varint)`**                    | 8,383 ns     |    8,256 B    |        2       |
| **`sint64 (ZigZag)`**                   | 8,466 ns     |    8,256 B    |        2       |
| **`sint64 (ZigZag) + vtproto`**         | 9,201 ns     |    8,192 B    |        1       |
| **`int64 (Varint) + vtproto`**          | 11,800 ns    |    8,192 B    |        1       |

</details>
{{< /tab >}}
{{< tab name="Negative" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "int64 (Varint) + vtproto",
      "int64 (Varint)",
      "int64 (Varint) + hyperpb Shared",
      "sint64 (ZigZag) + vtproto",
      "sfixed64 (Fixed) + vtproto",
      "sint64 (ZigZag)",
      "sfixed64 (Fixed)",
      "sint64 (ZigZag) + hyperpb Shared",
      "sfixed64 (Fixed) + hyperpb Shared"
    ],
    "datasets": [
      {
        "label": "ns/op",
        "data": [
          14518,
          9819,
          7445,
          3650,
          3230,
          3062,
          2465,
          1854,
          1473
        ],
        "backgroundColor": [
          "rgba(54, 162, 235, 0.45)",
          "rgba(54, 162, 235, 0.85)",
          "rgba(54, 162, 235, 0.30)",
          "rgba(75, 192, 192, 0.45)",
          "rgba(153, 102, 255, 0.45)",
          "rgba(75, 192, 192, 0.85)",
          "rgba(153, 102, 255, 0.85)",
          "rgba(75, 192, 192, 0.30)",
          "rgba(153, 102, 255, 0.30)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Unmarshal 64-bit (Negative): lower is better",
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
<summary><b>Show data table</b></summary>

| Benchmark (1000 Negatives)              |     ns/op    | Memory (B/op) | Allocations/op |
| :-------------------------------------- | :----------: | :-----------: | :------------: |
| **`sfixed64 (Fixed) + hyperpb Shared`** | **1,473 ns** |    10,419 B   |        1       |
| **`sint64 (ZigZag) + hyperpb Shared`**  | 1,854 ns     |    2,067 B    |        1       |
| **`sfixed64 (Fixed)`**                  | 2,465 ns     |    8,256 B    |        2       |
| **`sint64 (ZigZag)`**                   | 3,062 ns     |    8,256 B    |        2       |
| **`sfixed64 (Fixed) + vtproto`**        | 3,230 ns     |    8,192 B    |        1       |
| **`sint64 (ZigZag) + vtproto`**         | 3,650 ns     |    8,192 B    |        1       |
| **`int64 (Varint) + hyperpb Shared`**   | 7,445 ns     |    13,654 B   |        1       |
| **`int64 (Varint)`**                    | 9,819 ns     |    8,256 B    |        2       |
| **`int64 (Varint) + vtproto`**          | 14,518 ns    |    8,192 B    |        1       |

</details>
{{< /tab >}}
{{< /tabs >}}

## Analyzing the Numbers

### 1. Value Distribution Impacts Performance

* **Small Positive:** Varints shine on wire efficiency (1,003 B vs 8,003 B). Yet even with 8x larger payloads, `sfixed64` marshals faster in Go by skipping continuation loops. Standard unmarshaling is neck-and-neck (`sfixed64` at 2,366 ns vs `int64` at 2,538 ns). `hyperpb.Shared` leverages the compact varint payload best, reaching 478 ns via specialized arena parsing.
* **Large Positive:** At `2^50`, varints take 8 bytes, which matches `sfixed64` payload size. Without size savings, varint decoding overhead shows its cost: `int64` unmarshaling takes 8,383 ns in standard Go runtime vs 2,505 ns for `sfixed64` (a 3.3x speedup).
* **Negative:** Plain `int64` expands to 10 bytes per value (9,819 ns unmarshal). `sint64` (ZigZag) shrinks wire size back to 1,363 B (3,062 ns unmarshal). `sfixed64` still beats ZigZag at 2,465 ns because flat memory copies beat bit-shifting loops.

### 2. Why Standard Go Runtime Beat Generated Code on Scalar Slices

PlanetScale's `vtprotobuf` generated code was unexpectedly slower than standard `google.golang.org/protobuf` when unmarshaling scalar slices (e.g. `int64 (Varint) + vtproto` at 14,518 ns vs standard runtime at 9,819 ns for negative numbers).

While `vtprotobuf` eliminates reflection overhead on struct fields, packed repeated fixed-width fields are continuous byte blocks.

The standard Go runtime hits an optimized fast path: it reads total length, allocates the destination slice at once, and copies raw bytes into memory using `memmove` primitives.

In contrast, `vtprotobuf` generates an explicit Go loop:

```go
for len(b) > 0 {
    v := binary.LittleEndian.Uint64(b)
    b = b[8:]
    list = append(list, int64(v))
}
```

In CPU-bound array parsing, an explicit Go element-by-element loop cannot compete with bulk memory block copying. Generated code does not automatically beat runtime primitives for bulk primitive data.

## Choosing the Right Integer Type

Protobuf schema decisions map directly to your data distribution and service architecture:

| Type                   | Best for                                          | Avoid when                                   |
| :--------------------- | :------------------------------------------------ | :------------------------------------------- |
| `int64`                | Small non-negative values where wire size matters | Values may be negative or large in hot paths |
| `sint64`               | Small signed values where wire size matters       | Hot repeated fields where CPU dominates      |
| `fixed64` / `sfixed64` | Hot, repeated, CPU-bound fields                   | Small values in bandwidth-sensitive APIs     |

Use these rules to guide your schema definitions:

* **Use `fixed64` / `sfixed64`** for hot, repeated, or CPU-bound fields such as database IDs, timestamps, byte offsets, coordinate arrays, or high-range metrics counters.
* **Use `sint64`** for signed values that frequently hover near zero, especially when network bandwidth or storage footprint is your primary constraint.
* **Use plain `int64`** only when values are strictly non-negative, typically small, and not residing in a serialization hotspot.

Changing an existing field from `int64` to `fixed64` is a breaking wire-format change. Standard varints use wire type `0` (`VARINT`), whereas 64-bit fixed integers use wire type `1` (`I64`). You cannot swap integer types in place without coordinating producer and consumer migrations.

When you design a schema from scratch for high-throughput internal microservices, do not rely on `int64` out of habit. Evaluating your numerical distributions and reaching for `fixed64` or `sfixed64` is a straightforward way to trade away a few cheap network bytes for predictable, measurable CPU savings.
