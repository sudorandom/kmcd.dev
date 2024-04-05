---
categories: ["article"]
tags: ["networking", "grpc", "http", "go", "golang", "tutorial", "protobuf", "connectrpc"]
date: "2024-02-15"
description: "gRPC is an incredibly popular RPC framework that efficiently connects services. But how does it work? Let's dive in!"
cover: "cover.jpg"
images: ["/posts/grpc-from-scratch/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC From Scratch: Part 1 - Client"
slug: "grpc-from-scratch"
type: "posts"
devtoSkip: true
canonical_url: https://sudorandom/dev/posts/grpc-from-scratch
---

> Disclaimer: This article is *NOT* for beginners unfamiliar with gRPC. If you're looking to use gRPC like a sane individual, look elsewhere. Maybe start with [the official gRPC documentation](https://grpc.io/docs/).

In the realm of distributed systems and microservices, [gRPC](https://grpc.io/) has become the go-to communication protocol, boasting speed and efficiency over other web technologies like JSON-based HTTP APIs. If you've ever wondered what's happening under the hood of gRPC, you're in luck. Today, we're embarking on an exciting journey into the intricate world of gRPC, delving deep into the byte-level details that make it tick. While I do this, I hope to convince you that gRPC is a *much simpler* protocol than you probably think.

Today I'm focusing on the basics of how the gRPC protocol works from a protocol level. But first, let me tell you what this article is not going to cover: **[protobufs](https://protobuf.dev/)**. That is a separate topic that is covered extremely well [in the official documentation](https://protobuf.dev/programming-guides/encoding/). I may cover this topic later on in this series but for now, protobufs are being treated as a black box.

The first thing to know is that gRPC is built on top of HTTP. Let's outline how protobuf service definitions map to HTTP/2 semantics with gRPC... First, let's take this hello world example.

```protobuf
package helloworld;

// The greeting service definition.
service Greeter {
  // Sends a greeting
  rpc SayHello (HelloRequest) returns (HelloReply) {}
}
```

Take these simple observations:
- The package is named **helloworld**
- The service is named **Greeter**
- The method is named **SayHello**

The corresponding HTTP request starts like this:

```http
POST /helloworld.Greeter/SayHello HTTP/2
Content-Type: application/grpc+proto
```

That covers how to start a request. But what does the body look like? Take a look at this table. Sorry, yes: This is binary data. How do you expect gRPC to get its performance improvements if it didn't use a binary encoding??:

| Byte Offset | Content | Decoded | Description |
| ----------- | ------- | ------- | ----------- |
| 0 | 00000000 | off | Compressed-Flag |
| 1:4 | 00000000 00000000 00000000 00000111 | 7 | Message-Length (Unsigned 32-bit integer; [Big Endian ordering](https://en.wikipedia.org/wiki/Endianness)) |
| 5:12 | 00001010 00000101 01010111 01101111 01110010 01101100 01100100 | 1:"World" | Message content |

There are **5 bytes** in total as a prefix to each encoded protobuf message. The first byte is for a flag saying if the message is compressed or not. Even though headers might say that compression is supported, servers can make their own decisions on whether or not to compress each message. Some messages are just too small to make compression worth it.

The last 4 bytes of the prefix are an unsigned 32-bit integer ([using big-endian byte ordering](https://en.wikipedia.org/wiki/Endianness)) which tells the client/server how many bytes the next message will take.

gRPC always returns an HTTP 200 status code. That's weird, right? gRPC does this because, for streaming RPCs, it's impossible to know if a request succeeded ahead of time. Therefore, gRPC always returns a 200 as the status code and waits until the very end of the request to report the `gRPC status` using a lesser-known feature of HTTP called an HTTP trailer. Trailers are exactly like headers but come at the end of a request instead of the beginning.

> Did you know? gRPC doesn't *actually* require HTTP/2 support. Most HTTP/1.1 servers and proxies lack support for HTTP Trailers even though trailers were [in the HTTP spec since 1.1](https://www.rfc-editor.org/rfc/rfc7230.html#section-4.4). You can read more about the full story [in this blog post](https://carlmastrangelo.com/blog/why-does-grpc-insist-on-trailers).

## Okay, let's code something
For the second half of this article, we're going to build a very un-featureful gRPC client in Go. It won't support many features that are expected out of gRPC but it will be able to make RPC calls.

### Making a service in protobuf
First, here's the full protobuf file that I'm going to use for this example. It's as close to the simplest Hello World in protobufs as you can get.

{{% render-code file="go/greet.proto" language="protobuf" %}}

I used a [buf.gen.yanl file](https://github.com/sudorandom/sudorandom.dev/tree/main/content/posts/2024-02-15_grpc-from-scratch/go/buf.gen.yaml) along with the `buf generate` command to build this protobuf into Go types.

### Making a simple gRPC server
We're not writing the gRPC server from scratch in this example, just a client (but the principles are the same if you want to do this as an exercise on your own). Additionally, we need a real gRPC server to test our client against so I will use the server handler that [ConnectRPC](https://connectrpc.com/) provides for us. Here's what that looks like:

{{% render-code file="go/server/main.go" language="go" %}}

### Encoding the Request
Now that the setup is out of the way, let's build a client that can send a gRPC request and receive a response from the HTTP server. Note that this will only work for a unary RPC call (a call that does not support streaming). First, let's start with encoding/decoding messages using the format I mentioned above.

Here's the code to write a request to a gRPC server.
```go
func writeMessage(w io.Writer, msg []byte) {
	prefix := make([]byte, 5)
	binary.BigEndian.PutUint32(prefix[1:], uint32(len(msg)))
	w.Write(prefix)
    w.Write(msg)
}
```

### Decoding the Response
The response is returned using the exact same format as our request. With go, this is what it might look like (without any error handling at all)
```go
func readMessage(body io.Reader) []byte {
	// read the prefix/envelope
	prefixes := [5]byte{}
	io.ReadFull(body, prefixes[:])

	// Using the message size from the prefix, read that many bytes. That's our protobuf message.
	buffer := &bytes.Buffer{}
	msgSize := int64(binary.BigEndian.Uint32(prefixes[1:5]))
	io.CopyN(buffer, body, msgSize)
	return buffer.Bytes()
}
```

This code reads the prefixes (the compression flag and the message size) and then the message size is used when reading the message from the server. That's... essentially it.

## The rest
We have the foundation of the gRPC protocol completed. Now, the missing part is code that creates the actual HTTP request, encoding/decoding the actual protobuf types (which is a simple call to `proto.Marshal` and `proto.Unmarshal`) and doing some error handling that I didn't do above. To me, none of that is particularly interesting but you can explore [the entire working prototype here](https://github.com/sudorandom/sudorandom.dev/tree/main/content/posts/2024-02-15_grpc-from-scratch/go) on your own time.

Here's what the output looks like:

```text
send-> name:"World"
recv<- greeting:"Hello, World!"
```

## What about gRPC streams?
Streaming requests simply repeat this envelope encoding. gRPC, for better or worse, made the unary (simple request and response) use case a bit harder for the benefit of making the more complex use cases (streams) simple. Plus, you only have to write one encoder/decoder. For a server streaming RPC, you would simply repeat the `readMessage()` function call until you get an EOF or some other error. This showcases gRPC's simplicity in handling streaming.

Note that you will have to periodically call `Flush()` on the handler's `http.ResponseWriter` argument to get the bytes flushed to the TCP socket. This normally happens for you automatically after the `http.Handler` is complete... but with streaming calls we have to do this ourselves! Here's an example of how to do it:

```golang
if f, ok := w.(http.Flusher); ok {
	f.Flush()
}
```

> By the way, [ConnectRPC](https://connectrpc.com/docs/protocol) has chosen to "undo" this approach by offering unary RPC without the 5-byte custom envelope. This makes using tools like cURL possible and even pleasant, especially when using the JSON encoding. But it is also friendly with gRPC clients [by offering gRPC and gRPC-Web alongside the Connect protocol](https://connectrpc.com/docs/multi-protocol).

## Okay, what was the point?
Hopefully, I was able to shed a little bit of light on how gRPC *really* works. Binary protocols often have hard-to-understand documentation about each byte in a packet. However, gRPC only has 5 bytes of this weirdness so it's a perfect protocol to whet your appetite on network protocols.

If you want to look at more details about the gRPC specification, I would refer you to [the official gRPC specification on GitHub](https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md). [See the full prototype from this post here.](https://github.com/sudorandom/sudorandom.dev/tree/main/content/posts/2024-02-15_grpc-from-scratch/go)

[<< Continue to see gRPC From Scratch: Part 2 where I build a simple gRPC server. >>](/posts/grpc-from-scratch-part-2/)
