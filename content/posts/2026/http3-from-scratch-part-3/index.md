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

At the end of [HTTP/2 From Scratch: Part 4](/posts/http2-from-scratch-part-4/), I talked about why HTTP/2 only solved head-of-line blocking at the application layer. Since HTTP/2 multiplexes all of its streams over a single TCP connection, it still inherits TCP's strict sequential packet ordering. In Part 4, I drew this diagram of `Stream 4` waiting in the kernel's TCP receive buffer while TCP retransmits a lost packet for `Stream 0`:

{{< image src="holb.png" width="420px" class="center" >}}

In `quic_conn.go`, we use `net.ListenPacket("udp", addr)` instead of `net.Listen("tcp", addr)`. Because QUIC ([RFC 9000](https://www.rfc-editor.org/rfc/rfc9000.html)) frames its own packets over UDP and carries the TLS 1.3 handshake inside `CRYPTO` frames (`0x06`) (which I wrote about in [gRPC Over HTTP/3](/posts/grpc-over-http3/)), each `toyQUICStream` can buffer and read its own frames without waiting on other streams.

### Debugging QUIC Traffic

One downside of moving from HTTP/1.1 on TCP to QUIC on UDP is that you can't just use `nc` or `telnet` to see what your server is doing.

When I was working on my `quic-go` PRs, I used Wireshark with `SSLKEYLOGFILE=/tmp/keys.log`. With `/tmp/keys.log` configured under *Preferences -> Protocols -> TLS*, Wireshark decrypts the QUIC packets from `quic-go`, `curl`, and Chrome. I also used `QLOGDIR` in `quic-go` to dump `.qlog` files for [qvis.quictools.info](https://qvis.quictools.info/).

### QUIC Packets and STREAM Frames

In `quic_frame.go`, `ParsePacket` decodes the QUIC packet header ([RFC 9000 Section 17](https://www.rfc-editor.org/rfc/rfc9000.html#name-packet-formats)) from the start of each UDP datagram. We check the highest bit (`0x80`) of the first byte to see which of the two RFC 9000 packet headers we're parsing:

{{< image src="quic-packets.svg" width="820px" class="center" inline="true" >}}

Notice that both packet headers include a **Destination Connection ID** (`DestConnID`). In `quic_conn.go`, our listener stores active connections in `conns map[uint64]*toyQUICConn` keyed by `DestConnID` rather than by the client's UDP address (which is also how QUIC connections survive Wi-Fi to cellular IP changes).

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

Take a look at the `Stream ID` varint in that diagram too. Remember in Part 1 and Part 2 how client request streams were numbered `0, 4, 8...` while the control streams were `2` and `3`? The lowest two bits of a QUIC stream ID are bitflags for who initiated the stream (`client=0, server=1`) and whether it's unidirectional (`bidi=0, uni=1`). Counting up client bidirectional streams by 1 actually increments the ID by 4 (`0b00`, `0b100`, `0b1000`).

What about QUIC packet encryption? In a real QUIC stack like `quic-go`, the payload of every QUIC packet is encrypted with keys from the TLS 1.3 handshake in the `CRYPTO` frames. Just like I skipped implementing Huffman trees in HTTP/2 Part 4 and skipped HTTP trailers in Part 1, I'm not implementing RFC 9001 TLS packet header protection here. In `quic_conn.go`, we just write the encoded `Packet` bytes straight to the UDP socket.

### Reassembling Streams Over UDP

In `quic_conn.go`, each stream is a `toyQUICStream` that implements `io.Reader` and `io.Writer`. Whenever `handlePacket` decodes a `STREAM` frame from the UDP socket, it passes the frame's byte `Offset`, `Data`, and `Fin` bit to `pushFrame`, which stores the slice in `s.chunks[offset]` and wakes up any blocked reader on `s.cond`:

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

In `Read`, we look up `s.chunks[s.readOffset]` in a loop. If UDP datagrams arrive out of order (like `offset=6` arriving before `offset=0` in `TestOutOfOrderStreamReassembly`), `s.chunks[0]` isn't in the map yet and `Read` blocks on `s.cond.Wait()` without holding up other streams on the connection:

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

### Running HTTP/3 on Our UDP Transport

Removing `github.com/quic-go/quic-go` from `go.mod` leaves `golang.org/x/net` as our only dependency (which we added back in [HTTP/2 From Scratch: Part 4](/posts/http2-from-scratch-part-4/) for its HPACK/QPACK Huffman table):

```go
module http3-part-3

go 1.25.5

require golang.org/x/net v0.55.0
```

In `transport.go`, I pointed `ListenQUIC` and `DialQUIC` at `ListenToyQUIC` and `DialToyQUIC` instead of `quic-go`, while keeping `server.go` and `client.go` from Part 2 untouched:

```shell
\$ go test -v ./...
=== RUN   TestOutOfOrderStreamReassembly
--- PASS: TestOutOfOrderStreamReassembly (0.00s)
=== RUN   TestEndToEndHTTP3OverToyQUIC
--- PASS: TestEndToEndHTTP3OverToyQUIC (0.00s)
PASS
ok      http3-part-3    0.719s

\$ go run .
Homemade QUIC + HTTP/3 server listening on UDP 127.0.0.1:58114

[GET /hello] Status: 200 OK | X-Transport: toy-quic-udp | Body: HTTP/3 on homemade QUIC! Method=GET Path=/hello
[POST /echo] Status: 200 OK | Body: Echoing over our own UDP QUIC streams!
```

### Wrapping Up

That brings us to the end of the HTTP/3 posts and the "HTTP from Scratch" series ([HTTP/0.9](/posts/http0.9-from-scratch/), [HTTP/1.0](/posts/http1.0-from-scratch/), [HTTP/1.1](/posts/http1.1-from-scratch/), [HTTP/2](/posts/http2-from-scratch-part-1/), and now **HTTP/3**).

See all of the code mentioned in this article in [http3-from-scratch-part-3/go](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/http3-from-scratch-part-3/go).
