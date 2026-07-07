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
draft: true
---

Many Protocol Buffer schemas default to integer types like `int32` and `int64`. They are familiar, compact, and usually good enough.

But their compactness comes from varint encoding, and varint encoding is not free. Every value has to be written or read one byte at a time, with continuation-bit checks and shifting along the way. That is a great trade when wire size matters. It is less obviously great when the field is hot, repeated, and sitting inside an internal service where CPU time matters more than a few extra bytes.

Protobuf also gives us fixed-size integers: `fixed32`, `fixed64`, `sfixed32`, and `sfixed64`. These use a constant-width little-endian representation instead of a variable-length varint. They usually take more bytes for small values, but the CPU path is much simpler.

So I wanted to measure the trade directly. How much CPU time do varints cost in Go? How much faster are fixed-size integers? And where do ZigZag integers fit into the picture?

The answer from these benchmarks is pretty clear: **for CPU-bound protobuf encoding and decoding, fixed-size integers are faster. Sometimes much faster.**

In these Go benchmarks, fixed-size integers were up to **4.5x faster during marshaling** and **4.6x faster during unmarshaling** when handling larger or negative values. The wire-size trade-off is real, but so is the CPU cost of varints.

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

This compactness is the entire point of varints. If a value is usually small, protobuf can encode it using fewer bytes than a fixed-width integer.

The cost is that the encoder and decoder cannot treat every value the same way. They have to inspect each byte, check the continuation bit, shift the payload bits into place, and stop when the value is complete.

That work is tiny for one value. It is less tiny for thousands or millions of values in a hot path. Computers are very fast, yes. Annoyingly, they are not magic.

Negative values are the classic trap. With plain `int32` and `int64`, negative numbers are encoded as large unsigned varints. That means a value like `-42` takes 10 bytes on the wire.

If your field can be negative and you use plain `int64`, you are probably paying for the worst version of varint encoding.

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

For this benchmark, I mostly care about the CPU behavior. The important detail is that both fixed variants use a fixed-width representation on the wire.

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

## Benchmark Results: Marshaling

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
"data": [4546, 3735, 3645, 2814, 2166, 1703],
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
| **`sfixed64` (Fixed)** | **1,703 ns** |    8,192 B    |        1       |
| **`sfixed64` (Fixed) + vtproto** |   2,166 ns   |    8,192 B    |        1       |
| **`int64` (Varint) + vtproto** |   2,814 ns   |    1,024 B    |        1       |
| **`sint64` (Sint) + vtproto** |   3,644 ns   |    1,408 B    |        1       |
| **`int64` (Varint)** |   3,735 ns   |    1,024 B    |        1       |
| **`sint64` (Sint)** |   4,546 ns   |    1,408 B    |        1       |

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
      "data": [7600, 7046, 6992, 6813, 2026, 1686],
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
| **`sfixed64` (Fixed)** | **1,686 ns** |    8,192 B    |        1       |
| **`sfixed64` (Fixed) + vtproto** |   2,026 ns   |    8,192 B    |        1       |
| **`int64` (Varint) + vtproto** |   6,813 ns   |    8,192 B    |        1       |
| **`sint64` (Sint) + vtproto** |   6,992 ns   |    8,192 B    |        1       |
| **`int64` (Varint)** |   7,046 ns   |    8,192 B    |        1       |
| **`sint64` (Sint)** |   7,600 ns   |    8,192 B    |        1       |

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
      "data": [7572, 7374, 4532, 3544, 1859, 1659],
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
| **`sfixed64` (Fixed)** | **1,659 ns** |    8,192 B    |        1       |
| **`sfixed64` (Fixed) + vtproto** |   1,859 ns   |    8,192 B    |        1       |
| **`sint64` (Sint) + vtproto** |   3,544 ns   |    1,408 B    |        1       |
| **`sint64` (Sint)** |   4,532 ns   |    1,408 B    |        1       |
| **`int64` (Varint) + vtproto** |   7,374 ns   |    10,240 B   |        1       |
| **`int64` (Varint)** |   7,572 ns   |    10,240 B   |        1       |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Marshaling Takeaways

For marshaling, fixed-size integers are the clear CPU winner.

The most surprising result is the small-positive case. A thousand `sfixed64` values produce an 8,003-byte payload, while a thousand small `int64` values produce a 1,003-byte payload. The fixed-size version writes about 8x more data.

And it is still faster.

For small positive values:

* `int64` takes **3,735 ns/op**
* `sfixed64` takes **1,703 ns/op**
* `sfixed64` is about **2.2x faster**

That is the core tension of this article. Varints are much smaller here, but encoding them still costs CPU.

For large positive values, the contest is not close:

* `int64` takes **7,046 ns/op**
* `sfixed64` takes **1,686 ns/op**
* `sfixed64` is about **4.2x faster**

This is where fixed-size integers shine. Once varints no longer save bytes, they are mostly just extra work.

For negative values, plain `int64` is the worst of both worlds:

* `int64` takes **7,572 ns/op**
* `sint64` takes **4,532 ns/op**
* `sfixed64` takes **1,659 ns/op**

ZigZag helps a lot because it avoids the 10-byte negative-varint encoding. Fixed-size integers are still faster because they avoid the varint loop entirely.

---

## Benchmark Results: Unmarshaling

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
"data": [3361, 3229, 2901, 2763, 2577, 2045],
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
| **`sfixed64` (Fixed)** | **2,045 ns** |    8,256 B    |        2       |
| **`int64` (Varint)** |   2,577 ns   |    8,256 B    |        2       |
| **`sint64` (Sint)** |   2,763 ns   |    8,256 B    |        2       |
| **`sfixed64` (Fixed) + vtproto** |   2,901 ns   |    8,192 B    |        1       |
| **`int64` (Varint) + vtproto** |   3,229 ns   |    8,192 B    |        1       |
| **`sint64` (Sint) + vtproto** |   3,361 ns   |    8,192 B    |        1       |

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
      "data": [8952, 8869, 8334, 8076, 2906, 2080],
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
| **`sfixed64` (Fixed)** | **2,080 ns** |    8,256 B    |        2       |
| **`sfixed64` (Fixed) + vtproto** |   2,906 ns   |    8,192 B    |        1       |
| **`int64` (Varint)** |   8,076 ns   |    8,256 B    |        2       |
| **`sint64` (Sint)** |   8,334 ns   |    8,256 B    |        2       |
| **`sint64` (Sint) + vtproto** |   8,869 ns   |    8,192 B    |        1       |
| **`int64` (Varint) + vtproto** |   8,952 ns   |    8,192 B    |        1       |

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
      "data": [10791, 9513, 3346, 2902, 2754, 2062],
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
| **`sfixed64` (Fixed)** | **2,062 ns** |    8,256 B    |        2       |
| **`sint64` (Sint)** |   2,754 ns   |    8,256 B    |        2       |
| **`sfixed64` (Fixed) + vtproto** |   2,902 ns   |    8,192 B    |        1       |
| **`sint64` (Sint) + vtproto** |   3,346 ns   |    8,192 B    |        1       |
| **`int64` (Varint)** |   9,513 ns   |    8,256 B    |        2       |
| **`int64` (Varint) + vtproto** |   10,791 ns  |    8,192 B    |        1       |

</details>
  {{< /tab >}}
{{< /tabs >}}

### Unmarshaling Takeaways

The unmarshaling results tell the same story, but with a few interesting details.

For small positive values, fixed-size integers still win:

* `int64` takes **2,577 ns/op**
* `sfixed64` takes **2,045 ns/op**
* `sfixed64` is about **1.3x faster**

That gap is smaller than the marshaling gap. This makes sense. The varint payload is much smaller, so the decoder reads far fewer bytes. The smaller payload offsets some of the continuation-bit decoding overhead.

But once values get large, fixed-size integers pull away again:

* `int64` takes **8,076 ns/op**
* `sfixed64` takes **2,080 ns/op**
* `sfixed64` is about **3.9x faster**

The negative-value case is even more brutal:

* `int64` takes **9,513 ns/op**
* `sint64` takes **2,754 ns/op**
* `sfixed64` takes **2,062 ns/op**

Plain `int64` is slow here because it has to parse 10-byte negative varints. ZigZag fixes most of the damage by making the payload tiny. Fixed-size integers are still fastest because their decoding path is simpler and predictable.

---

## The vtprotobuf Surprise

One result surprised me. vtprotobuf did not improve the `sfixed64` benchmarks. In several cases, the standard Go protobuf runtime was actually faster.

That looked strange at first. vtprotobuf generates optimized marshal and unmarshal methods to bypass reflection, so I expected it to win across the board. But for this specific benchmark, we accidentally tested the exact scenario where the standard library has a massive home field advantage.

A packed repeated `sfixed64` field is just a length prefix followed by a continuous block of bytes. The standard `google.golang.org/protobuf` runtime doesn't use slow reflection to parse this. It reads the total byte length from the wire, pre-allocates the exact slice capacity, and routes execution to a highly optimized internal fast path to blast those bytes into memory.

vtprotobuf, on the other hand, generates pure and safe Go code. When it unmarshals a `sfixed64` slice, it generates a `for` loop that iterates through the buffer 8 bytes at a time. A generated Go loop in user space just cannot consistently beat the standard library's internal block-copy magic for raw primitives.

Code generation is incredible for eliminating protobuf's structural overhead on complex nested messages. But as this benchmark shows, it does not automatically beat the standard library at moving raw bytes.

---

## Why Fixed-Size Integers Are Faster

The benchmark results mostly come down to how much work the encoder and decoder have to do per value.

A varint encoder is conceptually doing something like this:

```go
for v >= 1<<7 {
    buf[idx] = byte(v&0x7f | 0x80)
    v >>= 7
    idx++
}
buf[idx] = byte(v)
```

That loop is the cost of compactness.

For a small value, it runs once. For a large 64-bit value, it may run many times. For a negative `int64`, it reaches the worst case.

Unmarshaling has the same problem in reverse. The decoder has to read a byte, check the continuation bit, shift the payload bits, combine them into the final integer, and keep going until the varint ends.

Fixed-size integers avoid that loop. The parser already knows the value is exactly 4 or 8 bytes, so it can read a fixed-width little-endian value directly.

That does not make fixed-size integers free. The encoder still writes bytes. The decoder still reads bytes. Larger payloads still move more memory. But the CPU work per value is more regular, and in these benchmarks that regularity paid off.

### The Size vs. CPU Crossover

The small-positive benchmark is the most interesting case because it is where fixed-size integers look worst on paper.

For 1,000 values from `0` to `99`:

* `int64` payload: **1,003 bytes**
* `sfixed64` payload: **8,003 bytes**

The fixed-size version is roughly 8x larger.

Despite that, `sfixed64` was still faster:

* Marshal: **1,703 ns/op** for `sfixed64` vs. **3,735 ns/op** for `int64`
* Unmarshal: **2,045 ns/op** for `sfixed64` vs. **2,577 ns/op** for `int64`

The unmarshal gap narrows because the varint payload is much smaller. Reading 1,003 bytes instead of 8,003 bytes matters. But the fixed-size path still wins in this benchmark.

For large positive values, the wire-size argument disappears:

* `int64` payload: **8,003 bytes**
* `sfixed64` payload: **8,003 bytes**

Same size. Very different CPU cost.

The fixed-size version is about **4.2x faster to marshal** and **3.9x faster to unmarshal**.

That is the cleanest result in the article. When the wire size is the same, fixed-size integers are much faster.

### The Negative Integer Gotcha

Plain `int64` is a bad fit for negative values.

For 1,000 values from `-100` to `-1`:

* `int64` payload: **10,003 bytes**
* `sint64` payload: **1,363 bytes**
* `sfixed64` payload: **8,003 bytes**

ZigZag is the wire-size champion here. It turns small negative values into small unsigned varints, so the payload stays compact.

But fixed-size integers still win on CPU:

* `int64` unmarshal: **9,513 ns/op**
* `sint64` unmarshal: **2,754 ns/op**
* `sfixed64` unmarshal: **2,062 ns/op**

This gives us two different lessons.

If you care about bytes and your values can be negative, use `sint64` instead of `int64`.

If you care about CPU and can afford 8 bytes per value, `sfixed64` is faster.

---

## The CPU Cost Model

For CPU-bound protobuf workloads, I would think about integer types like this:

| Integer Type           | Small Values               | Large Values        | Negative Values     | CPU Cost |
| :--------------------- | :------------------------- | :------------------ | :------------------ | :------- |
| **`fixed` / `sfixed`** | Largest payload            | Predictable payload | Predictable payload | Lowest   |
| **`sint`** | Slightly larger than `int` | Similar to `int`    | Smallest payload    | Medium   |
| **`int`** | Smallest payload           | Larger varints      | Worst payload       | Highest  |

That is the real model:

* `int64` optimizes for small positive values.
* `sint64` optimizes for small signed values.
* `sfixed64` optimizes for predictable CPU work.

The protobuf type is not just a semantic choice. It is also a performance choice.

### When I Would Use Each Type

For hot internal paths where serialization CPU matters, I would strongly consider fixed-size integers.

Use `fixed64` when:

* The value is unsigned.
* The value is often large.
* The field is repeated or appears in hot messages.
* You care more about CPU time than squeezing small values into fewer bytes.

Examples:

* Large database IDs
* High-range counters
* Timestamps represented as integers
* Byte offsets
* Hash-like numeric values
* Internal IDs that are not usually tiny

Use `sfixed64` when:

* The value can be negative.
* The field is hot.
* The value is not usually tiny enough for ZigZag's size savings to dominate.
* You want predictable CPU behavior.

Examples:

* Signed deltas
* Coordinate offsets
* Balances or adjustments
* Measurements that regularly cross zero

Use `sint64` when:

* The value can be negative.
* The absolute value is often small.
* Wire size still matters.
* You want to avoid the plain `int64` negative-number trap.

Examples:

* Small deltas
* Relative offsets
* Signed counters near zero
* Mobile or bandwidth-sensitive APIs

Use plain `int64` when:

* The value is non-negative in practice.
* The value is usually small.
* Wire size matters more than CPU.
* The field is not a serialization hotspot.

Examples:

* Small counts
* Limits
* Page sizes
* Low-range numeric settings

And if a field regularly stores negative values, I would avoid plain `int64` unless compatibility has already locked it in.

---

## Final Thoughts

Varints are one of protobuf's best tricks. They make small numbers tiny, which is a great default for general-purpose APIs.

But varints save bytes by spending CPU.

That trade-off is easy to ignore because the cost is hidden inside serialization. You do not see it in the schema. You just write `int64`, generate code, and move on with your life like a responsible adult. Terrible habit.

In these Go benchmarks, fixed-size integers were consistently faster for both marshaling and unmarshaling. The gap was small for tiny positive values, large for large values, and enormous for negative values encoded as plain `int64`.

The vtprotobuf result is also a useful reminder: once a case is simple enough, code generation does not guarantee a win. A packed repeated fixed-width scalar is already a friendly path for the standard runtime. The big CPU difference here is not standard runtime vs. vtprotobuf. It is varint vs. fixed-width encoding.

The lesson is not that every protobuf integer should become fixed-width. The lesson is that integer encoding deserves a place in the performance conversation.

If a protobuf message is cold, use the boring default and move on.

If a protobuf message is hot, repeated, and CPU-bound, `fixed64` and `sfixed64` are not obscure schema trivia. They are a real optimization lever.

Varints save bytes. Fixed integers save CPU.