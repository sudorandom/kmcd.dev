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

When you define an integer field in a Protocol Buffers schema, you probably type `int64` without thinking twice. It is a reasonable default. Varint encoding compresses small numbers into just a byte or two, keeping network payloads lean.

However, that compression happens at the expense of CPU cycles. To read or write a varint, the CPU must process the value byte by byte, checking continuation bits and shifting payloads along the way. When a field is hot, repeated, and sitting in an internal backend service, CPU time often matters much more than shaving off a few network bytes.

Protobuf also provides fixed-size integers: `fixed32`, `fixed64`, `sfixed32`, and `sfixed64`. These use a constant-width, little-endian representation on the wire. While they take more bytes for small values, the CPU path is substantially simpler.

To see how much difference this actually makes in Go, I benchmarked the CPU overhead of standard varints against fixed-size integers and ZigZag encoding across three different parsing implementations. In a packed repeated 64-bit workload, fixed-size integers are up to 4.4x faster to marshal and 4.5x faster to unmarshal using the standard Go protobuf runtime, especially when values are large or negative. The wire-size trade-off is real, but so is the CPU overhead of decoding long varints in a hot loop.

## How the Wire Formats Actually Differ

Protobuf integer types divide into three encoding groups:

1. Standard varints: `int32`, `int64`, `uint32`, and `uint64`
2. ZigZag varints: `sint32` and `sint64`
3. Fixed-size integers: `fixed32`, `fixed64`, `sfixed32`, and `sfixed64`

### Standard Varints (`int32` / `int64`)

Varints use protobuf's [Base 128 Varint](https://protobuf.dev/programming-guides/encoding/#varints) format. Each byte reserves its most significant bit as a continuation flag. If the bit is set, another byte follows. The remaining 7 bits carry the actual payload.

This makes small integers very compact:

* `42` fits in a single byte.
* Larger numbers require more bytes.
* A 64-bit integer can take up to 10 bytes.

Under the hood, the encoder must iterate over the value 7 bits at a time, setting continuation flags until the remaining bits are zero (note that `v` must be cast to an unsigned integer like `uint64` so that right-shifting logical shifts zeroes into high bits rather than preserving the sign bit):

```go
for v >= 1<<7 {
    buf[idx] = byte(v&0x7f | 0x80)
    v >>= 7
    idx++
}
buf[idx] = byte(v)
```

The decoder performs this work in reverse by reading a byte, checking the flag, shifting the bits into position, and accumulating the result. For a single scalar, this overhead is negligible. When processing slices of thousands of integers in high-throughput services, those bit-shifting loops add up quickly.

Negative values are particularly punishing here. In two's-complement representation, a negative number has its highest bits set—specifically bit 63. Because Base 128 varints only pack 7 payload bits per byte, that set 63rd bit forces the parser to evaluate all 10 bytes every single time, regardless of how close the actual value is to zero. When encoded as a standard `int32` or `int64` varint, protobuf treats it as a massive unsigned number, forcing the maximum 10-byte encoding every time. If your schema uses plain `int64` for numbers that frequently dip below zero, you are paying the maximum wire size and the maximum CPU decoding cost simultaneously.

### ZigZag Varints (`sint32` / `sint64`)

ZigZag encoding solves the negative-number penalty by mapping signed integers to unsigned integers before applying varint compression:

* `0` maps to `0`
* `-1` maps to `1`
* `1` maps to `2`
* `-2` maps to `3`

By interleaving positive and negative numbers, values close to zero remain small after mapping and compress into just one or two bytes on the wire.

While this fixes the network bloat of negative numbers, it does not eliminate the CPU overhead. ZigZag still relies on varint encoding after the bitwise mapping step. For negative values, shorter varints mean fewer loops and less CPU time than a 10-byte plain `int64`, but the parser still has to execute the continuation-bit loop.

### Fixed-Size Integers (`fixed` / `sfixed`)

Fixed-size integers abandon small-value compression entirely in favor of predictable memory layouts:

* `fixed32` and `sfixed32` always consume 4 bytes.
* `fixed64` and `sfixed64` always consume 8 bytes.

The wire format is simply a raw little-endian integer. Because the parser knows the exact byte length in advance, it reads the data directly without evaluating continuation bits or assembling 7-bit chunks.

In Go protobuf schemas, `fixed32` and `fixed64` represent unsigned integers (`uint32` and `uint64`), while `sfixed32` and `sfixed64` represent signed integers (`int32` and `int64`).

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

The value distribution dictates the parsing cost. Break the results down by shape:

**Small Positive:** Varints get their best opportunity to shine here. Because 1,000 integers pack into just 1,003 bytes, `int64` requires significantly less memory bandwidth than the 8,003-byte payload of `sfixed64`. Despite the 8x larger payload, fixed-size integers still marshal faster in Go because the encoder skips the continuation loop. During standard unmarshaling, `sfixed64` maintains a slight lead over `int64` (2,366 ns vs. 2,538 ns). The fastest result overall is `hyperpb.Shared` reading the tiny 1,003-byte varint payload in 478 ns, proving that specialized dynamic parsers with memory arenas can exploit ultra-compact payloads effectively.

**Large Positive:** When numbers cross the $2^{50}$ threshold, standard varints take 8 bytes each. Once the wire size matches fixed-width integers, the varint compression advantage vanishes. Unmarshaling large `int64` values takes 8,383 ns in the standard runtime, compared to just 2,505 ns for `sfixed64` (a 3.3x speedup). Avoiding the bitwise reconstruction loop entirely yields massive CPU gains when payloads are identically sized.

**Negative:** Plain `int64` falls off a cliff when encoding negative integers, consuming 10 bytes per value and taking 9,819 ns to unmarshal in the standard Go runtime. ZigZag encoding (`sint64`) successfully rescues network bandwidth by shrinking the payload to 1,363 bytes, cutting CPU decoding time to 3,062 ns. Yet `sfixed64` still beats ZigZag cleanly at 2,465 ns because constant-width memory reads outperform bit-shifting loops.

### Why Standard Go Runtime Beat Generated Code on Scalars

One counterintuitive result stands out: PlanetScale's `vtprotobuf` generated code was slower than the standard `google.golang.org/protobuf` runtime across every unmarshaling test for scalar slices.

This anomaly is especially pronounced for negative varints (`int64 (Varint) + vtproto` at 14,518 ns/op vs standard runtime at 9,819 ns/op). Because `vtprotobuf` generates inline decoding loops without table-driven decoding or specialized unrolling for 10-byte sequences, its per-byte bounds checking and continuation bit checks execute sequentially in Go code for all 10 bytes on every negative element.

While `vtprotobuf` typically excels at eliminating reflection overhead in complex nested messages, primitive scalar slices behave differently in Go. A packed repeated `sfixed64` field is fundamentally a length-delimited buffer of constant-width little-endian bytes.

When the standard Go protobuf runtime decodes a packed fixed-width slice, it hits an optimized fast path: it reads the byte length, sizes and allocates the destination slice in one go, and copies the raw bytes directly into the slice's backing array using `memmove` primitives. There is almost zero per-element dispatch overhead.

By contrast, `vtprotobuf` generates straightforward, unrolled Go code. When unmarshaling that same slice, it outputs a standard `for` loop that iterates over the buffer, reading values 8 bytes at a time:

```go
for len(b) > 0 {
    v := binary.LittleEndian.Uint64(b)
    b = b[8:]
    list = append(list, int64(v))
}
```

In a CPU-bound benchmark processing contiguous arrays of primitive integers, an iterated Go loop cannot compete with the runtime's direct memory block copying. Generated code does not automatically beat a highly optimized standard runtime on bulk memory operations.

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
