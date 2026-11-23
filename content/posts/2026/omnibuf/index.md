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

After my [gRPC From Scratch](/series/grpc-from-scratch/) series, I realized how simple the Protobuf binary encoding actually is. My [benchmarks comparing varints and fixed-size integers in Go](/posts/protobuf-varint-vs-fixed/) got me experimenting with the binary encoding. I wanted to see what Protobuf would look like with different tradeoffs, more modern encoding techniques, and the ability to blind-decode message values. What if we kept `.proto` files and services, but swapped out the binary wire format underneath? That experiment turned into an encoding that I call **[OmniBuf](https://omnibuf.com)**.

Protobuf's schema language and wire format are separate. A `.proto` file defines messages and services, while an encoding determines how message values are stored or sent. Protobuf already supports other encodings, including ProtoJSON and Text Format.

OmniBuf is a self-describing binary wire format built to work with `.proto` schemas, gRPC, and Connect, as well as plain native types in code. You can keep your `.proto` schemas and service definitions, though using OmniBuf for RPCs requires registering an OmniBuf codec on both the client and server. It keeps what makes Protobuf good for schema evolution, while changing the wire encoding to make a few different tradeoffs:

- **Faster on modern CPUs:** Replaces byte-by-byte LEB128 varints with [prefix-varints](https://omnibuf.com/how-it-works#prefix-varints) whose encoded length can be determined from the first byte, encodes nested messages in a [single forward pass](https://omnibuf.com/how-it-works#single-pass), and encodes arrays and [maps natively](https://omnibuf.com/how-it-works#native-maps) as length-prefixed containers.
- **Blind-decodable without a schema:** Expands the wire format to [16 explicit types](https://omnibuf.com/how-it-works#wire-types) (plus optional [Field #0 field names](https://omnibuf.com/how-it-works#field-0-names)), so a CLI, proxy, or debugger can inspect a raw `.obuf` payload and distinguish strings, bytes, numbers, booleans, arrays, maps, and nested messages without the original Protobuf schema or guessing like protoscope does.
- **Cheap partial reads and edits:** If you just want to read and write a field or two, you can [skip full message decoding](https://omnibuf.com/how-it-works#skip-fields) to read a single field and splice an updated value back in much faster than decoding the full message.
- **Smoother on-ramp:** Annotated structs with matching field numbers and types use the same OmniBuf wire format as generated code, so you can start with native types and move to `.proto` files later (or not!).

{{< diagram >}}
{{< figure src="overview.svg" alt="Diagram showing peer.proto branching to Protobuf, JSON, and OmniBuf encodings alongside native language files pointing directly to OmniBuf" caption="OmniBuf works both as an alternate encoding for `.proto` schemas and directly on native types across 11 languages." loading="lazy" >}}
{{< /diagram >}}

## Partial Reads and Splicing

Protobuf already lets you skip length-delimited fields like strings, bytes, and nested messages once you read their length. Integer varints take more work because you have to scan continuation bits one byte at a time to find where each number ends, and maps or non-packed repeated fields are split across separate tagged entries. OmniBuf puts an integer's encoded width in its first byte and wraps whole arrays and maps as length-prefixed containers, so the reader hops directly from [field to field](https://omnibuf.com/how-it-works#skip-fields):

{{< diagram caption="Skipping varint tags and values on the wire: Protobuf checks continuation bits byte by byte, while OmniBuf reads the width from the first byte and jumps straight ahead." >}}
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

I want to address an elephant in the room when talking about Protobuf. It's weird. You're defining types in a specialized language and need tooling to generate types in the language that you actually want to use. I can understand how that will turn people off from the format altogether. I wanted a way to use the format without the song and dance of Protobuf code generation.

With OmniBuf, you can use language features like [Go struct tags](https://omnibuf.com/go#struct-tags) (`omnibuf:"1"`), Rust derive macros, and [TypeScript defineMessage](https://omnibuf.com/es#define-message) definitions to emit the exact same messages as `.proto` generated code. The idea is that you can start with types that are native to your language of choice and then move to `.proto` files later with no change to the data on the wire:

```go
// From .proto
protoPkt, _ := omnibuf.Marshal(&pb.Node{Id: 1, City: "FRA"})

// From a plain Go struct
type Node struct {
    ID   uint32 `omnibuf:"1"`
    City string `omnibuf:"2"`
}
structPkt, _ := omnibuf.Marshal(&Node{ID: 1, City: "FRA"})
```

This gives you an on-ramp to OmniBuf without having to buy into the schema-first design philosophy. I think schema-first design is worth it, but I don't want that to get in the way of using a more efficient message format.

## Benchmarks and Tradeoffs

Okay, so how fast is OmniBuf, really? I expected these changes to help. I didn't expect gains this large.

These charts compare OmniBuf's generated codecs with Protobuf implementations in 11 languages, using an 11-field `PeerUpdate` message on an Apple M1 Pro. The Protobuf baselines include `google.golang.org/protobuf` in Go, `prost` in Rust, and `protobuf-java` in Java. Full methodology, library versions, and non-codegen struct benchmarks are on the [benchmark page](https://omnibuf.com/benchmarks).

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

Other formats still beat OmniBuf in a few areas. On the `PeerUpdate` fixture, Protobuf produces 341 bytes to OmniBuf's 366 bytes. OmniBuf is also still early and experimental compared to Protobuf's two decades in production. FlatBuffers and Cap'n Proto are faster at reading a single field from large messages by using offset tables. JSON loses on every benchmarked metric here (encode speed, decode speed, and payload size), though it is still the only format on the list you can read directly as text.

## Languages and Tooling

There are first-party runtimes and plugins for Buf and protoc across 11 languages, all verified against the same [conformance suite](https://omnibuf.com/spec#sec-11-2) so behavior stays consistent across ecosystems:

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

I also made a [CLI](https://omnibuf.com/cli) to convert OmniBuf messages to and from JSON, and to inspect raw `.obuf` payloads without a schema. Because the wire types don't collide, blind-decoding a packet gives you typed values instead of protoscope's guesses:

```bash
# Blind-decoding a raw packet with no .proto file:
$ omnibuf -c packet.obuf
{"field_1": true, "field_2": -2, "field_3": "FRA", "field_4": [10, 20]}

# With optional Field #0 metadata enabled:
$ omnibuf -c packet-field0.obuf
{"active": true, "delta": -2, "city": "FRA", "peers": [10, 20]}
```

You can also pipe JSON in or run jq queries directly against binary messages:

```bash
brew install omnibuf/tap/omnibuf
# or: mise use -g github:omnibuf/omnibuf-cli

omnibuf < event.json > event.obuf
omnibuf -r '.neighbors[] | select(.established) | .router' event.obuf
```

I also made editor extensions for VS Code ([omnibuf-vscode](https://github.com/omnibuf/omnibuf-vscode)) and Vim / Neovim ([omnibuf-vim](https://github.com/omnibuf/omnibuf-vim)).


{{< bigtext >}}[omnibuf.com](https://omnibuf.com){{< /bigtext >}}

{{< github-repo repo="omnibuf" description="Fast, self-describing binary wire format for Protocol Buffers and native types." >}}