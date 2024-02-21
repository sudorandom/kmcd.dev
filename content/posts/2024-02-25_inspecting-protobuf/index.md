+++
categories = ["article"]
tags = ["networking", "grpc", "http", "tutorial", "protobuf"]
date = "2024-02-25"
description = ""
cover = "cover.jpg"
images = ["/posts/inspecting-protobuf-messages/social.jpg"]
featured = ""
featuredalt = ""
featuredpath = "date"
linktitle = ""
title = "Inspecting Protobuf Messages"
slug = "inspecting-protobuf-messages"
type = "posts"
draft = true
+++


[Protocol Buffers](https://protobuf.dev/) is an amazing message format. It's [incredibly compact](https://nilsmagnus.github.io/post/proto-json-sizes/) and [performant](https://medium.com/@akresling/go-benchmark-json-v-protobuf-4ec3c62ec8d4). However, these advantages come at a cost. Since Protobuf is a binary format it lacks a lot in readability compared to text-based formats like JSON or XML.

However, all hope is not lost. Even if you just have a binary protobuf file with no knowledge of the corresponding protobuf file we can still get some information out of it. Let me introduce a tool called [Protoscope](https://github.com/protocolbuffers/protoscope). Protoscope is a tool for inspecting protobuf binary. It can do this with or without the protobuf files or the equivalent [descriptor set](https://protobuf.com/docs/descriptors) (but it can do a better job with the protobuf data).

-----
### Install Protoscope
Okay, let's hit the ground running. Here's how to install protoscope ([requires go](https://go.dev/dl/)).
```bash
go install github.com/protocolbuffers/protoscope/cmd/protoscope@latest
```

### Using Protoscope
If you have a binary protobuf file, here's what you can run to get protoscope output:
```bash
# protoscope [filename]
# variety.pb contains binary protobuf content.
$ protoscope -explicit-wire-types variety.pb
{{% import file="go/testdata/variety.txt" %}}

# Often times you might see binary files encoded with hexadecimal.
$ cat variety.pb.hex
{{% import file="go/testdata/variety.pb.hex" %}}

# We can use xxd to convert it to binary then pipe it into protoscope
$ xxd -r -ps variety.pb.hex | protoscope -explicit-wire-types
... [same as above] ...
```
-----

Note that with the examples above protoscope needs to guess types because we did not pass in the `-descriptor-set` and `-message-type` options. Why does protoscope need to guess types? One thing that will help with understanding this topic is knowing that *protobuf encoding only has _6 types_.* And two of them aren't even used in the latest version. You may be saying to yourself "I remember seeing a [table of protobuf types](https://protobuf.dev/programming-guides/proto3/#scalar) and it had way more than 6!" and you would be correct. Although protobufs support many types they are all serialized into 6 "wire types.":

| ID | Name | Used for|
| ----------- | ------- | -------|
| 0 | VARINT | int32, int64, uint32, uint64, sint32, sint64, bool, enum |
| 1 | I64 | fixed64, sfixed64, double |
| 2 | LEN | string, bytes, embedded messages, packed repeated fields |
| 3 | SGROUP | group start (deprecated) |
| 4 | EGROUP | group end (deprecated) |
| 5 | I32 | fixed32, sfixed32, float |

How protobuf encodes each protobuf type into a wire type differs depending on the type. The full explanation exists in [the programming guide for the protobuf encoding](https://protobuf.dev/programming-guides/encoding/). I recommend reading through it to fully understand the implications of using the protoscope tool.

### Strings
Let's see what it looks like with a trivial example that is sourced from [connectrpc.eliza.v1.SayRequest](https://buf.build/connectrpc/eliza/docs/main:connectrpc.eliza.v1#connectrpc.eliza.v1.SayRequest) `(protoscope -explicit-wire-types eliza.SayRequest.pb)`:

{{% render-code file="go/testdata/eliza.SayRequest.txt" language="text" %}}

What does this tell us? Well, it tells us the message has a single field with field number 1. It also tells us the value of the string is `"World"`. **That's pretty incredible compared to the nothing we knew about this collection of bytes a second ago!** Let me quickly explain protobuf field numbers. The `1` in this example is a field number and it corresponds to the protobuf [field number](https://protobuf.com/docs/language-spec#field-numbers). For repeated values, you may see this number appear multiple times. But notice how the name is completely missing. That's because protobuf doesn't want to waste resources transmitting or storing metadata like that. It would be fair to summarize protobuf as a list of key/value pairs. The key is the field number and the values are one of the basic types in protobuf.

For the record, here's what [connectrpc.eliza.v1.SayRequest](https://buf.build/connectrpc/eliza/docs/main:connectrpc.eliza.v1#connectrpc.eliza.v1.SayRequest) looks like:

```protobuf
message SayRequest {
	string sentence = 1;
}
```

> Disclaimer: Protoscope has to guess types in a lot of instances because the protobuf encoding has only 6 wire types (two deprecated): `VARINT`, `I64`, `LEN`, and `I32` are used with modern protobuf files. In the example above, field 1 could have been a string, byte array, an embedded message, or packed repeated fields. Protoscope **guessed** that it was a string and showed it to us as a string. It *can* guess wrong.

### More Strings
Okay, now we're going to look at a message derived from a different type: [connectrpc.eliza.v1.IntroduceRequest](https://buf.build/connectrpc/eliza/docs/main:connectrpc.eliza.v1#connectrpc.eliza.v1.IntroduceRequest) `(protoscope -explicit-wire-types eliza.IntroduceRequest.pb)`:

{{% render-code file="go/testdata/eliza.IntroduceRequest.txt" language="text" %}}

Wait, what? It's the *exact* same? Yep. If the field numbers and types match there's no distinguishable difference when protobuf is encoded into binary. Here's the protobuf type:

```protobuf
message IntroduceRequest {
	string name = 1;
}
```

Notice the message contains a single string field which is similar to `SayRequest`` above, but there are a few notable differences. The message name and the field name are different but since those two things are never transmitted over the wire with protobuf we can't tell the difference between these two message types without prior knowledge. This flexibility allows you to make certain significant changes to your protobuf file without changing what is encoded... but you do have to [follow some rules](https://earthly.dev/blog/backward-and-forward-compatibility/). These rules make more sense with more knowledge of the protobuf encoding.

### Bytes
Okay, now let's look at a new type: bytes. Let's take a look at what that looks like with a byte array `(protoscope -explicit-wire-types bytes.pb)`):

{{% render-code file="go/testdata/bytes.txt" language="text" %}}

What you're seeing here is a hexadecimal representation of the bytes in our field. In this example, we mostly have random data. But if you pass this hexadecimal text through a hex-to-string converter you may notice that the beginning text, `736563726574`, decodes to `secret`. Even though there is ASCII string content in the byte array, protoscope still treats this as a byte array, not a string. Here's a byte array with a string that says `Hello World!` as the content `(protoscope -explicit-wire-types bytes2.pb)`:

{{% render-code file="go/testdata/bytes2.txt" language="text" %}}

Wait, what? Protoscope renders the text as text! What gives? Protoscope is, again, guessing the type of the data. It notices that all of the included bytes are in the ASCII range so it renders the content as text. Could this be the wrong thing to do? Maybe!

## Numbers

Now let's at numbers represented in protobuf. You may see some... odd things. We'll break it down field by field `(protoscope -explicit-wire-types numbers.pb)`.

{{% render-code file="go/testdata/numbers.txt" language="text" %}}

Refer to this table to see the actual protobuf types and the intended values. You will notice that several of them don't match up with what protoscope outputs at all.

| Field Number | Actual Type | Actual Value|
| ----------- | ------- | -------|
| 2 | enum | 3 (AnEnum.C) |
| 3 | uint32 | 175 |
| 5 | repeated uint64 | 1, 2, 3, 4 |
| 7 | int64 | 921 |
| 8 | bool | true |
| 9 | float | 1.2345 |
| 19 | I32 | -1, -2, -3, -4 |

- **2**: Enum fields are encoded as numbers on the wire. because of this, the name of the enum value may be unknown to you without the protobuf file.
- **3**: uint32 types look like what you'd expect! Nice.
- **5**: This is the first super weird one. Why is it shown as a string? This has to do with [packed repeated fields](https://protobuf.dev/programming-guides/encoding/#packed). The protobuf encoding packs repeated primitive types into a single `LEN` field (instead of using `VARINT`, `I64` or `I32` as normal). Therefore, protoscope may simply represent this as a string or byte array because it can't tell the difference on the wire.
- **7**: int64 types also look like what you'd expect! Nice.
- **8**: Booleans are encoded as false = `0` and true = `1`.
- **9**: In this case, protobufs treated the float correctly and we get the correct value.
- **19**: This has to be the weirdest case. This looks so strange because it's a result of using `repeated int32` type, [which is packed](https://protobuf.dev/programming-guides/encoding/#packed) with negative values. Protoscope thinks this value looks like a `LEN` wire type with binary data in it. It guessed the type incorrectly this time.

## Submessages and Maps
In protobuf you can put messages instead of other messages, so let's look at what that looks like from protoscope's perspective `(protoscope -explicit-wire-types submessages.pb)`:
{{% render-code file="go/testdata/submessages.txt" language="text" %}}

From the protoscope output above you might also notice that we have two field `18` values. That is because submessages cannot be packed as primitive types can. The "unpacked" way of representing a repeated value in the protobuf encoding is to simply write the field multiple times with different values. Simple. So field `18` is likely a repeated submessage field.


Now let's look at maps: `(protoscope -explicit-wire-types maps.pb)`
{{% render-code file="go/testdata/maps.txt" language="text" %}}

Wait, what? This looks a lot like submessages! There's a reason for that! Maps ARE submessages in the protobuf encoding. Here's basically what the encoder is doing.

A map that looks like this:
```protobuf
message TestWithMap {
  map<string, int32> name_to_age = 7;
}
```

... is converted into a submessage that looks like this:

```protobuf
message TestWithMap {
  message name_to_age_Entry {
    optional string key = 1;
    optional int32 value = 2;
  }
  repeated name_to_age_Entry name_to_age = 7;
}
```

This shows that protobuf is a very practical encoding that re-uses basic concepts to support more complex structures.

## Summary
I didn't cover all of the weird edge cases. There are features in the, now deprecated, proto2 format that I didn't show. However, I hope that I've shown that you can get *something* from a binary protobuf file. This, alone, is quite impressive for a binary format. You would usually have a very hard time understanding anything without knowledge of the specific binary protocol. This demonstrates how protobufs takes some of the benefits you might get from text-based encodings (composability, support for "unknown" fields, some amount of discoverability) with the performance of binary formats (speed, reduced size) but protobuf does bring in an extra ingredient: contracts. Because protobuf files are the source of truth for the format and type-safe serialization code, gRPC client code, gRPC server code, and documentation can all be generated from protobuf files this shows the strength of the format... which is why you should try to never be in a situation where you NEED to use protoscope. You should always have a [descriptor set](https://protobuf.com/docs/descriptors) or the protobuf files nearby to decode these messages.

For a more extensive overview of the protobuf binary encoding refer to [the official documentation](https://protobuf.dev/programming-guides/encoding/).
