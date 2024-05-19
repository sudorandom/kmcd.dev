---
categories: ["article", "tutorial"]
tags: ["networking", "grpc", "go", "golang", "tutorial", "protobuf"]
series: ["gRPC from Scratch"]
date: "2024-07-21"
description: "We have more work to do with protobuf encoding!"
cover: "cover.jpg"
images: ["/posts/grpc-from-scratch-part-4/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC From Scratch: Part 4 - More Protobuf Encoding"
slug: "grpc-from-scratch-part-4"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-from-scratch-part-4
draft: true
---
TODO: intro

I hear that `bytes.Buffer` is often-times faster than using `[]bytes` append(), so I'm actually going to rewrite these functions to use `bytes.Buffer`. This will also prevent us from needing to manually manage the byte offset. My hunch is that this method is slower but I still like it because it results in more readable code. Here are the equivalent functions now:

**Writing a VARINT**
```go
func WriteUvarint(buf *bytes.Buffer, x uint64) {
	for x >= 0x80 {
		buf.WriteByte(byte(x) | 0x80)
		x >>= 7
	}
	buf.WriteByte(byte(x))
}
```
This function is pretty much the same, except I am using buf.WriteBytes instead of append() and I don't need to return anything now since the offset and the actual byte array are managed by the `bytes.Buffer`.

**Reading a VARINT**
```go
func ReadUvarint(buf *bytes.Buffer) (uint64, error) {
	var x uint64
	var s uint
	var i int
	for {
		b, err := buf.ReadByte()
		if err != nil {
			return 0, err
		}
		if i == MaxVarintLen64 {
			return 0, ErrOverflow // overflow
		}
		if b < 0x80 {
			if i == MaxVarintLen64-1 && b > 1 {
				return 0, ErrOverflow // overflow
			}
			return x | uint64(b)<<s, nil
		}
		x |= uint64(b&0x7f) << s
		s += 7
		i++
	}
}
```
The core of this function is also the same, but how the loop operates is slightly different and I return an error instead of a byte count (with sentinal values for errors).

## Integers
Now that we know how the VARINT wire-type works we now have enough raw material to write some protobuf packets. Again, we need three things to make a message with a single field: Field number, wire type, and the encoded value. As mentioned earlier, the field number and wire type are merged into a single VARINT value using the formula: `(field_number << 3) | wire_type`. This essentially means that the the least significant bits are reserved for the wire type and the rest are used for the field number. Okay, what does that look like to write a full field in protobuf? Using the functions I made above, I made a `WriteField` function to help with writing the field tag.

```go
buf := &bytes.Buffer{}
WriteUvarint(buf, uint64(1<<3)|uint64(0))
WriteUvarint(buf, 1234)
```

And... that's it. We've fully encoded probably the simpliest (non-empty) protobuf message. We can check that it works by running a test with the "real" protobuf library:

```go
func TestEncodeRaw(t *testing.T) {
	buf := &bytes.Buffer{}
	WriteUvarint(buf, uint64(1<<3)|uint64(0))
	WriteUvarint(buf, 1234)

	res := gen.TestMessage{}
	require.NoError(t, proto.Unmarshal(buf.Bytes(), &res))
	assert.Equal(t, int32(1234), res.IntValue)
}
```

By the way, these tests are using a protobuf file that looks like this:
{{% render-code file="go/types.proto" language="protobuf" %}}
All of the field numbers and corresponding types in this article match up to this protobuf file.

Okay, let's show what it looks like to read this message:

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
`ReadBytes` has error checking omitted to make it clearer what it is doing. First, it uses `ReadUvarint` to read the size of the byte array or string. Then it makes a new byte slice that matches this size of the byte array/string/whatever. Then it reads from the `buf` object into our new byte slices. Note that we're allocating new memory here, which does vary from what protowire does. Protowire returns a slice of the buffer. This avoids an allocation at this level but I think it also may cause more memory usage. But I wanted to point out the balance here. Avoiding extra allocations here may be a performance benefit and may be a good reason to keep with `[]byte` over `bytes.Buffer`. I'm also concerned about the size here. It may be the case where `bytes.Buffer` or another kind of `io.Writer` is better for writing messages and directly using `[]byte` is better for reading messages.

## Integer Arrays (Packed)

Packed repeated fields are a space-saving optimization for integer arrays. Instead of encoding each element individually, a single `VARINT` is used at the beginning specifying the number of elements, followed by the raw integer data, one after another, with no separators. 

Here's an example of how to encode a packed repeated integer field:
```go
func SizeVarint(v uint64) int {
	return int(9*uint32(bits.Len64(v))+64) / 64
}
```

```go
arr := []int32{100002130, 2, 3, 4, 5}
buf := &bytes.Buffer{}
WriteFieldTag(buf, 10, 2)
size := 0
for i := 0; i < len(arr); i++ {
	size += SizeVarint(uint64(arr[i]))
}
WriteUvarint(buf, uint64(size))
for i := 0; i < len(arr); i++ {
	WriteUvarint(buf, uint64(arr[i]))
}
```

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

Decoding a packed repeated field is similar. First, you read the number of elements using `ReadUvarint`. Then you would have a loop that reads that number of elements using the appropriate decoding function based on the field type (in this case, `ReadUvarint` again for `int32`).

## Embedded Messages

Embedded messages are another protobuf data type. They allow you to nest messages within other messages. When encoding an embedded message, you treat it like any other field. You write the field tag, and then you write the entire encoded message data using the same process you would use to encode the message itself. Decoding an embedded message involves reading the field tag and then treating the following bytes as the encoded message data, which you can then unmarshal using the appropriate message type.

## Conclusion

In this part of the series, we've taken a deep dive into the world of manual protobuf encoding. We explored wire types, tackled Base-128 Varints, and built functions to encode and decode basic protobuf data types. While for most use cases, you'll probably want to leverage the efficiency and convenience of generated code and libraries like `protowire`, understanding manual encoding can be a valuable asset for debugging or interfacing with protobuf data from other systems. 

In the next part of the series, we will put more pieces together with a layer on top of our encoding code and integrate this protobuf library into the client and server that we made in parts 1 and 2. build a real gRPC client and server that uses protobufs for data exchange!
