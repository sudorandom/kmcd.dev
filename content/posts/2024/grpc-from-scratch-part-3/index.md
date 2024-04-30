---
categories: ["article", "tutorial"]
tags: ["networking", "grpc", "go", "golang", "tutorial", "protobuf"]
date: "2024-05-07"
description: "We've made the world's simplest gRPC client and server for unary RPCs. Now let's tackle ~protobuf encoding~."
cover: "cover.jpg"
images: ["/posts/grpc-from-scratch-part-3/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC From Scratch: Part 3 - Protobuf Encoding"
slug: "grpc-from-scratch-part-3"
type: "posts"
devtoSkip: true
canonical_url: https://sudorandom.dev/posts/grpc-from-scratch-part-3
draft: true
---

> This is part three of a series. [Click here to see gRPC From Scratch: Part 1 where I build a simple gRPC client](/posts/grpc-from-scratch/) and [gRPC From Scratch: Part 2 where I build a simple gRPC server.](/posts/grpc-from-scratch-part-2/)

In the last two parts, I showed how to make an extremely simple gRPC client and server that... kind-of works. But I punted on a topic last time that is pretty important: I used generated protobuf types and the Go protobuf library to do all of the heavy lifting of encoding and decoding protobufs for me. That ends today. I'll start by looking at the [`protowire`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protowire) library directly, which is a bit closer to what is actually happening on the wire. The library includes a fun disclaimer:

> For marshaling and unmarshaling entire protobuf messages, use the google.golang.org/protobuf/proto package instead.

Am I going to listen to this solid advice? No! I want to know how this works! No reflection and no reliance on generated code. So let's get started:

## Wire Types
I discuss this in my [Inspecting Protobuf Messages](/posts/inspecting-protobuf-messages/) post that protobuf only has a small handful of types. Here they are again:

| ID | Name | Used for|
| ----------- | ------- | -------|
| 0 | VARINT | int32, int64, uint32, uint64, sint32, sint64, bool, enum |
| 1 | I64 | fixed64, sfixed64, double |
| 2 | LEN | string, bytes, embedded messages, packed repeated fields |
| 3 | SGROUP | group start (deprecated) |
| 4 | EGROUP | group end (deprecated) |
| 5 | I32 | fixed32, sfixed32, float |

For today we're only going to implement some of the `LEN` and `VARINT` wire types. Protobuf messages are a series of key-value pairs. Keys are "field numbers" and values are one of the types in the table above. But since this is a binary format we also need to know the byte length of the key-value pair if the type doesn't have a fixed lenght. We need this to know where the value ends and the next one begins. This kind of encoding is very common. So common, in fact, that there's a name for the generic form: [Tag-Length-Value or TLV](https://en.wikipedia.org/wiki/Type%E2%80%93length%E2%80%93value). To save space, protobuf decided to encode both the field number and wire type into a single byte. The lower three bits are the wire type and the other 5 (and maybe more) are used for the field number. This is what the "Tag-Length" part looks like for field number `1` with the `VARINT` wire type:

```
0000 1000
```

You can shift the three least significant bits off to get the protobuf wire type and the rest is used for the field number. It is at this point where the bitwise operators start becoming useful:
```
(field_number << 3) | wire_type
```

So... 3 bits for the wire type leaves room for 8 different options so protobuf has room for two more wire types before they have to break compatibility with older versions. And 5 bits for the field number which leaves room for...... **32 different fields**??? What?! You can't have a message that has more than 32 fields?! Why is no one talking about this *glaring* limitation in protobufs?! At this point, I am now forced to explain what protobuf calls `Base 128 Varints`.

### Big numbers
In the previous example, we saw `VARINT` take a single byte. What does it look like when your number is too big? Where does the variableness come in? The protobuf encoding uses what it calls Base-128 Variable Integers. "Base 128" means you can count to 127 before rolling over to the next "digit" (or in this case, byte). The most significant bit is used as a continuation bit, which is a signal that there's at least one more byte worth of data to complete this integer. Let's decode one for practice:

```binary
11000000 11000100 00000111   // Original inputs.
 1000000  1000100  0000111   // Drop continuation bits.
 0000111  1000100  1000000   // Convert to big-endian.
 000011110001001000000       // Concatenate.
 (1 × 2^16) + (1 × 2^15) +
 (1 × 2^14) + (1 × 2¹³) +
 (1 × 2⁹) + (1 × 2⁶)
 = 123456                    // Interpret as an unsigned 64-bit integer.
```

Next, I will show you some go code that can do this encoding and decoding of the `VARINT`. Note that this code is actually in Go's [standard library](https://pkg.go.dev/encoding/binary) and makes reference to protocol-buffers directly ([source](https://github.com/golang/go/blob/go1.22.2/src/encoding/binary/varint.go#L39-L47)).

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
- The core logic is in a `for` loop that continues as long as x is greater than or equal to 128 (represented by `0x80` in hexadecimal). Why 128? That's the highest number (in hexadecimal) that you can count to only using 7 bits. Remember that the most significant bit (MSB) is used as a continuation bit.
- The next line appends the uint64 argument truncated to a byte, so the last 8 bits. One of those bits isn't actually used because of the `|0x80` part of the line. This combines our extracted 7 bits with `0x80` to ensure that the continuation bit is set to true.
- The next line uses the right shift operator to shift 7 bits off of our current uint64 value because we've "dealt with" these bits already. Next, we check the for-loop condition again until we get a number lower than `128`.
- Finally, we append the final byte as-is because we know that it's less than 128 which assures us that the MSB is NOT set.

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

Now that we've gone through that, I will tell you that **these functions aren't actually used by the protobuf library.** I suspect that they were initially used but the current [`protowire`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protowire) implementation doesn't use `for` loops at all; it has unwrapped for loops. Here's [AppendVarint](https://github.com/protocolbuffers/protobuf-go/blob/v1.33.0/encoding/protowire/wire.go#L184-L263) and [ConsumeVarint](https://github.com/protocolbuffers/protobuf-go/blob/v1.33.0/encoding/protowire/wire.go#L265-L367). That code looks insane but it is likely a bit faster than the version of the code that I just showed you.

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

TODO: fix this code
```go
tagNumber, protoType, err := ReadFieldTag(buf)
...
i, err := ReadUvarint(buf)
```
Note that I omitted error handling and in a real scenario we need to check the `tagNumber` and `protoType` to decide how to read this field.

## Strings/byte arrays
Next, we're going to start with strings and byte arrays. These use the `LEN` wire type. This type is a little more complex than `VARINT`. It's composed of a `VARINT` that tells us the size of the `LEN` field followed by the actual content in bytes. For `LEN` this content can be a string, a byte array, an embedded message or packed repeated fields. Here's what a field looks like:
- Field Tag Byte (field number along with the field type, set to `2` for the `LEN` type)
- Content size as a `VARINT`
- The actual content

TODO: fix this code
Encode:
```go
s := "hello world"
WriteFieldTag(buf, 4, 2)
WriteUvarint(buf, uint64(len(s)))
buf.WriteString(s)
```

Decode:
```go
buf := bytes.NewBuffer(b)
tagNumber, protoType, err := ReadFieldTag(buf)
...
s, err := ReadString(buf)
```

And to complete the context, you need the `ReadString` and `ReadBytes` functions here:
TODO: update this code
```go
func ReadBytes(buf *bytes.Buffer) ([]byte, error) {
	size, err := ReadUvarint(buf)
	...
	result := make([]byte, size)
	n, err := buf.Read(result)
	...
	return result, nil
}

func ReadString(buf *bytes.Buffer) (string, error) {
	b, err := ReadBytes(buf)
	if err != nil {
		return "", err
	}
	return string(b), err
}
```
TODO: update this text
`ReadBytes` has error checking omitted to make it clearer what it is doing. First, it uses `ReadUvarint` to read the size of the byte array or string. Then it makes a new byte slice that matches this size of the byte array/string/whatever. Then it reads from the `buf` object into our new byte slices. Note that we're allocating new memory here, which does vary from what protowire does. Protowire returns a slice of the buffer. This avoids an allocation at this level but I think it also may cause more memory usage. But I wanted to point out the balance here. Avoiding extra allocations here may be a performance benefit and may be a good reason to keep with `[]byte` over `bytes.Buffer`. I'm also concerned about the size here. It may be the case where `bytes.Buffer` or another kind of `io.Writer` is better for writing messages and directly using `[]byte` is better for reading messages.

## Integer Arrays (Packed)

Packed repeated fields are a space-saving optimization for integer arrays. Instead of encoding each element individually, a single `VARINT` is used at the beginning specifying the number of elements, followed by the raw integer data, one after another, with no separators. 

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
Notice that we are writing the list of `int32` values to a second buffer. I do this so that we can know the size of the encoded/packed `int32` values so that I can properly set the size for the encapsulating `LEN` wire type. This time reading is a bit easier. See how that works here:

TODO: fix this example
```go
func ReadRepeatedInt32(buf *bytes.Buffer) ([]int32, error) {
	result := []int32{}
	for {
		res, err := ReadUvarint(buf)
		if err == io.EOF {
			return result, nil
		} else if err != nil {
			return nil, err
		}
		result = append(result, int32(res))
	}
}
```

```go
buf := bytes.NewBuffer(b)
tagNumber, protoType, err := ReadFieldTag(buf)

packedBytes, err := ReadBytes(buf)
assert.NoError(t, err)

result, err := ReadRepeatedInt32(bytes.NewBuffer(packedBytes))
assert.NoError(t, err)
assert.Equal(t, []int32{1, 2, 3, 400}, result)
```

TODO: fix this text
Decoding a packed repeated field is similar. First, you read the number of elements using `ReadUvarint`. Then you would have a loop that reads that number of elements using the appropriate decoding function based on the field type (in this case, `ReadUvarint` again for `int32`).

## Conclusion

In this part of the series, we've taken a deep dive into the world of manual protobuf encoding. We explored a few wire types and the primatives that they are built on, like Base-128 varints, and built functions to encode and decode basic protobuf data types. While for most use cases, you'll probably want to leverage the efficiency and convenience of generated code and libraries like `golang/protobuf`, understanding how to manually encode these messages can be a valuable asset for debugging, creating a protobuf library implementation in your favorite new language or for possibly creating your own binary encoding.

In the next part of the series, we will put more pieces together with a layer on top of our encoding code and integrate this protobuf library into the client and server that we made in parts 1 and 2. build a real gRPC client and server that uses protobufs for data exchange!
