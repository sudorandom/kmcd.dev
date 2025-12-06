---
categories: ["article", "tutorial"]
tags: ["networking", "grpc", "go", "golang", "tutorial", "protobuf"]
series: ["gRPC from Scratch"]
date: "2024-05-07"
description: "Let's look under the hood of gRPC by getting into the weeds of protocol buffers."
cover: "cover.jpg"
images: ["/posts/grpc-from-scratch-part-3/cover.jpg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC From Scratch: Part 3 - Protobuf Encoding"
slug: "grpc-from-scratch-part-3"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-from-scratch-part-3/
mastodonID: "112398336658297977"
---

In the last two parts, I showed how to make an extremely simple gRPC client and server that... kind-of works. But I punted on a topic last time that is pretty important: I used generated protobuf types and the Go protobuf library to do all of the heavy lifting of encoding and decoding protobufs for me. That ends today. I'll start by looking at the [`protowire`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protowire) library directly, which is a bit closer to what is actually happening on the wire. The library includes a fun disclaimer:

> For marshaling and unmarshaling entire protobuf messages, use the google.golang.org/protobuf/proto package instead.

Am I going to listen to this solid advice? No! I want to know how this works! No reflection and no reliance on generated code. All of the code in this post are taken from unit tests [available here](https://github.com/sudorandom/kmcd.dev/blob/main/content/posts/2024/grpc-from-scratch-part-3/go/protowire_test.go) so feel free to download the tests and play around with it locally. Now, let's get started.

## Wire Types
I discuss in my [Inspecting Protobuf Messages](/posts/inspecting-protobuf-messages/) post that protobuf only has a small handful of types. Here they are again:

| ID | Name | Used for|
| ----------- | ------- | -------|
| 0 | VARINT | int32, int64, uint32, uint64, sint32, sint64, bool, enum |
| 1 | I64 | fixed64, sfixed64, double |
| 2 | LEN | string, bytes, embedded messages, packed repeated fields |
| 3 | SGROUP | group start (deprecated) |
| 4 | EGROUP | group end (deprecated) |
| 5 | I32 | fixed32, sfixed32, float |

For today we're only going to implement some of the `LEN` and `VARINT` wire types. Protobuf messages are a series of key-value pairs. Keys are "field numbers" and values are one of the types in the table above. To save space, protobuf decided to encode both the field number and wire type into a single byte. The lower three bits are the wire type and the other 5 (and maybe more, more on that later) are used for the field number. This is what the "Tag-Length" part looks like for field number `1` with the `VARINT` wire type:

```plaintext
0000 1000
```

You can shift the three least significant bits off to get the protobuf wire type and the rest is used for the field number. It is at this point that the bitwise operators start becoming useful:
```plaintext
(field_number << 3) | wire_type
```

So... 3 bits for the wire type leaves room for 8 different options so protobuf has room for two more wire types before they have to break compatibility with older versions or add another byte to describe more types. And 5 bits for the field number which leaves room for...... **32 different fields**??? What?! You can't have a message that has more than 32 fields?! Why is no one talking about this *glaring* limitation where the number of fields is limited to a four-year-old's counting ability?! Well, obviously this is not true and, at this point, I am now forced to explain what protobuf refers to as `Base 128 Varints`.

### Big numbers
In the previous example, we saw `VARINT` take a single byte. What does it look like when your number is too big? Where does the variableness part of `VARINT` come in? The protobuf encoding uses what it calls Base-128 Variable Integers in several places. "Base 128" means you can count to 127 before rolling over to the next "digit" (or in this case, byte). The most significant bit is used as a continuation bit, which is a signal that there's at least one more byte worth of data to complete this integer. Let's decode one for practice:

```plaintext
11000000 11000100 00000111   // Original inputs.
 1000000  1000100  0000111   // Drop continuation bits.
 0000111  1000100  1000000   // Convert to big-endian.
 000011110001001000000       // Concatenate.
 (1 × 2^16) + (1 × 2^15) +   // Convert binary to decimal
 (1 × 2^14) + (1 × 2^13) +   // because we're not a computer.
 (1 × 2^9) + (1 × 2^6)
 = 123456                    // Interpret as an unsigned 64-bit integer.
```

Next, I will show you some go code that can do this `VARINT` encoding process for us. Note that this code is actually in Go's [standard library](https://pkg.go.dev/encoding/binary) and makes reference to protocol-buffers directly [in the documentation](https://pkg.go.dev/encoding/binary); ([source](https://github.com/golang/go/blob/go1.22.2/src/encoding/binary/varint.go#L39-L47)).

```go
// AppendUvarint appends the varint-encoded form of x,
// as generated by [PutUvarint], to buf and returns the extended buffer.
func AppendUvarint(buf []byte, x uint64) []byte {
	for x >= 0x80 {
		buf = append(buf, byte(x)|0x80)
		x >>= 7
	}
	return append(buf, byte(x))
}
```

What's happening here? Let's break it down:
- The core logic is in a `for` loop that continues as long as x is greater than or equal to 128 (represented by `0x80` in hexadecimal). Why 128? That's the highest number (in hexadecimal) that you can count to only using 7 bits. Remember that the most significant bit (MSB) is used as a continuation bit so we can only use 7 out 8 of the available bits in each byte.
	- The next line appends the `uint64` argument truncated to a byte, so the last 8 bits. One of those bits isn't actually used because of the `|0x80` part of the line. This combines our extracted 7 bits with `0x80` to ensure that the continuation bit is set to true.
	- The next line uses the right shift operator to shift 7 bits off of our current `uint64` value because we've "dealt with" these bits already. Next, we check the for-loop condition again until we get a number lower than `128`.
- Finally, we append the final byte as-is because we know that it's less than 128 which assures us that the MSB is not set, so we are done building this integer.

Next, we have the decoding code for VARINT. That looks like this ([source](https://github.com/golang/go/blob/go1.22.2/src/encoding/binary/varint.go#L69-L88)):

```go
// Uvarint decodes a uint64 from buf and returns that value and the
// number of bytes read (> 0). If an error occurred, the value is 0
// and the number of bytes n is <= 0 meaning:
//
//	n == 0: buf too small
//	n  < 0: value larger than 64 bits (overflow)
//	        and -n is the number of bytes read
func Uvarint(buf []byte) (uint64, int) {
	var x uint64
	var s uint
	for i, b := range buf {
		if i == MaxVarintLen64 {
			// Catch byte reads past MaxVarintLen64.
			// See issue https://golang.org/issues/41185
			return 0, -(i + 1) // overflow
		}
		if b < 0x80 {
			if i == MaxVarintLen64-1 && b > 1 {
				return 0, -(i + 1) // overflow
			}
			return x | uint64(b)<<s, i + 1
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
	return 0, 0
}
```
- It loops through each byte:
   - Checks for overflow (reading past expected max size).
   - Handles the last byte (if the continuation bit is `0`):
     - Validates it and returns the decoded value.
   - Processes continuation bytes:
     - Extracts data bits and combines them with the accumulated value, shifting them to their correct position. The expression `b&0x7f` ensures that the continuation bit is not included in the result by using a bitwise AND with the current byte and `0x7f`, which looks like this in binary `01111111`.
     - `s` keeps track of the total bits already processed. It increments by 7 because that's how many bits we can use for each byte because of the continuation bit.
- If the loop finishes without a valid ending, it signals an error.

As a short aside, you should probably know that the modern protobuf library doesn't use these functions from the `encoding/binary` standard library package. I suspect that they were initially used but the current [`protowire`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protowire) implementation doesn't use `for` loops at all because it has performed an optimization technique called [loop unrolling](https://en.wikipedia.org/wiki/Loop_unrolling). Here's [AppendVarint](https://github.com/protocolbuffers/protobuf-go/blob/v1.33.0/encoding/protowire/wire.go#L184-L263) and [ConsumeVarint](https://github.com/protocolbuffers/protobuf-go/blob/v1.33.0/encoding/protowire/wire.go#L265-L367) that are now used in the modern protobuf library. That code looks insane but it is likely a bit faster than the version of the code that I just showed you.

As a test for comprehension of this last section, you should now understand that smaller numbers take up less room on the wire. That's true for all numeric types except for the fixed-length types: `I64` and `I32`. Encoding `20` in protobbuf takes a single byte on the wire but `1234` would take two bytes.

## Integers
Now that we know how the `VARINT` wire-type works we now have enough raw material to write some protobuf packets. Again, we need three things to make a message with a single field: Field number, wire type, and the encoded value. As mentioned earlier, the field number and wire type are merged into a single `VARINT` value using the formula: `(field_number << 3) | wire_type`. This essentially means that the the least significant bits are reserved for the wire type and the rest are used for the field number, and we expand using the Base-128 varint method above if the field number needs more bits to be represented. Now, let's write a full field in protobuf! Here's how we encode a message with a single int32 field:

```plaintext
1: 1234
```

```go
var buf []byte
buf = protowire.AppendVarint(buf, uint64(1<<3)|uint64(0))
buf = protowire.AppendVarint(buf, 1234)
```

And... that's it. We've fully encoded probably the simplest (non-empty) protobuf message. We can check that it works by running a test with the "real" protobuf library:

```go
func TestEncodeRaw(t *testing.T) {
	var buf []byte
	buf = protowire.AppendVarint(buf, uint64(1<<3)|uint64(0))
	buf = protowire.AppendVarint(buf, 1234)

	res := gen.TestMessage{}
	require.NoError(t, proto.Unmarshal(buf, &res))
	assert.Equal(t, int32(1234), res.IntValue)
}
```

By the way, these tests are using a protobuf file that looks like this:
{{% render-code file="go/types.proto" language="protobuf" %}}
All of the field numbers and corresponding types in this article match up to this protobuf file.

Okay, let's show what it looks like to read this message:

```go
tagNumber, protoType, n := protowire.ConsumeTag(b)
...
b = b[n:]
i, n := protowire.ConsumeVarint(b)
```
Note that I omitted error handling and in a real scenario we need to check the `tagNumber` and `protoType` to decide how to read this field.

## Strings/byte arrays
Next, we're going to start with strings and byte arrays. These use the `LEN` wire type. This type is a little more complex than `VARINT`. It's composed of a `VARINT` that tells us the size of the `LEN` field followed by the actual content in bytes. For `LEN` this content can be a string, a byte array, an embedded message or packed repeated fields. Here's what a field looks like:
- Field Tag Byte (field number along with the field type, set to `2` for the `LEN` type)
- Byte size of our content as a `VARINT`
- The actual content

Encode:
```go
var buf []byte
buf = protowire.AppendTag(buf, protowire.Number(4), protowire.BytesType)
buf = protowire.AppendString(buf, "Hello World!")
```

Decode:
```go
tagNumber, protoType, n := protowire.ConsumeTag(b)
b = b[n:]
i, n := protowire.ConsumeVarint(b)
```

## Integer Arrays (Packed)

Packed repeated fields are a space-saving optimization for integer arrays. It's a feature that's enabled by default for most `repeated` scalar types when using the proto3 syntax. Instead of encoding each element individually as a separate field/value pair an entire array is packed into a single `LEN` type. Therefore it looks like the following for repeated `int32` types:
- `VARINT` of our field number with the three least significant bits being reserved for field type for `LEN` which is `2`. This is the
- The raw integer data, also encoded with VARINT for our `int32` type, one after another

Here's an example of how to encode a packed repeated integer field:
```go
arr := []int32{100002130, 2, 3, 4, 5}
var buf, buf2 []byte
buf = protowire.AppendVarint(buf, uint64(10<<3)|uint64(2))
for i := 0; i < len(arr); i++ {
	buf2 = protowire.AppendVarint(buf2, uint64(arr[i]))
}
buf = protowire.AppendVarint(buf, uint64(len(buf2)))
buf = append(buf, buf2...)
```
Notice that we are writing the list of `int32` values to a second buffer. I do this so that we can know the size of the encoded/packed `int32` values so that I can properly set the size for the encapsulating `LEN` wire type.

Decoding a packed repeated field is similar. First, you read the field tag (like always). Then we are using `protowire.ConsumeBytes` that reads the entire `LEN` value into a byte array. Then you read `varint` values until you exhaust the buffer. Remember that `varint` values have that continuation but so the `protowire.ConsumeVarint` function knows when to finish reading each `varint` value.

Here's what that looks like in code:

```go
tagNumber, protoType, n := protowire.ConsumeTag(b)
...
b = b[n:]
int32buf, n := protowire.ConsumeBytes(b)
b = b[n:]
...
res := []int32{}
for len(int32buf) > 0 {
	v, n := protowire.ConsumeVarint(int32buf)
	int32buf = int32buf[n:]
	res = append(res, int32(v))
}
```

## Conclusion

In this part of the series, we've taken a deep dive into the world of manual protobuf encoding. We explored a few wire types and the primitives that they are built on, like Base-128 varints, and built functions to encode and decode basic protobuf data types. While for most use cases, you'll probably want to leverage the efficiency and convenience of generated code and libraries like `golang/protobuf`, understanding how to manually encode these messages can be a valuable asset for debugging, creating a protobuf library implementation in your favorite new language or for possibly creating your own binary encoding that's better in some way than protobuf.

In the next part of the series, we will put more pieces together with a layer on top of our encoding code and integrate this protobuf library into the client and server that we made in parts 1 and 2. Build a real gRPC client and server that uses protobufs for data exchange!
