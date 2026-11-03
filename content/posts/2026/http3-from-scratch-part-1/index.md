---
categories: ["article"]
tags: ["golang", "http3", "quic", "protocols", "networking"]
date: "2026-11-12T10:00:00Z"
description: "Building HTTP/3 from scratch: HTTP/3 framing and QPACK over QUIC streams in Go."
cover: "cover.svg"
images: ["/posts/http3-from-scratch-part-1/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "HTTP/3 From Scratch: Part 1"
slug: "http3-from-scratch-part-1"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/http3-from-scratch-part-1/
series: ["HTTP from Scratch"]
---

In my previous posts in this series ([HTTP/0.9](/posts/http0.9-from-scratch/), [HTTP/1.0](/posts/http1.0-from-scratch/), [HTTP/1.1](/posts/http1.1-from-scratch/), and [HTTP/2](/posts/http2-from-scratch-part-1/)), I always started with a raw TCP socket. We would dial a port, wrap the connection in TLS, and then build the HTTP parser on top of it.

For HTTP/3, I'm doing the opposite and building from the sky, down.

HTTP/3 doesn't use TCP. It runs over QUIC, which is built on UDP and implements its own encryption (TLS 1.3) and streams. If we started with a raw UDP socket today, we would have to write a lot of QUIC packet and crypto code before we could send a single HTTP request. I didn't want to spend two or three posts on UDP plumbing before even touching HTTP/3.

Instead, we're going to start at the top using [`quic-go`](https://github.com/quic-go/quic-go) for the QUIC streams in Part 1 and Part 2 while we build our HTTP/3 framing, QPACK encoder/decoder, server, and client. Once our HTTP/3 implementation works against `curl` and browsers, we'll come back in Part 3, remove `quic-go`, and write a basic QUIC layer over UDP sockets.

All of the code for this post is in Go (using [`mise`](/posts/mise-en-place/) to manage the Go version) and is linked on GitHub at the bottom of the page.

### Sending Bytes Over QUIC

If you've written TCP servers in Go with the `net` package, `quic-go`'s API takes a minute to get used to. `net.Listener.Accept()` gives you a `net.Conn` that you can read from and write to directly. `quic.Listener.Accept()` gives you a `quic.Conn`, which doesn't have `Read` or `Write` methods at all. Because a single QUIC connection carries multiple streams, you have to call `AcceptStream` on the connection before you get something that looks like a `net.Conn`:

```go
conn, _ := listener.Accept(ctx)
stream, _ := conn.AcceptStream(ctx)

data, _ := io.ReadAll(stream)
stream.Write(append([]byte("echo: "), data...))
stream.Close()
```

Dialing from the client works the same way. As I wrote about in [gRPC Over HTTP/3](/posts/grpc-over-http3/), QUIC always uses TLS 1.3 and checks the ALPN string (`tls.Config.NextProtos`) during the handshake, so we pass a `tls.Config` into `quic.DialAddr`, open a stream, write our bytes, and read back the response:

```go
conn, _ := quic.DialAddr(ctx, addr, tlsConf, nil)
defer conn.CloseWithError(0, "done")

stream, _ := conn.OpenStreamSync(ctx)
stream.Write([]byte("Hello over QUIC!"))
stream.Close()

reply, _ := io.ReadAll(stream)
```

Notice that the client calls `stream.Close()` *before* calling `io.ReadAll(stream)`. In `quic-go`, `stream.Close()` only closes our write side of the stream (which sends a `FIN` bit on that stream), so we can still read the server's response afterward. Because QUIC streams already have a `FIN` bit, we won't need an `END_STREAM` flag on our HTTP/3 frames like we had in HTTP/2.

I also ran into the same bug here that I hit with `defer conn.Close()` in [HTTP/2 From Scratch: Part 4](/posts/http2-from-scratch-part-4/). In my first draft of `runEchoServer`, I put `defer conn.CloseWithError(0, "done")` at the top of the server function. When the server handler returned, `CloseWithError` immediately sent a `CONNECTION_CLOSE` packet and killed the connection before the client finished reading the response. Instead, the server should only call `stream.Close()` and wait on `<-conn.Context().Done()` until the client disconnects.

### Reusing HTTP/2 Framing

When you open [RFC 9114 (HTTP/3)](https://www.rfc-editor.org/rfc/rfc9114.html), you get a strong sense of déjà vu if you've read the HTTP/2 spec.

{{< image src="h3-unmasking.jpg" width="420px" class="center" >}}

In [HTTP/2 From Scratch: Part 2](/posts/http2-from-scratch-part-2/), we built a parser for HTTP/2's 9-byte frame header (`Length`, `Type`, `Flags`, and `StreamID`). Why did HTTP/2 have `StreamID` and `Flags` on every frame? Because all of those frames were being sent over a single TCP connection, so HTTP/2 had to handle its own stream IDs and `END_STREAM` / `END_HEADERS` flags. In HTTP/3, each request already has its own QUIC stream and QPACK header blocks aren't split across `CONTINUATION` frames. As a result, HTTP/3 drops `StreamID` and `Flags` from the frame header completely, leaving only `Type` and `Length`:

{{< image src="h2-vs-h3-frame.svg" width="800px" class="center" inline="true" >}}

The frame `Type` identifiers also use the same values we had in HTTP/2:

```go
const (
	FrameData     uint64 = 0x00
	FrameHeaders  uint64 = 0x01
	FrameSettings uint64 = 0x04
	FrameGoAway   uint64 = 0x07
)

type Frame struct {
	Type    uint64
	Payload []byte
}
```

Frame types `0x02`, `0x03`, `0x05`, and `0x06` from HTTP/2 (`PRIORITY`, `RST_STREAM`, `PUSH_PROMISE`, and `PING`) are gone here. Stream resets and pings moved into QUIC itself, and priority frames and Server Push were removed.

The main difference in the wire format is that `Type` and `Length` use QUIC variable-length integers ([RFC 9000 Section 16](https://www.rfc-editor.org/rfc/rfc9000.html#name-variable-length-integer-enc)) instead of fixed-width integers. If you've used the varint tool on [protobuf.com](https://protobuf.com), you might remember that Protobuf varints use the most significant bit of each byte as a continuation flag. Instead of checking a continuation bit on every byte, a QUIC varint uses the top two bits of the first byte (`00`, `01`, `10`, or `11`) to encode whether the number is 1, 2, 4, or 8 bytes long.

{{< image src="quic-varint.svg" width="800px" class="center" inline="true" >}}

In `ReadVarint`, we grab that first byte, shift the top two bits down with `first[0] >> 6`, and use `1 << prefix` to figure out how many bytes to read:

```go
func ReadVarint(r io.Reader) (uint64, error) {
	var first [1]byte
	if _, err := io.ReadFull(r, first[:]); err != nil {
		return 0, err
	}

	prefix := first[0] >> 6
	length := 1 << prefix

	val := uint64(first[0] & 0x3f)
	if length == 1 {
		return val, nil
	}

	rest := make([]byte, length-1)
	_, err := io.ReadFull(r, rest)
	for _, b := range rest {
		val = (val << 8) | uint64(b)
	}
	return val, err
}
```

Once we can read and write varints, `ReadFrame` is just two varint reads followed by `io.ReadFull` for the payload (in `frame.go` I also cap `length` at 16 MB so a bogus 62-bit varint doesn't try to allocate terabytes of memory):

```go
func ReadFrame(r io.Reader) (Frame, error) {
	fType, err := ReadVarint(r)
	if err != nil {
		return Frame{}, err
	}
	length, err := ReadVarint(r)
	if err != nil {
		return Frame{}, err
	}
	payload := make([]byte, length)
	_, err = io.ReadFull(r, payload)
	return Frame{Type: fType, Payload: payload}, err
}
```

### From HPACK to QPACK

In [HTTP/2 From Scratch: Part 3](/posts/http2-from-scratch-part-3/) and [Part 4](/posts/http2-from-scratch-part-4/), we wrote an `HPACKDecoder` and `HPACKEncoder` ([RFC 7541](https://www.rfc-editor.org/rfc/rfc7541.html)). Why couldn't the HTTP/3 authors just reuse HPACK as-is and let us all go home early?

The issue is HPACK's dynamic table. In HTTP/2, every `HEADERS` frame that inserts a new header into the dynamic table is sent over the same underlying TCP connection. Because TCP delivers all bytes strictly in order, the decoder is guaranteed to see table insertions in the exact order the encoder sent them, even across different streams.

With QUIC, packets on different streams can arrive out of order. If a packet on Stream 0 gets dropped on the network while a packet on Stream 4 arrives, QUIC can deliver Stream 4 right away. But if Stream 4's headers referenced a dynamic table entry that was supposed to be created by Stream 0's lost packet, the decoder wouldn't be able to read Stream 4 until Stream 0 was retransmitted. We'd be right back to head-of-line blocking.

To get around this, HTTP/3 uses QPACK ([RFC 9204](https://www.rfc-editor.org/rfc/rfc9204.html)). In QPACK, any dynamic table updates happen on separate unidirectional streams instead of inside the request streams, and every `HEADERS` frame starts with a 2-byte prefix (`Required Insert Count` and `Delta Base`). The static table is also bigger than HPACK's (99 entries starting at index 0, compared to 61 entries starting at index 1 in HPACK) and includes common headers like `content-type: application/json` (46) and `:status: 200` (25).

{{< image src="qpack-wire.svg" width="800px" class="center" inline="true" >}}

If we don't use the dynamic table and only encode headers using the 99-entry static table and string literals, we don't have to open the encoder/decoder streams at all, and the 2-byte prefix at the beginning of every `HEADERS` frame is just `0x00, 0x00`. [RFC 9204 Section 4.1](https://www.rfc-editor.org/rfc/rfc9204.html#section-4.1) also keeps the same integer encoding (`decodeInt`, `encodeInt`) and Huffman table (`golang.org/x/net/http2/hpack`) that HPACK used. We can reuse those functions from our HTTP/2 package and just change the bitmasks on the first byte of each field line:

```go
func (d *QPACKDecoder) Decode(payload []byte) ([]HeaderField, error) {
	r := bytes.NewReader(payload)
	_, _ = decodeInt(0, r, 8)
	_, _ = decodeInt(0, r, 7)

	var headers []HeaderField
	for r.Len() > 0 {
		b, _ := r.ReadByte()
		switch {
		case b&0xc0 == 0xc0:
			index, _ := decodeInt(b, r, 6)
			headers = append(headers, QPACKStaticTable[index])

		case b&0xd0 == 0x50:
			index, _ := decodeInt(b, r, 4)
			val, _ := decodeString(r)
			headers = append(headers, HeaderField{Name: QPACKStaticTable[index].Name, Value: val})

		case b&0xe0 == 0x20:
			name, _ := decodeNameString(b, r)
			val, _ := decodeString(r)
			headers = append(headers, HeaderField{Name: name, Value: val})
		}
	}
	return headers, nil
}
```

### Modeling Requests and Responses

Now that we can encode QPACK headers and read/write HTTP/3 frames, sending an HTTP request or response over a QUIC stream looks a lot like HTTP/2. We still have the same lowercase header names and pseudo-headers (`:method`, `:scheme`, `:authority`, `:path` on requests, and `:status` on responses). In `WriteRequest`, we QPACK-encode the headers into a `HEADERS` frame (`0x01`) and then write the request body into a `DATA` frame (`0x00`):

```go
func WriteRequest(w io.Writer, req *http.Request) error {
	headers := []HeaderField{
		{Name: ":method", Value: req.Method},
		{Name: ":scheme", Value: req.URL.Scheme},
		{Name: ":authority", Value: req.URL.Host},
		{Name: ":path", Value: req.URL.RequestURI()},
	}

	enc := NewQPACKEncoder()
	WriteFrame(w, FrameHeaders, enc.Encode(headers))

	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		if len(bodyBytes) > 0 {
			WriteFrame(w, FrameData, bodyBytes)
		}
	}
	return nil
}
```

To read the message on the other side, we call `ReadFrame` in a loop until `io.EOF`:

```go
for {
	frame, err := ReadFrame(r)
	if errors.Is(err, io.EOF) {
		break
	}
	switch frame.Type {
	case FrameHeaders:
		fields, _ := dec.Decode(frame.Payload)
		applyHeaders(req, fields)
	case FrameData:
		bodyBuf.Write(frame.Payload)
	}
}
```

### What About HTTP Trailers?

In HTTP/3, how do you send HTTP trailers (like `grpc-status` at the end of a gRPC call)? Because there are no flags on HTTP/3 frames, trailers are simply sent as a **second `HEADERS` frame after the `DATA` frames**, right before closing the stream.

Are we going to implement HTTP trailers in our from-scratch server and client?

{{< image src="trailers.png" width="420px" class="center" >}}

**Hard pass. We're not implementing those.**

If you really want to see how trailers work in HTTP/3, you can reference my PRs to `quic-go` ([#4581](https://github.com/quic-go/quic-go/pull/4581), [#4630](https://github.com/quic-go/quic-go/pull/4630), [#4639](https://github.com/quic-go/quic-go/pull/4639), [#4656](https://github.com/quic-go/quic-go/pull/4656)) where I added trailer support to the `quic-go` HTTP/3 implementation, or read my [gRPC Over HTTP/3](/posts/grpc-over-http3/) and [gRPC Over HTTP/3: Followup](/posts/grpc-over-http3-followup/) articles. Here, if a second `HEADERS` frame shows up after `DATA`, we'll simply ignore it.

### Running the Demo

In `main.go`, I wired up the raw QUIC echo test followed by an HTTP/3 request and response over a QUIC stream:

```shell
$ go run .
=== Step 1: Raw QUIC Echo Demo ===
[Echo Client] Sending on stream 0: "Hello over QUIC!"
[Echo Server] Received on stream 0: "Hello over QUIC!"
[Echo Client] Got reply: "echo: Hello over QUIC!"

=== Step 2: HTTP/3 Framing & QPACK Over a QUIC Stream ===
[H3 Server] Decoded POST /hello (Host: kmcd.dev, User-Agent: kmcd-h3/0.1, Body: "ping from h3")
[H3 Client] Status: 200 OK
[H3 Client] Content-Type: text/plain; charset=utf-8, Server: kmcd-h3-scratch
[H3 Client] Body: Hello over HTTP/3! You sent 12 bytes.
```

That's our first real `*http.Request` encoded into QPACK headers and HTTP/3 `HEADERS` + `DATA` frames, sent across a QUIC stream, and answered with a QPACK-encoded `*http.Response`.

### What's Next?

If you pointed `curl --http3` or Chrome at our listener right now, they would hang up on us immediately. Real HTTP/3 peers expect both sides to open a one-way **Control Stream** (`0x00`) as soon as the connection starts and swap `SETTINGS` frames (`0x04`). Plus, browsers need a way to find out that your server speaks HTTP/3 over UDP in the first place.

In **Part 2**, we'll look at how browsers discover HTTP/3 (`Alt-Svc` and DNS `HTTPS` records), how to convince Chrome and `curl` to talk HTTP/3 to `localhost`, and wrap our framing code into a proper `http.Handler` server and `Client`.

See all of the code mentioned in this article in [http3-from-scratch-part-1/go](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/http3-from-scratch-part-1/go).
