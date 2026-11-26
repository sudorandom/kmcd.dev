---
title: "Variable-Length Integers in Go"
date: "2026-12-01T18:00:00Z"
categories: ["article"]
tags: ["golang", "performance", "protobuf", "http3", "omnibuf", "serialization"]
keywords: ["varint", "variable-length integers", "leb128", "quic varint", "prefix varint", "omnibuf", "branch prediction", "golang benchmarks"]
description: "Comparing Protobuf LEB128, QUIC, OmniBuf, Prefix-9, MessagePack, and fixed-width 64-bit integers in Go across wire size, CPU speed, compression, and branch prediction."
slug: "variable-length-integers"
cover: "cover.svg"
images: ["/posts/variable-length-integers/cover.svg"]
type: "posts"
devtoSkip: true
---

I've run into variable-length integers a few times recently, first when [benchmarking Protobuf varints against fixed-width integers](/posts/protobuf-varint-vs-fixed/) and again while implementing QUIC's framing for [HTTP/3 From Scratch](/posts/http3-from-scratch-part-1/). Comparing those two encodings is what pushed me to design a different varint strategy for [OmniBuf](/posts/omnibuf/).

I ended up benchmarking six different integer encodings in Go. I was originally interested in the tradeoff between CPU time and wire size, but some of the results were pretty unexpected. For example, sorting the input before encoding it made some of the decoders run three times as fast.

## Why Variable-Length Integers?

So... why do we want to use variable-length integers, anyway?

Let's use the number `300`. In binary, `300` is `1 0010 1100` (9 bits). Send it over the wire as a big-endian, fixed-width 64-bit unsigned integer (`uint64`), and those bits look like this:

```text
00000000 00000000
00000000 00000000
00000000 00000000
00000001 00101100
```

That's 55 bits of nothing. If we're encoding something like a Protobuf field number or an array length, we probably don't need anywhere near 64 bits.

We could remove those leading zeros, but then how would we encode two integers next to each other? If I write out the bits for `300` immediately followed by the bits for `5`, there's no way to recover those two numbers unless we know their lengths. There are quite a few ways to solve this.

## Six Integer Encodings

### Protobuf / LEB128

Protobuf uses [LEB128](https://protobuf.dev/programming-guides/encoding/#varints) (Little-Endian Base 128). Read a byte, take the bottom 7 bits, and check the top bit (the continuation bit). If it's set, read another byte and place its bottom 7 bits above the previous ones. Repeat until you get a byte where the top bit is zero.

{{< diagram >}}
{{< image src="layout-leb128.svg" class="center" inline="true" >}}
{{< /diagram >}}

There are already two implementations of this in Go that we can benchmark: [`encoding/binary.Uvarint`](https://pkg.go.dev/encoding/binary#Uvarint) in the standard library and [`protowire.ConsumeVarint`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protowire#ConsumeVarint) in the Protobuf library. The Protobuf implementation unrolls the decoding loop, while the standard library uses a regular loop. I also wrote my own version (`LEB128 (fast1B)` in the benchmarks below) that handles single-byte integers directly inside the outer loop:

```go
func DecodeLEB128(src []byte, dst []uint64) int {
	i, n := 0, 0
	for i < len(src) {
		b := src[i]
		if b < 0x80 {
			dst[n] = uint64(b)
			i++
			n++
			continue
		}
		v := uint64(b & 0x7f)
		i++
		var shift uint = 7
		for {
			b = src[i]
			i++
			v |= uint64(b&0x7f) << shift
			if b < 0x80 {
				break
			}
			shift += 7
		}
		dst[n] = v
		n++
	}
	return n
}
```

We get 7 bits of our integer per byte, so an integer between 0 and 127 takes one byte, but we need as many as ten bytes to store a `uint64`.

There's also an issue with negative numbers in Protobuf. `-1` takes *ten bytes* with `int32` or `int64`. Yes, a 32-bit type actually uses 10 bytes to represent `-1` because Protobuf sign-extends negative `int32` values to 64-bit two's complement before varint encoding! Protobuf provides the `sint32` and `sint64` types, which use ZigZag encoding to map small negative numbers to small unsigned numbers before encoding them. Formats like DWARF and WebAssembly also use Signed LEB128 (SLEB128), which can represent small negative numbers with no ZigZag encoding.

### QUIC / HTTP/3

QUIC does something quite different from Protobuf. In [RFC 9000 Section 16](https://www.rfc-editor.org/rfc/rfc9000.html#name-variable-length-integer-enc), the top two bits of the first byte tell us how many bytes to read. There are only four possible sizes: 1, 2, 4, and 8 bytes. The rest of the bits contain the integer in big-endian order.

{{< diagram >}}
{{< image src="layout-quic.svg" class="center" inline="true" >}}
{{< /diagram >}}

```go
func DecodeQUIC(src []byte, dst []uint64) int {
	i, n := 0, 0
	for i < len(src) {
		b0 := src[i]
		switch b0 >> 6 {
		case 0:
			dst[n] = uint64(b0)
			i++
		case 1:
			dst[n] = uint64(binary.BigEndian.Uint16(src[i:]) & 0x3fff)
			i += 2
		case 2:
			dst[n] = uint64(binary.BigEndian.Uint32(src[i:]) & 0x3fffffff)
			i += 4
		case 3:
			dst[n] = binary.BigEndian.Uint64(src[i:]) & 0x3fffffffffffffff
			i += 8
		}
		n++
	}
	return n
}
```

What I like about this encoding is that we know exactly how many bytes to read after looking at the first byte. We can also use the functions in `encoding/binary` to read the remaining bits in one go.

The downside is that we lose two bits even when encoding a single-byte integer. We also can't store anything larger than `2^62 - 1`. That's fine for HTTP/3 frame lengths, but I wanted OmniBuf to support the entire `uint64` range.

### OmniBuf

For [OmniBuf](https://omnibuf.com/how-it-works#prefix-varints), I wanted something like QUIC's varints that still kept the full 64-bit range and could fit `127` in a single byte. I also wanted little-endian byte order to avoid byte swapping on the CPUs I typically work with.

I went with a variation of prefix varints, similar to Rust's [`vint64`](https://docs.rs/vint64/latest/vint64/) and [LLVM's MLIR bytecode varints](https://mlir.llvm.org/docs/BytecodeFormat/). Instead of QUIC's top two bits, we count the trailing zeros at the bottom of the first byte. Since the integer is little-endian, keeping the tag in the low bits leaves all the payload bits contiguous so we can strip it with a single right shift:

- `...1` (0 trailing zeros): 1 byte, 7 payload bits (`0..127`)
- `...10` (1 trailing zero): 2 bytes, 14 payload bits (`0..16,383`)
- `...100` (2 trailing zeros): 4 bytes, 29 payload bits (`0..536,870,911`)
- `0x00`: 9 bytes, followed by a raw 8-byte little-endian `uint64`

{{< diagram >}}
{{< image src="layout-omnibuf.svg" class="center" inline="true" >}}
{{< /diagram >}}

```go
func DecodeOmniBuf(src []byte, dst []uint64) int {
	i, n := 0, 0
	for i < len(src) {
		b0 := src[i]
		switch {
		case b0&1 != 0:
			dst[n] = uint64(b0 >> 1)
			i++
		case b0&2 != 0:
			dst[n] = uint64(binary.LittleEndian.Uint16(src[i:]) >> 2)
			i += 2
		case b0&4 != 0:
			dst[n] = uint64(binary.LittleEndian.Uint32(src[i:]) >> 3)
			i += 4
		case b0 == 0:
			dst[n] = binary.LittleEndian.Uint64(src[i+1:])
			i += 9
		default:
			return n
		}
		n++
	}
	return n
}
```

By limiting the encoded widths to 1, 2, 4, and 9 bytes, we can extract each payload with a single `uint8`, `uint16`, `uint32`, or `uint64` read. Anything at `2^29` or above takes nine bytes, even when LEB128 would only use five.

### Prefix-9

What if we didn't skip those other widths? Each additional trailing zero in the first byte indicates another byte of encoded data: zero trailing zeros for 1 byte, one for 2 bytes, two for 3 bytes, and so on up to 9 bytes. (That also means OmniBuf's 4-byte encoding gets 29 payload bits, one more than Prefix-9's 28.) I'll call this encoding `Prefix-9` throughout the benchmarks.

It takes the same number of bytes as LEB128 for integers up to 56 bits. Anything larger gets a zero byte followed by a `uint64`, so we never need that tenth byte that LEB128 sometimes uses.

For values up to `16,383` (including `300`), Prefix-9 produces the exact same bytes as OmniBuf. Switching to `70,000` shows the 3-byte case that OmniBuf skips:

{{< diagram >}}
{{< image src="layout-prefix9.svg" class="center" inline="true" >}}
{{< /diagram >}}

There's a neat trick we can do with this format. Since the number of trailing zeros tells us both the length of the integer and how many bits we need to remove, we can decode the first eight widths with no switch statement. Read eight bytes and shift the bits we don't want off either end:

```go
tz := uint(bits.TrailingZeros8(b0))
width := tz + 1
word := binary.LittleEndian.Uint64(src[i:])
shiftLeft := (8 - width) * 8
dst[n] = (word << shiftLeft) >> (shiftLeft + width)
i += int(width)
```

In the full decoder, we check for the `0x00` 9-byte escape first and fall back to a byte-by-byte loop when fewer than eight bytes remain.

I also wanted to see how much we'd gain by special-casing small integers again. `Prefix-9 (fast1B)` is the same decoder with an extra `if b0&1 != 0` check before it does any of this work.

### MessagePack

[MessagePack](https://github.com/msgpack/msgpack/blob/master/spec.md#int-format-family) (MsgPack) is a self-describing format where the first byte encodes the value's type (and, for small integers, the value itself). Values from 0 to 127 just use `0x00` through `0x7f` directly, so they don't need a separate tag byte.

To encode `300` as an unsigned integer, we first write `0xcd` (the `uint16` type) and then write the two bytes of the integer in big-endian order.

{{< diagram >}}
{{< image src="layout-msgpack.svg" class="center" inline="true" >}}
{{< /diagram >}}

```go
b0 := src[i]
if b0 <= 127 {
	dst[n] = uint64(b0)
	i++
} else {
	switch b0 {
	case 0xcc:
		dst[n] = uint64(src[i+1])
		i += 2
	case 0xcd:
		dst[n] = uint64(binary.BigEndian.Uint16(src[i+1:]))
		i += 3
	case 0xce:
		dst[n] = uint64(binary.BigEndian.Uint32(src[i+1:]))
		i += 5
	case 0xcf:
		dst[n] = binary.BigEndian.Uint64(src[i+1:])
		i += 9
	}
}
```

The switch here looks a lot like the one for QUIC, except we're spending a whole byte on the type rather than a couple of bits from the integer.

### Fixed64

I also want to have a baseline for all of these benchmarks where we just write out the eight bytes of each `uint64` in little-endian order.

{{< diagram >}}
{{< image src="layout-fixed64.svg" class="center" inline="true" >}}
{{< /diagram >}}

There's an interesting opportunity to cheat a bit here. On a little-endian machine, the bytes we want to write are already laid out correctly in memory. We can use `unsafe.Slice` to treat our integers as a slice of bytes and just call `copy`.

I'll benchmark both that version (`Fixed64 Unsafe`) and a regular Go loop (`Fixed64`).

```go
func DecodeFixed64(src []byte, dst []uint64) int {
	n := len(src) / 8
	dst = dst[:n]
	for j := range dst {
		dst[j] = binary.LittleEndian.Uint64(src[:8])
		src = src[8:]
	}
	return n
}

func DecodeFixed64Unsafe(src []byte, dst []uint64) int {
	n := len(src) / 8
	if n == 0 {
		return 0
	}
	dst = dst[:n]
	out := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(dst))), n*8)
	copy(out, src[:n*8])
	return n
}
```

I originally wrote that loop with `src[j*8:]`, but reslicing with `src[:8]` and `src = src[8:]` lets the Go compiler drop a bounds check inside `binary.LittleEndian.Uint64`. That one change made Fixed64 noticeably faster.

### Other Schemes

There are a few other varint encodings that I didn't benchmark.

Group Varint (Varint-GB) encodes four integers together. The first byte contains four 2-bit lengths, so we can figure out where all four integers are before we read any of them. This is particularly useful for decoding arrays with SIMD instructions like `pshufb` or `tbl`. It doesn't fit quite as nicely into formats where we're encoding one integer at a time.

[CBOR](https://www.rfc-editor.org/rfc/rfc8949.html#section-3) is similar to MessagePack, except its first byte packs a 3-bit major type with a 5-bit value. Anything from `0` to `23` lives right in those 5 bits, and values `24` through `27` mean a 1, 2, 4, or 8-byte integer follows.

[SQLite4](https://sqlite.org/src4/doc/trunk/www/varint.wiki) went a different route: its varints sort in the same order whether you compare the raw bytes or the decoded unsigned integers, which is handy for database indexes.

### Format Summary

| Format | Widths (B) | Endian | 1-Byte | Max |
| :--- | :--- | :---: | :---: | :---: |
| **LEB128** | 1–10 | LE | `0..127` | 64-bit |
| **QUIC** | 1, 2, 4, 8 | BE | `0..63` | 62-bit |
| **OmniBuf** | 1, 2, 4, 9 | LE | `0..127` | 64-bit |
| **Prefix-9** | 1–9 | LE | `0..127` | 64-bit |
| **MessagePack** | 1, 2, 3, 5, 9 | BE | `0..127` | 64-bit |
| **Fixed64** | 8 | LE | None | 64-bit |

## Benchmark Setup

All of these benchmarks run on my M1 Pro MacBook with Go 1.25.5 (`-count=3 -benchtime=250ms`), and the numbers below are the means across those three runs. Each benchmark processes 10,000 integers at a time and reuses buffers small enough to stay in CPU cache.

My hand-rolled decoders also cheat a bit. They assume the input is valid and the output slice is already large enough, so they skip the bounds and error checks that `encoding/binary` and `protowire` have to run on every call.

I generated six datasets of 10,000 integers each:

| Dataset | Values |
| :--- | :--- |
| `Small` | Random integers from 0 to 63 |
| `Large` | Random 41-bit integers around `2^40` |
| `FullWidth` | Uniformly distributed random `uint64` values |
| `Skewed` | 70% small, 20% medium, 8% big, 2% huge (shuffled) |
| `MixedRandom` | 25% of each of the four sizes, shuffled |
| `MixedSorted` | The same integers as `MixedRandom`, sorted |

For `Skewed` and the two mixed datasets, I'm using these ranges:

| Bucket | Range |
| :--- | :--- |
| Small | `0..127` |
| Medium | `128..16,127` |
| Big | `65,536..100M` |
| Huge | Around `2^40` |

I used 63 as the upper limit for `Small` so that even QUIC can encode everything in one byte. I've left QUIC out of the `FullWidth` benchmarks because it can't encode the entire `uint64` range.

## Wire Size

Here is the encoded byte count for 10,000 `uint64` values across each dataset:

{{< tabs >}}
{{< tab name="Skewed" >}}
{{< chart sort="asc" unit="B" >}}
{
  "type": "bar",
  "data": {
    "labels": ["LEB128", "Prefix-9", "OmniBuf", "MsgPack", "QUIC", "Fixed64"],
    "datasets": [
      {
        "label": "Bytes",
        "data": [15386, 15386, 16009, 18812, 19266, 80000],
        "backgroundColor": [
          "rgba(255, 107, 74, 0.85)",
          "rgba(148, 163, 184, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(168, 85, 247, 0.85)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Size: Skewed (bytes)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="Mixed" >}}
{{< chart sort="asc" unit="B" >}}
{
  "type": "bar",
  "data": {
    "labels": ["LEB128", "Prefix-9", "QUIC", "OmniBuf", "MsgPack", "Fixed64"],
    "datasets": [
      {
        "label": "Bytes",
        "data": [32706, 32706, 39135, 40491, 45440, 80000],
        "backgroundColor": [
          "rgba(255, 107, 74, 0.85)",
          "rgba(148, 163, 184, 0.85)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(168, 85, 247, 0.85)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Size: Mixed (bytes)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="Large (2^40)" >}}
{{< chart sort="asc" unit="B" >}}
{
  "type": "bar",
  "data": {
    "labels": ["LEB128", "Prefix-9", "QUIC", "Fixed64", "OmniBuf", "MsgPack"],
    "datasets": [
      {
        "label": "Bytes",
        "data": [60000, 60000, 80000, 80000, 90000, 90000],
        "backgroundColor": [
          "rgba(255, 107, 74, 0.85)",
          "rgba(148, 163, 184, 0.85)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(168, 85, 247, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(251, 191, 36, 0.85)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Size: Large (bytes)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="FullWidth" >}}
{{< chart sort="asc" unit="B" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64", "Prefix-9", "OmniBuf", "MsgPack", "LEB128"],
    "datasets": [
      {
        "label": "Bytes",
        "data": [80000, 89967, 90000, 90000, 94988],
        "backgroundColor": [
          "rgba(168, 85, 247, 0.85)",
          "rgba(148, 163, 184, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(255, 107, 74, 0.85)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Size: FullWidth 64-bit (bytes)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="Small" >}}
{{< chart sort="asc" unit="B" >}}
{
  "type": "bar",
  "data": {
    "labels": ["LEB128", "Prefix-9", "QUIC", "OmniBuf", "MsgPack", "Fixed64"],
    "datasets": [
      {
        "label": "Bytes",
        "data": [10000, 10000, 10000, 10000, 10000, 80000],
        "backgroundColor": [
          "rgba(255, 107, 74, 0.85)",
          "rgba(148, 163, 184, 0.85)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(168, 85, 247, 0.85)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Size: Small (bytes)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< /tabs >}}

<details>
<summary><b>Show wire size table</b></summary>

| Codec | Small | Large (`2^40`) | FullWidth | Skewed | Mixed |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **LEB128** | **10,000 B** | **60,000 B** | 94,988 B | **15,386 B** | **32,706 B** |
| **Prefix-9** | **10,000 B** | **60,000 B** | 89,967 B | **15,386 B** | **32,706 B** |
| **OmniBuf** | **10,000 B** | 90,000 B | 90,000 B | 16,009 B | 40,491 B |
| **MsgPack** | **10,000 B** | 90,000 B | 90,000 B | 18,812 B | 45,440 B |
| **QUIC** | **10,000 B** | 80,000 B | N/A | 19,266 B | 39,135 B |
| **Fixed64** | 80,000 B | 80,000 B | **80,000 B** | 80,000 B | 80,000 B |

</details>

The `Skewed` dataset is roughly what I had in mind when designing OmniBuf's varints. I gave up five of Prefix-9's possible byte widths, and it only cost me about 4% in this dataset.

That decision looks considerably worse on `Large`. OmniBuf needs nine bytes for every integer, while LEB128 and Prefix-9 only need six.

There's also not much point in using a varint for something like a random 64-bit hash. On `FullWidth`, all of these varint formats make the data bigger than it was to begin with.

## Decode Speed

{{< tabs >}}
{{< tab name="Skewed" >}}
{{< chart sort="asc" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Fixed64", "OmniBuf", "QUIC", "MsgPack", "Prefix-9 (fast1B)", "LEB128 (fast1B)", "Prefix-9", "LEB128 (protowire)", "LEB128 (stdlib)"],
    "datasets": [
      {
        "label": "ns/op",
        "data": [954, 7410, 8560, 12557, 13662, 14117, 18691, 28690, 37062, 40458],
        "backgroundColor": [
          "rgba(192, 132, 252, 0.95)",
          "rgba(168, 85, 247, 0.55)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(148, 163, 184, 0.95)",
          "rgba(255, 107, 74, 0.85)",
          "rgba(148, 163, 184, 0.55)",
          "rgba(255, 107, 74, 0.6)",
          "rgba(255, 107, 74, 0.35)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Decode: Skewed (ns/op)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="Small" >}}
{{< chart sort="asc" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Prefix-9 (fast1B)", "MsgPack", "OmniBuf", "LEB128 (fast1B)", "Fixed64", "QUIC", "LEB128 (stdlib)", "LEB128 (protowire)", "Prefix-9"],
    "datasets": [
      {
        "label": "ns/op",
        "data": [974, 6336, 6389, 6466, 6470, 7753, 8087, 15901, 22082, 29115],
        "backgroundColor": [
          "rgba(192, 132, 252, 0.95)",
          "rgba(148, 163, 184, 0.95)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(255, 107, 74, 0.85)",
          "rgba(168, 85, 247, 0.55)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(255, 107, 74, 0.35)",
          "rgba(255, 107, 74, 0.6)",
          "rgba(148, 163, 184, 0.55)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Decode: Small (ns/op)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="Large" >}}
{{< chart sort="asc" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Fixed64", "QUIC", "MsgPack", "OmniBuf", "Prefix-9 (fast1B)", "Prefix-9", "LEB128 (protowire)", "LEB128 (fast1B)", "LEB128 (stdlib)"],
    "datasets": [
      {
        "label": "ns/op",
        "data": [992, 7547, 11275, 15851, 16490, 29864, 30033, 31976, 34520, 54440],
        "backgroundColor": [
          "rgba(192, 132, 252, 0.95)",
          "rgba(168, 85, 247, 0.55)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(148, 163, 184, 0.95)",
          "rgba(148, 163, 184, 0.55)",
          "rgba(255, 107, 74, 0.6)",
          "rgba(255, 107, 74, 0.85)",
          "rgba(255, 107, 74, 0.35)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Decode: Large (ns/op)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="FullWidth" >}}
{{< chart sort="asc" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Fixed64", "Prefix-9", "Prefix-9 (fast1B)", "MsgPack", "OmniBuf", "LEB128 (protowire)", "LEB128 (fast1B)", "LEB128 (stdlib)"],
    "datasets": [
      {
        "label": "ns/op",
        "data": [956, 7424, 9955, 10608, 15612, 15647, 67850, 79632, 105586],
        "backgroundColor": [
          "rgba(192, 132, 252, 0.95)",
          "rgba(168, 85, 247, 0.55)",
          "rgba(148, 163, 184, 0.55)",
          "rgba(148, 163, 184, 0.95)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(255, 107, 74, 0.6)",
          "rgba(255, 107, 74, 0.85)",
          "rgba(255, 107, 74, 0.35)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Decode: FullWidth 64-bit (ns/op)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="MixedRandom" >}}
{{< chart sort="asc" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Fixed64", "OmniBuf", "QUIC", "Prefix-9 (fast1B)", "Prefix-9", "MsgPack", "LEB128 (fast1B)", "LEB128 (protowire)", "LEB128 (stdlib)"],
    "datasets": [
      {
        "label": "ns/op",
        "data": [929, 7456, 18940, 20417, 27659, 28701, 39606, 50911, 60640, 71539],
        "backgroundColor": [
          "rgba(192, 132, 252, 0.95)",
          "rgba(168, 85, 247, 0.55)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(148, 163, 184, 0.95)",
          "rgba(148, 163, 184, 0.55)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(255, 107, 74, 0.85)",
          "rgba(255, 107, 74, 0.6)",
          "rgba(255, 107, 74, 0.35)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Decode: MixedRandom (ns/op)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="MixedSorted" >}}
{{< chart sort="asc" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Fixed64", "QUIC", "OmniBuf", "MsgPack", "LEB128 (fast1B)", "Prefix-9 (fast1B)", "LEB128 (protowire)", "Prefix-9", "LEB128 (stdlib)"],
    "datasets": [
      {
        "label": "ns/op",
        "data": [983, 7698, 10070, 10695, 11801, 19431, 23217, 25660, 28708, 33553],
        "backgroundColor": [
          "rgba(192, 132, 252, 0.95)",
          "rgba(168, 85, 247, 0.55)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(255, 107, 74, 0.85)",
          "rgba(148, 163, 184, 0.95)",
          "rgba(255, 107, 74, 0.6)",
          "rgba(148, 163, 184, 0.55)",
          "rgba(255, 107, 74, 0.35)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Decode: MixedSorted (ns/op)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< /tabs >}}

<details>
<summary><b>Show full decode table (ns/op for 10,000 values; Skewed ns/int is the average time per integer)</b></summary>

| Codec | Small | Large | FullWidth | Skewed | Skewed (ns/int) | MixedRandom | MixedSorted |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Fixed64 Unsafe** | **974** | **992** | **956** | **954** | **0.10** | **929** | **983** |
| **Fixed64** | 7,753 | 7,547 | 7,424 | 7,410 | 0.74 | 7,456 | 7,698 |
| **OmniBuf** | 6,466 | 16,490 | 15,647 | 8,560 | 0.86 | 18,940 | 10,695 |
| **QUIC** | 8,087 | 11,275 | N/A | 12,557 | 1.26 | 20,417 | 10,070 |
| **MsgPack** | 6,389 | 15,851 | 15,612 | 13,662 | 1.37 | 39,606 | 11,801 |
| **Prefix-9 (fast1B)** | 6,336 | 29,864 | 10,608 | 14,117 | 1.41 | 27,659 | 23,217 |
| **LEB128 (fast1B)** | 6,470 | 34,520 | 79,632 | 18,691 | 1.87 | 50,911 | 19,431 |
| **Prefix-9** | 29,115 | 30,033 | 9,955 | 28,690 | 2.87 | 28,701 | 28,708 |
| **LEB128 (protowire)** | 22,082 | 31,976 | 67,850 | 37,062 | 3.71 | 60,640 | 25,660 |
| **LEB128 (stdlib)** | 15,901 | 54,440 | 105,586 | 40,458 | 4.05 | 71,539 | 33,553 |

</details>

On `Small`, every hand-rolled decoder with a 1-byte fast path takes around 6.4 µs: Prefix-9 (fast1B), MsgPack, OmniBuf, and LEB128 (fast1B). Neither library implementation matches the hand-rolled fast path on `Small`; function-call overhead and input validation are likely contributors. On `FullWidth`, protowire's unrolling beats both other LEB128 decoders, but it is still over 4x slower than OmniBuf and nearly 7x slower than Prefix-9. Prefix-9 has an advantage here because it checks for the 9-byte escape first, while OmniBuf has to check the other three widths before reaching that case.

I should point out the scale of these numbers. The difference between OmniBuf and my LEB128 (fast1B) decoder on `Skewed` works out to about one nanosecond per integer. If your Protobuf message has five integer fields, you probably have more important things to optimize.

## Encode Speed

{{< tabs >}}
{{< tab name="Skewed" >}}
{{< chart sort="asc" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Fixed64", "Prefix-9", "QUIC", "OmniBuf", "MsgPack", "LEB128 (fast1B)", "LEB128 (stdlib)", "LEB128 (protowire)"],
    "datasets": [
      {
        "label": "ns/op",
        "data": [939, 3946, 15241, 17137, 17479, 21493, 21993, 26113, 39831],
        "backgroundColor": [
          "rgba(192, 132, 252, 0.95)",
          "rgba(168, 85, 247, 0.55)",
          "rgba(148, 163, 184, 0.85)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(255, 107, 74, 0.85)",
          "rgba(255, 107, 74, 0.35)",
          "rgba(255, 107, 74, 0.6)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Encode: Skewed (ns/op)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="Small" >}}
{{< chart sort="asc" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Fixed64", "MsgPack", "LEB128 (fast1B)", "Prefix-9", "QUIC", "OmniBuf", "LEB128 (stdlib)", "LEB128 (protowire)"],
    "datasets": [
      {
        "label": "ns/op",
        "data": [928, 3924, 6316, 6320, 6400, 6503, 6539, 9479, 20480],
        "backgroundColor": [
          "rgba(192, 132, 252, 0.95)",
          "rgba(168, 85, 247, 0.55)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(255, 107, 74, 0.85)",
          "rgba(148, 163, 184, 0.85)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(255, 107, 74, 0.35)",
          "rgba(255, 107, 74, 0.6)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Encode: Small (ns/op)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="Large" >}}
{{< chart sort="asc" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Fixed64", "QUIC", "OmniBuf", "MsgPack", "Prefix-9", "LEB128 (stdlib)", "LEB128 (protowire)", "LEB128 (fast1B)"],
    "datasets": [
      {
        "label": "ns/op",
        "data": [937, 3924, 15623, 18967, 21878, 21934, 31494, 34597, 34668],
        "backgroundColor": [
          "rgba(192, 132, 252, 0.95)",
          "rgba(168, 85, 247, 0.55)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(148, 163, 184, 0.85)",
          "rgba(255, 107, 74, 0.35)",
          "rgba(255, 107, 74, 0.6)",
          "rgba(255, 107, 74, 0.85)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Encode: Large (ns/op)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="FullWidth" >}}
{{< chart sort="asc" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Fixed64", "OmniBuf", "MsgPack", "Prefix-9", "LEB128 (stdlib)", "LEB128 (fast1B)", "LEB128 (protowire)"],
    "datasets": [
      {
        "label": "ns/op",
        "data": [944, 4054, 18748, 23736, 34579, 68311, 73782, 76063],
        "backgroundColor": [
          "rgba(192, 132, 252, 0.95)",
          "rgba(168, 85, 247, 0.55)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(148, 163, 184, 0.85)",
          "rgba(255, 107, 74, 0.35)",
          "rgba(255, 107, 74, 0.85)",
          "rgba(255, 107, 74, 0.6)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Encode: FullWidth 64-bit (ns/op)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="MixedRandom" >}}
{{< chart sort="asc" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Fixed64", "Prefix-9", "OmniBuf", "QUIC", "LEB128 (fast1B)", "MsgPack", "LEB128 (stdlib)", "LEB128 (protowire)"],
    "datasets": [
      {
        "label": "ns/op",
        "data": [970, 4320, 33720, 34309, 34977, 45089, 47485, 50517, 55254],
        "backgroundColor": [
          "rgba(192, 132, 252, 0.95)",
          "rgba(168, 85, 247, 0.55)",
          "rgba(148, 163, 184, 0.85)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(255, 107, 74, 0.85)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(255, 107, 74, 0.35)",
          "rgba(255, 107, 74, 0.6)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Encode: MixedRandom (ns/op)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< tab name="MixedSorted" >}}
{{< chart sort="asc" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Fixed64", "OmniBuf", "QUIC", "Prefix-9", "MsgPack", "LEB128 (stdlib)", "LEB128 (fast1B)", "LEB128 (protowire)"],
    "datasets": [
      {
        "label": "ns/op",
        "data": [1008, 4192, 12100, 12533, 13540, 16082, 19714, 21482, 25525],
        "backgroundColor": [
          "rgba(192, 132, 252, 0.95)",
          "rgba(168, 85, 247, 0.55)",
          "rgba(45, 212, 191, 0.85)",
          "rgba(56, 189, 248, 0.85)",
          "rgba(148, 163, 184, 0.85)",
          "rgba(251, 191, 36, 0.85)",
          "rgba(255, 107, 74, 0.35)",
          "rgba(255, 107, 74, 0.85)",
          "rgba(255, 107, 74, 0.6)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Encode: MixedSorted (ns/op)" },
      "legend": { "display": false }
    },
    "scales": { "x": { "min": 0 } }
  }
}
{{< /chart >}}
{{< /tab >}}
{{< /tabs >}}

<details>
<summary><b>Show full encode table (ns/op)</b></summary>

| Codec | Small | Large | FullWidth | Skewed | MixedRandom | MixedSorted |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: |
| **Fixed64 Unsafe** | **928** | **937** | **944** | **939** | **970** | **1,008** |
| **Fixed64** | 3,924 | 3,924 | 4,054 | 3,946 | 4,320 | 4,192 |
| **OmniBuf** | 6,539 | 18,967 | 18,748 | 17,479 | 34,309 | 12,100 |
| **QUIC** | 6,503 | 15,623 | N/A | 17,137 | 34,977 | 12,533 |
| **Prefix-9** | 6,400 | 21,934 | 34,579 | 15,241 | 33,720 | 13,540 |
| **MsgPack** | 6,316 | 21,878 | 23,736 | 21,493 | 47,485 | 16,082 |
| **LEB128 (fast1B)** | 6,320 | 34,668 | 73,782 | 21,993 | 45,089 | 21,482 |
| **LEB128 (stdlib)** | 9,479 | 31,494 | 68,311 | 26,113 | 50,517 | 19,714 |
| **LEB128 (protowire)** | 20,480 | 34,597 | 76,063 | 39,831 | 55,254 | 25,525 |

</details>

All of the hand-rolled varint encoders take around 6.4 µs on `Small`, while Fixed64 stays flat around 4 µs in safe Go and around 1 µs with `memmove`. On `FullWidth`, the LEB128 encoders are roughly 4x slower than OmniBuf.

## Branch Prediction

Look back at all those decoder implementations. In the LEB128 loop, we check a bit every time we read a byte. In OmniBuf, QUIC, and MessagePack, we have a switch to decide how many bytes to read.

If we're decoding a long series of small integers, we'll keep taking the same branch. The CPU can predict this and start executing instructions from the next iteration before it has finished the current one. If the widths are constantly changing, the CPU has a much harder time figuring out what to execute next.

Comparing `MixedSorted` against `MixedRandom` gives us a good indication of how sensitive each decoder is to branch prediction. Both inputs contain the exact same 10,000 integers and produce the exact same total byte count. Only the order changes.

{{< chart sort="false" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["Fixed64 Unsafe", "Fixed64", "OmniBuf", "QUIC", "Prefix-9 (fast1B)", "Prefix-9", "MsgPack", "LEB128 (fast1B)", "LEB128 (protowire)", "LEB128 (stdlib)"],
    "datasets": [
      {
        "label": "Sorted",
        "data": [983, 7698, 10695, 10070, 23217, 28708, 11801, 19431, 25660, 33553],
        "backgroundColor": "rgba(45, 212, 191, 0.85)",
        "barPercentage": 0.85,
        "categoryPercentage": 0.75
      },
      {
        "label": "Shuffled",
        "data": [929, 7456, 18940, 20417, 27659, 28701, 39606, 50911, 60640, 71539],
        "backgroundColor": "rgba(255, 107, 74, 0.85)",
        "barPercentage": 0.85,
        "categoryPercentage": 0.75
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Decode: Sorted vs Shuffled (ns/op)" }
    },
    "scales": {
      "x": { "min": 0 }
    }
  }
}
{{< /chart >}}

MessagePack is the most affected by shuffling (3.4x slower), followed by the LEB128 implementations (2.1x to 2.6x). QUIC and OmniBuf take roughly twice as long. Prefix-9 (fast1B) only slows down by 19%, while the original Prefix-9 and both Fixed64 implementations are largely unaffected.

### Why Branchless Decoding Was Slower

Prefix-9 stayed around 29 µs whether the input was sorted or shuffled. Even on `MixedRandom`, though, it still lost to OmniBuf. And on `FullWidth`, where nearly every integer skips the branchless path and takes the 9-byte escape, it suddenly ran almost three times as fast.

So what happened here? I expected the branchless version to do much better on `MixedRandom`.

Part of the problem is a loop-carried data dependency: there's quite a bit of work between reading the first byte of an integer and figuring out where the next integer starts. We have to count the trailing zeros and calculate the width before we know the next offset.

Compare that with OmniBuf's 1-byte case. Once the CPU predicts that branch, the next offset is just `i + 1`. It doesn't have to wait for the previous load to finish before figuring out where to read next.

We're also doing more work to decode the integer itself. In OmniBuf's 1-byte case, we read a byte and shift it right once. Prefix-9 reads eight bytes, counts trailing zeros, and does two variable shifts, even if the integer is just `1`.

Adding a 1-byte fast path to Prefix-9 fixed most of that on `Small` and cut `Skewed` in half while keeping the same wire size. I haven't measured CPU performance counters to confirm how much each factor contributes, but the speed of the 9-byte escape path points in the same direction.

## Fixed-Width Isn't Bad

For all this effort trying to encode integers efficiently, it's worth looking at what happens when we don't do any of it.

With a bit of `unsafe` Go code, we can copy all 80 KB of `Fixed64` data in around a microsecond. That's more than 50 times as fast as my `LEB128 (fast1B)` decoder on `MixedRandom`. Even with safe Go, the fixed-width implementation wins five of the six decode benchmarks against the varint implementations and all of the encode benchmarks.

There's another advantage that none of the benchmarks capture. If I want the 500th integer in an array of `uint64` values, I know exactly where it is. With any of these varint formats, I'd need to scan through the preceding integers or maintain some sort of index.

### What about compression?

What if we're compressing the data anyway? Do we really need to worry about all those extra zeros?

I used `github.com/klauspost/compress` (`v1.19.2`) with the default `zstd` compression level (`SpeedDefault`, roughly level 3). I'm reusing the encoder, decoder, and output buffers between benchmark iterations.

First, the compressed payload sizes (for 10,000 `uint64` values):

| Codec | Small | Skewed | FullWidth |
| :--- | :---: | :---: | :---: |
| **LEB128 (protowire)** | 10,014 B | **15,400 B** | 95,004 B |
| **OmniBuf** | 10,014 B | 15,428 B | 90,016 B |
| **Fixed64 Unsafe** | **7,690 B** | 22,039 B | **80,016 B** |

I wasn't expecting the result on `Small`. Even after running zstd, the LEB128 and OmniBuf buffers are 14 bytes larger than they were before (`10,014 B`), which is consistent with a zstd frame containing an uncompressed block. Meanwhile, zstd takes our 80 KB of fixed-width integers down to 7,690 B.

We're generating random numbers between 0 and 63 here, so there are only six bits of entropy per integer (7,500 B of information in the entire array). At `SpeedDefault`, `klauspost/compress` turns off entropy coding when there are no repeated sequences to match (see [`WithAllLitEntropyCompression`](https://pkg.go.dev/github.com/klauspost/compress/zstd#WithAllLitEntropyCompression)). Because the 1-byte varint values are random, zstd doesn't find enough repetition and just writes a raw block instead. With Fixed64, every value has seven zero bytes in a row, so zstd finds plenty of matches and can compress the remaining 6-bit values down close to that 7,500 B limit.

It doesn't work out quite so well on `Skewed`. We're still sending about 43% more bytes with Fixed64 than with OmniBuf, even after compression. The 4% wire-size difference between OmniBuf and LEB128 also almost completely disappears after compression: there's only a 28-byte difference between them now. And we haven't accounted for the time spent compressing and decompressing the data yet.

{{< chart sort="false" unit="ns" >}}
{
  "type": "bar",
  "data": {
    "labels": ["LEB128 (protowire)", "OmniBuf", "Fixed64 Unsafe"],
    "datasets": [
      {
        "label": "Encode + zstd",
        "data": [42888, 154255, 392237],
        "backgroundColor": "rgba(255, 107, 74, 0.85)",
        "barPercentage": 0.85,
        "categoryPercentage": 0.72
      },
      {
        "label": "zstd + Decode",
        "data": [39326, 12883, 102567],
        "backgroundColor": "rgba(45, 212, 191, 0.85)",
        "barPercentage": 0.85,
        "categoryPercentage": 0.72
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": { "display": true, "text": "Skewed with zstd (ns/op)" }
    },
    "scales": {
      "x": { "min": 0 }
    }
  }
}
{{< /chart >}}

We've gone from copying 80 KB in around a microsecond to spending roughly 390 µs encoding and compressing it. Decoding is now over 100 µs. I'd much rather pay for the varint decoding in this case.

The same thing happens with LEB128 and OmniBuf on `Skewed`. LEB128 grew by 14 bytes (`15,386 B` to `15,400 B`), consistent with zstd writing it uncompressed. That's why `Encode + zstd` barely took any longer than raw encoding. OmniBuf shrank from `16,009 B` to `15,428 B`, probably because of the extra zero bytes in its 9-byte `Huge` values. In that case, zstd actually ran the full compressor and burned a lot of CPU to save 581 bytes.

## Which Encoding Would I Use?

I'm still happy with my choice of varint encoding for OmniBuf. And after running these benchmarks, I'd be much more inclined to use fixed-width integers for large arrays of numbers.

All benchmark code, compression tests, and round-trip tests are [on GitHub](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/variable-length-integers/benchmarks) alongside this post.

Prefix-varints are just one of the wire-format decisions I made for [OmniBuf](/posts/omnibuf/). There are some equally interesting tradeoffs in how it encodes collections, nested messages, and wire types. I'll cover those in future posts.
