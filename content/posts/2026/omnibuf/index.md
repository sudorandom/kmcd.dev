---
title: "OmniBuf"
date: "2026-11-24T10:00:00Z"
categories: ["article", "project"]
tags: ["protobuf", "omnibuf", "grpc", "connectrpc", "performance", "golang", "serialization"]
keywords: ["omnibuf", "binary serialization", "protobuf alternative", "wire format", "prefix varints", "self-describing binary", "grpc codec"]
description: "What happens if you keep .proto schemas and replace Protobuf's binary wire format underneath?"
cover: "cover.svg"
images: ["/posts/omnibuf/cover.svg"]
slug: "omnibuf"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/omnibuf/
---

Protobuf's binary format is old. As a result, it has some warts: Decoding is slow partially because of byte-by-byte varints. Raw payloads are opaque enough that building inspection tools is difficult even for a binary format. Language runtimes behave differently and have wildly different performance characteristics. Add that to the fact that Protobuf requires buying into a lot of the toolchain up front without a smooth onramp.

Protocol Buffers, AKA Protobuf, actually refers to two things: the schema language (where you have `.proto` files defining messages, field numbers, and services) and the binary wire format. Both have their warts and issues but today we need to focus on the wire format, mostly. It's also important to note that Protobuf already has alternate, officially supported, encodings. ProtoJSON defines a standard way to represent Protobuf messages as JSON. ProtoText does the same but in a different human-readable format.

After my [gRPC From Scratch](/series/grpc-from-scratch/) series, I internalized how dead simple the Protobuf binary encoding actually is. I also [benchmarked varints against fixed-size integers in Go](/posts/protobuf-varint-vs-fixed/), which clearly showed that Protobuf-flavored varints weren't the most performant datatype. After toying around with the binary encoding, I wanted to see what Protobuf would look like with different tradeoffs, more modern encoding techniques and the ability to blind-decode message values. What if we kept `.proto` files and services, but swapped out the binary wire format underneath? That experiment turned into an encoding that I call **[OmniBuf](https://omnibuf.com)**.

OmniBuf is a self-describing binary wire format built to work with `.proto` schemas, gRPC, and Connect, as well as plain native types in code. It keeps what makes Protobuf good for schema evolution, while changing the wire encoding to address these four pain points:

- **Faster on modern CPUs:** Replaces the slow byte-by-byte LEB128 varints with branchless [prefix-varints](https://omnibuf.com/how-it-works#prefix-varints), encodes nested messages in a [single forward pass](https://omnibuf.com/how-it-works#single-pass), writes primitive arrays contiguously and encodes [maps natively](https://omnibuf.com/how-it-works#native-maps).
- **Blind-decodable without a schema:** Expands the wire format to [16 explicit types](https://omnibuf.com/how-it-works#wire-types) (plus optional [Field #0 field names](https://omnibuf.com/how-it-works#field-0-names)), so a CLI, proxy, or debugger can inspect a raw `.obuf` payload and get exact types without the original Protobuf schema or guessing like protoscope does.
- **Cheap partial reads and edits:** If you just want to read and write a field or two, you can now [skip full message decoding](https://omnibuf.com/how-it-works#skip-fields) to read a single field and you can splice an updated value back in much faster than decoding the full message.
- **Smoother onramp and consistent runtimes:** Annotated structs with field-number tags can emit the exact same bytes as `.proto` codegen, so you can start with native types and graduate to fully-fledged `.proto` files later (or not!)

{{< diagram >}}
{{< figure src="overview.svg" alt="Diagram showing peer.proto branching to Protobuf, JSON, and OmniBuf encodings alongside native language files pointing directly to OmniBuf" caption="OmniBuf works both as an alternate encoding for `.proto` schemas and directly on native types across 11 languages." loading="lazy" >}}
{{< /diagram >}}

## Partial Reads and Splicing

In Protobuf, skipping ahead to a later field essentially means parsing all varint continuation bits one byte at a time and since varints are often used as numeric values, you also have to parse those as well to know when they end and the next field begins. There's a neat trick that you can do if you have a byte prefix for every field. Suddenly, you can skip over fields that you don't care about. You can append fields to the end of the message without any extra work. In OmniBuf, the reader hops directly from [field to field](https://omnibuf.com/how-it-works#skip-fields):

{{< diagram caption="Skipping fields on the wire: Protobuf checks continuation bits byte by byte, while OmniBuf jumps straight past containers using their header lengths." >}}
{{< image src="skip-fields.svg" class="center" inline="true" >}}
{{< /diagram >}}

We can also use these length-delimited fields to splice in an edited value without parsing and re-encoding the rest of the message. There is a full walkthrough of how this works [on the website](https://omnibuf.com/#read-or-splice):

```go
var view pb.OpaquePeerUpdate
_ = view.Unmarshal(pkt)
router := view.GetRouterId() // skips earlier fields by byte length
view.SetActive(false)        // marks field #3 dirty
pkt = view.MarshalAppend(nil)
```

## Plain Structs

I want to address an elephant in the room when talking about Protobuf. It's weird. You're defining types in a specialized language and need tooling to generate types in the language that you actually want to use. I can understand how that will turn people off from the format altogether. I feel like there's a step in-between that would allow you to use the serialization format but not require the song and dance of Protobuf code generation.

With OmniBuf, you can use language features like [Go struct tags](https://omnibuf.com/go#struct-tags) (`omnibuf:"1"`), Rust derive macros, and [TypeScript defineMessage](https://omnibuf.com/es#define-message) definitions to emit the exact same messages as `.proto` generated code. The idea is that you can start with types that are native to your language of choice and then move to `.proto` files later with no change to the data on the wire:

```go
// From .proto
pkt, _ := omnibuf.Marshal(&pb.Node{Id: 1, City: "FRA"})

// From a plain Go struct
type Node struct {
    ID   uint32 `omnibuf:"1"`
    City string `omnibuf:"2"`
}
pkt, _ := omnibuf.Marshal(&Node{ID: 1, City: "FRA"})
```

This gives you an on-ramp to OmniBuf without having to buy into the schema-first design philosophy. I deeply think that schema-first design is worth it, but I don't want that to get in the way of using a more efficient message format.

## Benchmarks and Tradeoffs

Okay, so how fast is OmniBuf, really? I've talked about the few tricks I've used to make it more efficient but I'm genuinely surprised at how effective it has been to improve the encoding and decoding speed. From my benchmarks, it quite often comes out ahead of all of the formats that I've tested.

{{< tabs >}}
  {{< tab name="Full Decode (vs Protobuf)" >}}
{{< chart sort="desc" unit="x" >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Go",
      "Dart",
      "PHP",
      "TypeScript / JS",
      "Elixir",
      "C++",
      "C# / .NET",
      "Swift",
      "Rust",
      "Java",
      "Python"
    ],
    "datasets": [
      {
        "label": "Speedup vs Protobuf (higher is better)",
        "data": [5.2, 4.8, 3.7, 2.9, 2.9, 2.6, 2.3, 2.3, 2.0, 2.0, 1.8],
        "backgroundColor": "rgba(0, 168, 150, 0.88)",
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Full Decode Speedup vs Protobuf (PeerUpdate, higher is better)"
      },
      "legend": {
        "display": false
      }
    },
    "scales": {
      "x": {
        "type": "linear",
        "min": 0,
        "title": {
          "display": true,
          "text": "Speedup (x faster than Protobuf)"
        }
      }
    }
  }
}
{{< /chart >}}
  {{< /tab >}}
  {{< tab name="Full Encode (vs Protobuf)" >}}
{{< chart sort="desc" unit="x" >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Elixir",
      "Go",
      "Dart",
      "Swift",
      "C# / .NET",
      "Rust",
      "TypeScript / JS",
      "C++",
      "PHP",
      "Java",
      "Python"
    ],
    "datasets": [
      {
        "label": "Speedup vs Protobuf (higher is better)",
        "data": [7.4, 5.4, 3.9, 3.8, 1.8, 1.6, 1.5, 1.4, 1.4, 1.2, 1.2],
        "backgroundColor": "rgba(255, 84, 73, 0.88)",
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Full Encode Speedup vs Protobuf (PeerUpdate, higher is better)"
      },
      "legend": {
        "display": false
      }
    },
    "scales": {
      "x": {
        "type": "linear",
        "min": 0,
        "title": {
          "display": true,
          "text": "Speedup (x faster than Protobuf)"
        }
      }
    }
  }
}
{{< /chart >}}
  {{< /tab >}}
{{< /tabs >}}

See the [benchmark page](https://omnibuf.com/benchmarks) for all of the benchmarks.

Other formats still beat OmniBuf in a few areas. Protobuf is slightly smaller on dense messages. FlatBuffers and Cap'n Proto are faster at reading a single field from large messages by using offset tables. JSON loses on every aspect except being readable. It's the only format of the list that can be read without assistance.

## Languages and Tooling

There are first-party runtimes and buf / protoc plugins for 11 languages:

- [Go](https://omnibuf.com/go)
- [TypeScript / JS](https://omnibuf.com/es)
- [Rust](https://omnibuf.com/rs)
- [Python](https://omnibuf.com/py)
- [C++](https://omnibuf.com/cpp)
- [Java](https://omnibuf.com/java)
- [C# / .NET](https://omnibuf.com/cs)
- [Swift](https://omnibuf.com/swift)
- [Dart](https://omnibuf.com/dart)
- [PHP](https://omnibuf.com/php)
- [Elixir](https://omnibuf.com/elixir)

I also made a [CLI](https://omnibuf.com/cli) to convert OmniBuf messages to and from JSON. You can also run jq queries against the binary messages.

```bash
brew install omnibuf/tap/omnibuf
# or: mise use -g github:omnibuf/omnibuf-cli

omnibuf < event.json > event.obuf
omnibuf -r '.neighbors[] | select(.established) | .router' event.obuf
```

I also made editor extensions for VS Code ([omnibuf-vscode](https://github.com/omnibuf/omnibuf-vscode)) and Vim / Neovim ([omnibuf-vim](https://github.com/omnibuf/omnibuf-vim)).


{{< bigtext >}}[omnibuf.com](https://omnibuf.com){{< /bigtext >}}

{{< github-repo repo="omnibuf" description="Fast, self-describing binary wire format for Protocol Buffers and native types." >}}