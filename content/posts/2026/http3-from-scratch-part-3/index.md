---
categories: ["article"]
tags: ["golang", "http3", "quic", "udp", "protocols", "networking"]
date: "2026-11-24T10:00:00Z"
description: "Digging all the way down to UDP: implementing QUIC packets, stream multiplexing, and plugging in our HTTP/3 server and client."
cover: "cover.svg"
images: ["/posts/http3-from-scratch-part-3/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "HTTP/3 From Scratch: Part 3"
slug: "http3-from-scratch-part-3"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/http3-from-scratch-part-3/
series: ["HTTP from Scratch"]
draft: true
---

In [Part 1](/posts/http3-from-scratch-part-1/) and [Part 2](/posts/http3-from-scratch-part-2/), we wrote our HTTP/3 framing, QPACK encoder and decoder, control streams, and `Server` and `Client` types in Go. At the end of Part 2, our HTTP/3 client and server could talk to `curl`, `quic-go`, and Chrome, but we were still using `quic-go` to handle the UDP sockets and QUIC streams.

In this post, we'll remove `quic-go` and implement our own QUIC packet and stream layer over UDP.

### Leaving TCP Behind

At the end of [HTTP/2 From Scratch: Part 4](/posts/http2-from-scratch-part-4/), I talked about why HTTP/2 only solved head-of-line blocking at the application layer. Since HTTP/2 multiplexes all of its streams over a single TCP connection, it still inherits TCP's strict sequential packet ordering. If a TCP packet carrying bytes for `Stream 0` gets dropped on the network while a packet for `Stream 4` arrives, the OS kernel holds `Stream 4`'s bytes in its buffer until the lost packet for `Stream 0` is retransmitted:

{{< image src="holb.png" width="420px" class="center" >}}

As I mentioned back then, the only ways around this were to make big changes to TCP (which isn't going to happen with how many routers and firewalls inspect TCP headers) or abandon TCP for UDP. QUIC ([RFC 9000](https://www.rfc-editor.org/rfc/rfc9000.html)) takes the UDP route. Since UDP datagrams are just 8 bytes of port and length headers followed by raw bytes, QUIC can manage its own streams and retransmissions on top of `net.PacketConn` without one lost packet stalling every other stream on the connection.

And as I wrote about in [gRPC Over HTTP/3](/posts/grpc-over-http3/), QUIC also folds the TLS 1.3 handshake into its own `CRYPTO` frames (`0x06`) instead of waiting for a separate TCP 3-way handshake to finish first, which gets new connections down to 1-RTT (or 0-RTT when resuming a session).

### Debugging QUIC Traffic

One downside of moving from HTTP/1.1 on TCP to QUIC on UDP is that you can't just use `nc` or `telnet` to see what your server is doing.

When I was working on my `quic-go` PRs, I used Wireshark with `SSLKEYLOGFILE=/tmp/keys.log`. If you set that environment variable when running `quic-go`, `curl`, or Chrome, they dump their TLS keys to `/tmp/keys.log`. You can configure that path in Wireshark under *Preferences -> Protocols -> TLS* and Wireshark will decrypt the UDP datagrams and show you the `CRYPTO`, `ACK`, and `STREAM` frames inside. `quic-go` also has a `QLOGDIR` environment variable that writes JSON trace files you can open in [qvis.quictools.info](https://qvis.quictools.info/).

### QUIC Packets and STREAM Frames

When we read a UDP datagram from `net.PacketConn`, we first need to parse the QUIC packet header ([RFC 9000 Section 17](https://www.rfc-editor.org/rfc/rfc9000.html#name-packet-formats)). The highest bit of the first byte tells us whether we're looking at a **Long Header** (used during the handshake) or a **Short Header** (used for 1-RTT data):

{{< image src="quic-packets.svg" width="820px" class="center" inline="true" >}}

Notice that both packet headers include a **Destination Connection ID** (`DestConnID`). With TCP, a connection is identified by the client and server IP and port, so if you switch from Wi-Fi to mobile data and your IP address changes, your TCP connection dies. In QUIC, the server looks up the connection in a map using `DestConnID` instead of the UDP source address.

Inside the packet payload is a sequence of QUIC frames ([RFC 9000 Section 19](https://www.rfc-editor.org/rfc/rfc9000.html#name-frame-types-and-formats)). Here are the `Packet` and `QUICFrame` structs we use in `quic_frame.go`:

```go
type Packet struct {
	IsLongHeader bool
	LongType     uint8
	Version      uint32
	DestConnID   uint64
	SrcConnID    uint64
	PacketNumber uint32
	Frames       []QUICFrame
}

type QUICFrame struct {
	Type     uint64
	StreamID uint64
	Offset   uint64
	Fin      bool
	Data     []byte
}
```

The frame type we care about most for HTTP/3 is the **`STREAM` frame (`0x08` to `0x0f`)**. In a `STREAM` frame, the lowest 3 bits of the frame type byte tell the parser whether the `Offset` field (`0x04`), `Length` field (`0x02`), and `FIN` bit (`0x01`) are present:

{{< image src="quic-stream-frame.svg" width="820px" class="center" inline="true" >}}

In `AppendQUICFrame`, we set those bits on `QUICFrameStreamBase` (`0x08`) and encode the fields with the `AppendVarint` function we wrote in Part 1:

```go
frameType := QUICFrameStreamBase | StreamFlagLEN
if f.Offset > 0 {
	frameType |= StreamFlagOFF
}
if f.Fin {
	frameType |= StreamFlagFIN
}
buf = AppendVarint(buf, frameType)
buf = AppendVarint(buf, f.StreamID)
if f.Offset > 0 {
	buf = AppendVarint(buf, f.Offset)
}
buf = AppendVarint(buf, uint64(len(f.Data)))
buf = append(buf, f.Data...)
```

The `Stream ID` varint in that diagram also uses its lowest two bits as flags: bit `0` is `0` for client-initiated streams and `1` for server-initiated streams, and bit `1` is `0` for bidirectional streams and `1` for unidirectional streams. That explains the stream numbers we saw in Part 1 and Part 2: client request streams have `0b00` in the bottom two bits (`0, 4, 8...`), the client's control stream has `0b10` (`2`), and the server's control stream has `0b11` (`3`).

What about QUIC packet encryption? In a real QUIC stack like `quic-go`, the payload of every QUIC packet is encrypted with keys from the TLS 1.3 handshake in the `CRYPTO` frames. Just like I skipped implementing Huffman trees in HTTP/2 Part 4 and skipped HTTP trailers in Part 1, I'm not implementing RFC 9001 TLS packet header protection here. Our toy `quic_conn.go` sends the encoded `Packet` bytes directly over the UDP socket.

### Reassembling Streams Over UDP

How do we turn unordered UDP datagrams into an `io.Reader` stream that delivers bytes in order? In `quic_conn.go`, each `toyQUICStream` keeps a `chunks map[uint64][]byte` that maps a byte `Offset` to its data slice, along with `readOffset` and `finOffset`:

```go
func (s *toyQUICStream) pushFrame(offset uint64, data []byte, fin bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(data) > 0 {
		s.chunks[offset] = bytes.Clone(data)
	}
	if fin {
		s.finReceived = true
		s.finOffset = offset + uint64(len(data))
	}
	s.cond.Broadcast()
}
```

When the HTTP/3 server or client calls `stream.Read(p)`, we check if `s.chunks` has an entry for the current `s.readOffset`. If the next chunk hasn't arrived yet, `Read` blocks on `s.cond.Wait()`:

```go
func (s *toyQUICStream) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		if chunk, ok := s.chunks[s.readOffset]; ok {
			n := copy(p, chunk)
			s.advanceReadOffset(n, chunk)
			return n, nil
		}
		if s.finReceived && s.readOffset >= s.finOffset {
			return 0, io.EOF
		}
		s.cond.Wait()
	}
}
```

If `offset=6 ("world!")` arrives on the UDP socket before `offset=0 ("hello ")`, `pushFrame` stores `"world!"` at `s.chunks[6]` and `Read` waits until `offset=0` shows up. Once `offset=0` arrives, `Read` reads `"hello "`, advances `s.readOffset` to `6`, and can immediately read `"world!"` on the next call. And since every `toyQUICStream` has its own `chunks` map, a delayed packet on `Stream 0` won't block `Read` on `Stream 4`.

### Running HTTP/3 on Our UDP Transport

With `quic_frame.go` and `quic_conn.go` in place, we can drop `github.com/quic-go/quic-go` from `go.mod`:

```go
module http3-part-3

go 1.25.5

require golang.org/x/net v0.55.0
```

The only dependency left in `go.mod` is `golang.org/x/net` for the HPACK/QPACK Huffman table from [HTTP/2 From Scratch: Part 4](/posts/http2-from-scratch-part-4/).

In `transport.go`, we wire `ListenQUIC` and `DialQUIC` up to `ListenToyQUIC` and `DialToyQUIC` and run the tests and demo with our `server.go` and `client.go` from Part 2:

```shell
$ go test -v ./...
=== RUN   TestOutOfOrderStreamReassembly
--- PASS: TestOutOfOrderStreamReassembly (0.00s)
=== RUN   TestEndToEndHTTP3OverToyQUIC
--- PASS: TestEndToEndHTTP3OverToyQUIC (0.00s)
PASS
ok  	http3-part-3	0.719s

$ go run .
Homemade QUIC + HTTP/3 server listening on UDP 127.0.0.1:58114

[GET /hello] Status: 200 OK | X-Transport: toy-quic-udp | Body: HTTP/3 on homemade QUIC! Method=GET Path=/hello
[POST /echo] Status: 200 OK | Body: Echoing over our own UDP QUIC streams!
```

### Wrapping Up

That brings us to the end of the HTTP/3 posts and the "HTTP from Scratch" series ([HTTP/0.9](/posts/http0.9-from-scratch/), [HTTP/1.0](/posts/http1.0-from-scratch/), [HTTP/1.1](/posts/http1.1-from-scratch/), [HTTP/2](/posts/http2-from-scratch-part-1/), and now **HTTP/3**).

Starting at the top with `quic-go` in Part 1 and Part 2 before writing the UDP packet code here in Part 3 worked out much better than I expected when I started planning these posts. By the time we got to `STREAM` frames and stream ID bitmasks in this post, we had already seen how HTTP/3 uses them from the application side.

See all of the code mentioned in this article in [http3-from-scratch-part-3/go](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/http3-from-scratch-part-3/go).
