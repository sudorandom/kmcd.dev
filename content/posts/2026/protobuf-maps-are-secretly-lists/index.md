---
categories: ["article", "engineering"]
tags: ["protobuf", "omnibuf", "serialization", "wire-format", "performance"]
keywords: ["protobuf map encoding", "protobuf wire format", "binary serialization", "omnibuf"]
date: "2026-12-15T10:00:00Z"
description: "Protobuf doesn't have a map wire type. Every map<K, V> desugars into a repeated list of synthetic entry messages that adds 4 bytes of framing to every kv pair."
cover: "cover.svg"
images: ["/posts/protobuf-maps-are-secretly-lists/cover.svg"]
linktitle: ""
title: "Protobuf maps are secretly lists"
slug: "protobuf-maps-are-secretly-lists"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/protobuf-maps-are-secretly-lists/
---

You may not know this, but the Protobuf binary encoding doesn't actually support maps. Well, it does, kind of, through a hack one layer up. Maps were added well after Protobuf was established, alongside proto3. To avoid changing the wire format, Protobuf libraries [convert `map<K, V>` into a repeated message](https://protobuf.dev/programming-guides/proto3/#backwards) with a `key` field and a `value` field. So this:

```protobuf
message LatencyTable {
  map<string, uint32> rtt_ms = 1;
}
```

Actually looks like this on the wire:

```protobuf
message LatencyTable {
  message RttMsEntry {
    string key = 1;
    uint32 value = 2;
  }
  repeated RttMsEntry rtt_ms = 1;
}
```

Google went this route so proto2 libraries wouldn't immediately explode when seeing maps for the first time, because on the wire maps just looked like a list of entry messages. At the time, this was a nice tradeoff: proto2 was forward-compatible with proto3. But now, I think it's a wart that will be very hard to remove and it actually makes maps pretty inefficient on the wire due to the extra submessage wrapping that has to happen for each map entry.

Let's use a 5-entry `map<string, uint32>` as an example:

```json
{"fra": 12, "ams": 8, "iad": 24, "nrt": 95, "syd": 142}
```

{{< figure src="map-wire-comparison.svg" alt="Wire layout comparison of a 5-entry map in Protobuf versus OmniBuf" >}}

Using Protobuf, those five entries are encoded as 46 bytes on the wire with 20 bytes being map-specific framing. Fun fact: this is actually the exact same size as the JSON version with the whitespace removed. In [OmniBuf](https://omnibuf.com/), the same map is 31 bytes.

But wire size isn't the only reason that this setup isn't ideal. The decoder also has to parse every map entry like a nested message (because it is one). One more issue is that Protobuf maps (and lists) don't have an entry count as part of the wire data, so the decoder can't make the simple optimization of pre-allocating the map. It just has to grow and rehash as entries come in.

For normal structs with scalar fields, OmniBuf payloads are usually a bit larger than Protobuf. Maps and arrays are a different story. OmniBuf maps work by writing a header containing the field tag, byte length, key type, value type, and entry count before the map, which is typically 5 bytes long in total. Following that, all of the key/value pairs are written back to back as raw values with no per-entry field tags, since we already know the types. This kind of structure works well for a decoder because it can pre-allocate the map in memory at the start and completely avoid growing it. It also reduces conditional branches by knowing the types up-front. These are small improvements but they add up to a much more performant decoder.

I've been advised that compression makes wire size "not an issue" for text and binary formats. While compression saves bytes on the wire, it does come at a CPU performance cost. Here's what a 50-entry `map<string, uint32>` looks like going through gzip and zstd for JSON, Protobuf and OmniBuf:

| Format | Raw | Gzip (`-9`) | Zstd (level 3) |
| :--- | ---: | ---: | ---: |
| **JSON** | 884 B | 257 B | 195 B |
| **Protobuf** | 867 B | 263 B | 210 B |
| **OmniBuf** | **673 B** | **236 B** | **185 B** |

Even after compression, OmniBuf is still the smallest. Protobuf actually ends up a bit bigger than JSON with both gzip and zstd, because all those per-entry tags and length bytes don't compress down as much as I expected. If you want to learn more about OmniBuf, check out [omnibuf.com](https://omnibuf.com/)!
