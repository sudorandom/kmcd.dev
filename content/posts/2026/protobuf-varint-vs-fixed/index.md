---
title: "Varints Save Bytes. Fixed Integers Save CPU"
date: "2026-07-28T10:00:00Z"
categories: ["article"]
tags: ["protobuf", "go", "performance", "software-architecture"]
description: "Do fixed-size integers really serialize faster than varints? We measure the encoding and decoding overhead with Go and vtproto benchmarks."
slug: "protobuf-varint-vs-fixed"
cover: "cover.svg"
images: ["/posts/protobuf-varint-vs-fixed/cover.svg"]
type: "posts"
devtoSkip: true
---

Many Protocol Buffer schemas default to integer types like `int32` and `int64`. They are familiar, compact, and usually good enough.

But their compactness comes from varint encoding, and varint encoding is not free. Every value has to be written or read one byte at a time, with continuation-bit checks and shifting along the way. That is a great trade when wire size matters. It is less obviously great when the field is hot, repeated, and sitting inside an internal service where CPU time matters more than a few extra bytes.

Protobuf also gives us fixed-size integers: `fixed32`, `fixed64`, `sfixed32`, and `sfixed64`. These use a constant-width little-endian representation instead of a variable-length varint. They usually take more bytes for small values, but the CPU path is much simpler.

So I wanted to measure the trade directly. How much CPU time do varints cost in Go? How much faster are fixed-size integers? And where do ZigZag integers fit into the picture?

The answer from these benchmarks is pretty clear: **for this kind of CPU-bound protobuf encoding and decoding, fixed-size integers are faster. Sometimes much faster.**

In these Go benchmarks, fixed-size integers were up to **4.4x faster during marshaling** and **4.5x faster during unmarshaling** when handling larger or negative values. The wire-size trade-off is real, but so is the CPU cost of varints.

---

## Wire Format Mechanics: Varints vs. Fixed

To understand the benchmark results, we need to look at what protobuf actually writes to the wire.

This article focuses on three groups of integer types:

1. Standard varints: `int32`, `int64`, `uint32`, and `uint64`
2. ZigZag varints: `sint32` and `sint64`
3. Fixed-size integers: `fixed32`, `fixed64`, `sfixed32`, and `sfixed64`

They all represent integers, but they have very different CPU and wire-size behavior.

### 1. Standard Varints (`int32` / `int64`)

Varints use protobuf's [Base 128 Varint](https://protobuf.dev/programming-guides/encoding/#varints) format. Each byte uses its most significant bit as a continuation flag. If the bit is set, another byte follows. The remaining 7 bits carry the actual value.

That makes small values very compact:

* `42` fits in a single byte.
* Larger values require more bytes.
* A 64-bit integer can take up to 10 bytes.

This compactness is the entire point of varints. Under the hood, the encoder must loop over the value to write it 7 bits at a time, checking and setting the continuation bit along the way:

```go
for v >= 1<<7 {
    buf[idx] = byte(v&0x7f | 0x80)
    v >>= 7
    idx++
}
buf[idx] = byte(v)
```

The decoder has to do the same work in reverse: reading a byte, checking the continuation bit, shifting the payload bits, and combining them into the final integer.

While this work is tiny for a single value, it becomes significant when handling thousands or millions of values in a hot serialization path.

Negative values are the classic trap. With plain `int32` and `int64`, negative numbers are encoded as large unsigned varints. That means a value like `-42` takes 10 bytes on the wire. If your field can be negative and you use plain `int64`, you are probably paying for the worst version of varint encoding.

### 2. ZigZag Varints (`sint32` / `sint64`)

ZigZag encoding exists to fix the negative-number problem.

Instead of encoding a signed integer directly, protobuf maps signed values onto unsigned values:

* `0` maps to `0`
* `-1` maps to `1`
* `1` maps to `2`
* `-2` maps to `3`

This keeps small negative numbers close to zero after mapping, which means they become small varints.

For example, `-42` as a plain `int64` takes 10 bytes. As a `sint64`, it becomes a small unsigned value first, then gets encoded as a normal varint.

That makes `sint32` and `sint64` much better choices for signed values that are often small in magnitude.

The CPU cost does not disappear, though. ZigZag still uses varint encoding after the signed-to-unsigned mapping. The encoder and decoder still need the continuation-bit loop.

### 3. Fixed-Size Integers (`fixed` / `sfixed`)

Fixed-size integers make the opposite trade. They do not try to save bytes for small values.

Instead:

* `fixed32` and `sfixed32` always take 4 bytes.
* `fixed64` and `sfixed64` always take 8 bytes.

The wire representation is a fixed-width little-endian value. The parser knows exactly how many bytes to read, so it does not need to loop over continuation bits to figure out where the integer ends.

That is why fixed-size integers are interesting for CPU-heavy protobuf workloads. They spend predictable bytes to avoid variable-length integer work.

#### Unsigned (`fixed`) vs. Signed (`sfixed`)

The `fixed32` and `fixed64` types are unsigned fixed-width integers. In Go, they map to `uint32` and `uint64`.

The `sfixed32` and `sfixed64` types are the signed versions. In Go, they map to `int32` and `int64`.

That naming can feel a little backwards if you expect `fixed` to mean signed by default. But in protobuf, the unsigned form gets the shorter name, and the signed form gets the `s` prefix.

The important detail is that both fixed variants use a fixed-width representation on the wire.

---

## The Rules of the Game

To measure the performance difference, I set up a Go module containing protobuf messages with repeated integer slices.

Each benchmark message contains **1,000 elements**.

I tested three value distributions:

1. **Small Positive**: integers in the range `[0, 99]`
2. **Large Positive**: integers in the range `[2^50, 2^50 + 999]`
3. **Negative**: integers in the range `[-100, -1]`

The benchmarks compare two Go protobuf paths:

1. Standard Go protobuf serialization with `proto.Marshal` and `proto.Unmarshal`
2. Generated marshal and unmarshal methods from PlanetScale's [`vtprotobuf`](https://github.com/planetscale/vtprotobuf) plugin

All benchmarks were executed on an Apple M1 Pro (`darwin/arm64`) using Go 1.25.5. Averages represent 5 independent runs of 1 second each using:

```sh
go test -bench=. -benchmem -benchtime=1s -count=5
```

This is not meant to model every possible protobuf workload. It is intentionally narrow. I wanted to isolate the CPU cost of protobuf integer encoding across different value shapes.

### Wire Size Comparison

Before looking at CPU time, here are the serialized payload sizes for 1,000 elements:

| Integer Type                 | Small Positive | Large Positive |   Negative  |
| :--------------------------- | :------------: | :------------: | :---------: |
| **`int64` (Varint)** |   **1,003 B** |     8,003 B    |   10,003 B  |
| **`sint64` (ZigZag Varint)** |     1,363 B    |     8,003 B    | **1,363 B** |
| **`sfixed64` (Fixed-Size)** |     8,003 B    |   **8,003 B** |   8,003 B   |

{{% tip-box %}}
ZigZag (`sint64`) is slightly larger than plain `int64` for small positive numbers because the mapping shifts positive values upward. Values above 63 cross into 2-byte varint territory sooner. But for negative values, ZigZag is dramatically smaller.
{{% /tip-box %}}

The size table already tells part of the story.

For small positive values, varints crush fixed-size integers on bytes. A thousand small `int64` values serialize to 1,003 bytes. The same number of `sfixed64` values takes 8,003 bytes.

For large positive values, the size advantage disappears. The large values in this benchmark require 8 bytes as varints, which puts them at the same payload size as fixed-size integers.

For negative values, plain `int64` is terrible. The payload becomes 10,003 bytes. ZigZag fixes the size problem, and fixed-size integers land in the middle.

Now the real question: what does the CPU do with those bytes?

---

## Benchmark Results

### Marshaling

Serialization benchmarks measure the cost of converting Go structs into protobuf binary data.

{{< tabs >}}
{{< tab name="Small Positive" >}}
{{< chart >}}
{
"type": "bar",
"data": {
"labels": [
"sint64 (Sint)",
"int64 (Varint)",
"sint64 (Sint) + vtproto",
"int64 (Varint) + vtproto",
"sfixed64 (Fixed) + vtproto",
"sfixed64 (Fixed)"
],
"datasets": [{
"label": "ns/op",
"data": [4398, 3642, 3451, 2748, 1881, 1642],
"backgroundColor": [
"rgba(75, 192, 192, 0.85)",
"rgba(54, 162, 235, 0.85)",
"rgba(75, 192, 192, 0.45)",
"rgba(54, 162, 235, 0.45)",
"rgba(153, 102, 255, 0.45)",
"rgba(153, 102, 255, 0.85)"
],
"borderWidth": 0
}]
},
"options": {
"indexAxis": "y",
"plugins": {
"title": {
"display": true,
"text": "Marshal 64-bit (Small Positive): lower is better",
"color": "#fff"
},
"legend": { "display": false }
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

| Benchmark (1000 Small Positives) |     ns/op    | Memory (B/op) | Allocations/op |
| :------------------------------- | :----------: | :-----------: | :------------: |
| **`sfixed64` (Fixed)**           | **1,642 ns** |    8,192 B    |        1       |
| **`sfixed64` (Fixed) + vtproto** |   1,881 ns   |    8,192 B    |        1       |
| **`int64` (Varint) + vtproto**   |   2,748 ns   |    1,024 B    |        1       |
| **`sint64` (Sint) + vtproto**    |   3,451 ns   |    1,408 B    |        1       |
| **`int64` (Varint)**             |   3,642 ns   |    1,024 B    |        1       |
| **`sint64` (Sint)**              |   4,398 ns   |    1,408 B    |        1       |

</details>
  {{< /tab >}}
  {{< tab name="Large Positive" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "sint64 (Sint)",
      "int64 (Varint)",
      "sint64 (Sint) + vtproto",
      "int64 (Varint) + vtproto",
      "sfixed64 (Fixed) + vtproto",
      "sfixed64 (Fixed)"
    ],
    "datasets": [{
      "label": "ns/op",
      "data": [6900, 6739, 6649, 6280, 1783, 1647],
      "backgroundColor": [
        "rgba(75, 192, 192, 0.85)",
        "rgba(54, 162, 235, 0.85)",
        "rgba(75, 192, 192, 0.45)",
        "rgba(54, 162, 235, 0.45)",
        "rgba(153, 102, 255, 0.45)",
        "rgba(153, 102, 255, 0.85)"
      ],
      "borderWidth": 0
    }]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Marshal 64-bit (Large Positive): lower is better",
        "color": "#fff"
      },
      "legend": { "display": false }
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

| Benchmark (1000 Large Positives) |     ns/op    | Memory (B/op) | Allocations/op |
| :------------------------------- | :----------: | :-----------: | :------------: |
| **`sfixed64` (Fixed)**           | **1,647 ns** |    8,192 B    |        1       |
| **`sfixed64` (Fixed) + vtproto** |   1,783 ns   |    8,192 B    |        1       |
| **`int64` (Varint) + vtproto**   |   6,280 ns   |    8,192 B    |        1       |
| **`sint64` (Sint) + vtproto**    |   6,649 ns   |    8,192 B    |        1       |
| **`int64` (Varint)**             |   6,739 ns   |    8,192 B    |        1       |
| **`sint64` (Sint)**              |   6,900 ns   |    8,192 B    |        1       |

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
      "sint64 (Sint)",
      "sint64 (Sint) + vtproto",
      "sfixed64 (Fixed) + vtproto",
      "sfixed64 (Fixed)"
    ],
    "datasets": [{
      "label": "ns/op",
      "data": [7224, 6876, 4360, 3427, 1874, 1630],
      "backgroundColor": [
        "rgba(54, 162, 235, 0.85)",
        "rgba(54, 162, 235, 0.45)",
        "rgba(75, 192, 192, 0.85)",
        "rgba(75, 192, 192, 0.45)",
        "rgba(153, 102, 255, 0.45)",
        "rgba(153, 102, 255, 0.85)"
      ],
      "borderWidth": 0
    }]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Marshal 64-bit (Negative): lower is better",
        "color": "#fff"
      },
      "legend": { "display": false }
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

| Benchmark (1000 Negatives)       |     ns/op    | Memory (B/op) | Allocations/op |
| :------------------------------- | :----------: | :-----------: | :------------: |
| **`sfixed64` (Fixed)**           | **1,630 ns** |    8,192 B    |        1       |
| **`sfixed64` (Fixed) + vtproto** |   1,874 ns   |    8,192 B    |        1       |
| **`sint64` (Sint) + vtproto**    |   3,427 ns   |    1,408 B    |        1       |
| **`sint64` (Sint)**              |   4,360 ns   |    1,408 B    |        1       |
| **`int64` (Varint) + vtproto**   |   6,876 ns   |    10,240 B   |        1       |
| **`int64` (Varint)**             |   7,224 ns   |    10,240 B   |        1       |

</details>
  {{< /tab >}}
{{< /tabs >}}

---

### Unmarshaling

Deserialization benchmarks measure the cost of parsing protobuf binary data back into Go structs.

{{< tabs >}}
{{< tab name="Small Positive" >}}
{{< chart >}}
{
"type": "bar",
"data": {
"labels": [
"sint64 (Sint) + vtproto",
"int64 (Varint) + vtproto",
"sfixed64 (Fixed) + vtproto",
"sint64 (Sint)",
"int64 (Varint)",
"sfixed64 (Fixed)"
],
"datasets": [{
"label": "ns/op",
"data": [3414, 3278, 3049, 3153, 2523, 2251],
"backgroundColor": [
"rgba(75, 192, 192, 0.45)",
"rgba(54, 162, 235, 0.45)",
"rgba(153, 102, 255, 0.45)",
"rgba(75, 192, 192, 0.85)",
"rgba(54, 162, 235, 0.85)",
"rgba(153, 102, 255, 0.85)"
],
"borderWidth": 0
}]
},
"options": {
"indexAxis": "y",
"plugins": {
"title": {
"display": true,
"text": "Unmarshal 64-bit (Small Positive): lower is better",
"color": "#fff"
},
"legend": { "display": false }
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

| Benchmark (1000 Small Positives) |     ns/op    | Memory (B/op) | Allocations/op |
| :------------------------------- | :----------: | :-----------: | :------------: |
| **`sfixed64` (Fixed)**           | **2,251 ns** |    8,256 B    |        2       |
| **`int64` (Varint)**             |   2,523 ns   |    8,256 B    |        2       |
| **`sfixed64` (Fixed) + vtproto** |   3,049 ns   |    8,192 B    |        1       |
| **`sint64` (Sint)**              |   3,153 ns   |    8,256 B    |        2       |
| **`int64` (Varint) + vtproto**   |   3,278 ns   |    8,192 B    |        1       |
| **`sint64` (Sint) + vtproto**    |   3,414 ns   |    8,192 B    |        1       |

</details>
  {{< /tab >}}
  {{< tab name="Large Positive" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "int64 (Varint) + vtproto",
      "sint64 (Sint) + vtproto",
      "sint64 (Sint)",
      "int64 (Varint)",
      "sfixed64 (Fixed) + vtproto",
      "sfixed64 (Fixed)"
    ],
    "datasets": [{
      "label": "ns/op",
      "data": [9547, 8896, 9157, 9181, 2979, 2119],
      "backgroundColor": [
        "rgba(54, 162, 235, 0.45)",
        "rgba(75, 192, 192, 0.45)",
        "rgba(75, 192, 192, 0.85)",
        "rgba(54, 162, 235, 0.85)",
        "rgba(153, 102, 255, 0.45)",
        "rgba(153, 102, 255, 0.85)"
      ],
      "borderWidth": 0
    }]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Unmarshal 64-bit (Large Positive): lower is better",
        "color": "#fff"
      },
      "legend": { "display": false }
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

| Benchmark (1000 Large Positives) |     ns/op    | Memory (B/op) | Allocations/op |
| :------------------------------- | :----------: | :-----------: | :------------: |
| **`sfixed64` (Fixed)**           | **2,119 ns** |    8,256 B    |        2       |
| **`sfixed64` (Fixed) + vtproto** |   2,979 ns   |    8,192 B    |        1       |
| **`sint64` (Sint) + vtproto**    |   8,896 ns   |    8,192 B    |        1       |
| **`sint64` (Sint)**              |   9,157 ns   |    8,256 B    |        2       |
| **`int64` (Varint)**             |   9,181 ns   |    8,256 B    |        2       |
| **`int64` (Varint) + vtproto**   |   9,547 ns   |    8,192 B    |        1       |

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
      "sint64 (Sint) + vtproto",
      "sfixed64 (Fixed) + vtproto",
      "sint64 (Sint)",
      "sfixed64 (Fixed)"
    ],
    "datasets": [{
      "label": "ns/op",
      "data": [10798, 9556, 3539, 3181, 2768, 2126],
      "backgroundColor": [
        "rgba(54, 162, 235, 0.45)",
        "rgba(54, 162, 235, 0.85)",
        "rgba(75, 192, 192, 0.45)",
        "rgba(153, 102, 255, 0.45)",
        "rgba(75, 192, 192, 0.85)",
        "rgba(153, 102, 255, 0.85)"
      ],
      "borderWidth": 0
    }]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Unmarshal 64-bit (Negative): lower is better",
        "color": "#fff"
      },
      "legend": { "display": false }
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

| Benchmark (1000 Negatives)       |     ns/op    | Memory (B/op) | Allocations/op |
| :------------------------------- | :----------: | :-----------: | :------------: |
| **`sfixed64` (Fixed)**           | **2,126 ns** |    8,256 B    |        2       |
| **`sint64` (Sint)**              |   2,768 ns   |    8,256 B    |        2       |
| **`sfixed64` (Fixed) + vtproto** |   3,181 ns   |    8,192 B    |        1       |
| **`sint64` (Sint) + vtproto**    |   3,539 ns   |    8,192 B    |        1       |
| **`int64` (Varint)**             |   9,556 ns   |    8,256 B    |        2       |
| **`int64` (Varint) + vtproto**   |   10,798 ns  |    8,192 B    |        1       |

</details>
  {{< /tab >}}
{{< /tabs >}}

### What the Results Mean

The results split into three useful cases.

For small positive values, varints get their best chance to win. A thousand `int64` values serialize to only 1,003 bytes, while the same number of `sfixed64` values takes 8,003 bytes. Even so, the fixed-size path is faster in both directions: much faster when marshaling, and slightly faster when unmarshaling. In this benchmark, the cost of variable-length encoding outweighed the cost of writing the larger fixed-width payload. The smaller varint payload helps during reads, but it does not fully erase the cost of decoding variable-length integers. That is the most interesting case, because it is where fixed-size integers look worst on paper and still win on CPU.

For large positive values, the byte-size argument disappears. The values in this benchmark require 8 bytes as varints, which puts them at the same payload size as `sfixed64`. Once the wire size is equal, fixed-width encoding wins clearly because it avoids the varint loop entirely.

Negative values are where plain `int64` looks worst. Protobuf encodes negative `int64` values as large varints, so the payload grows to 10,003 bytes for 1,000 elements. ZigZag encoding fixes the wire-size problem by mapping small negative values back near zero before varint encoding. That makes `sint64` a much better choice than plain `int64` for small signed values.

But ZigZag still uses varints. It saves bytes, not CPU. In these benchmarks, `sfixed64` remained the fastest option for negative values because it kept the encode and decode paths fixed-width and predictable.

---

## The vtprotobuf Surprise

One result surprised me. vtprotobuf did not improve the `sfixed64` benchmarks. In several cases, the standard Go protobuf runtime was actually faster.

That looked strange at first. vtprotobuf generates optimized marshal and unmarshal methods to bypass reflection, so I expected it to win across the board. But for this specific benchmark, we accidentally tested a case where the standard runtime is already very hard to beat.

A packed repeated `sfixed64` field is just a length prefix followed by fixed-width values. For this packed scalar case, the standard runtime appears to hit a very efficient path: read the length, allocate the slice, and decode fixed-width values with very little per-field overhead.

vtprotobuf, on the other hand, generates straightforward Go code. When it unmarshals a `sfixed64` slice, it generates a `for` loop that iterates through the buffer 8 bytes at a time. For this narrow scalar case, generated code does not automatically beat the runtime's specialized path.

Code generation is incredible for eliminating protobuf's structural overhead on complex nested messages. But this benchmark is a reminder that the big CPU difference here is not standard protobuf vs. vtprotobuf. It is varint vs. fixed-width encoding.

---

## Choosing the Right Integer Type

So the practical model is simple:

| Type                   | Best for                                          | Avoid when                                   |
| :--------------------- | :------------------------------------------------ | :------------------------------------------- |
| `int64`                | Small non-negative values where wire size matters | Values may be negative or large in hot paths |
| `sint64`               | Small signed values where wire size matters       | CPU is the main bottleneck                   |
| `fixed64` / `sfixed64` | Hot, repeated, CPU-bound fields                   | Small values in bandwidth-sensitive APIs     |

That is the trade: `int64` optimizes for small positive values, `sint64` optimizes for compact signed values, and fixed-size integers optimize for predictable CPU work.

To put this into practice:
- **Use `fixed64` / `sfixed64`** for hot, repeated, or CPU-bound fields (such as database IDs, timestamps, byte offsets, coordinate offsets, or high-range counters).
- **Use `sint64`** for signed values that are often small in magnitude, especially if bandwidth is a constraint.
- **Use plain `int64`** only when values are non-negative, usually small, and you are not in a serialization hotspot.

---

## Final Thoughts

Varints are one of protobuf's best tricks. They make small numbers tiny, which is a great default for general-purpose APIs.

But varints save bytes by spending CPU.

That trade-off is easy to ignore because the cost is hidden inside serialization. You do not see it in the schema. You just write `int64`, generate code, and move on with your life.

In these Go benchmarks, fixed-size integers were consistently faster for both marshaling and unmarshaling. The gap was small for tiny positive values, large for large values, and enormous for negative values encoded as plain `int64`.

The lesson is not that every protobuf integer should become fixed-width. The lesson is that integer encoding deserves a place in the performance conversation.

If a protobuf field is cold, use the boring default and move on.

If a protobuf field is hot, repeated, and CPU-bound, `fixed64` and `sfixed64` are not obscure schema trivia. They are a real optimization lever.

Varints save bytes. Fixed integers save CPU.
