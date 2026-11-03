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

In [Part 1](/posts/http3-from-scratch-part-1/) and [Part 2](/posts/http3-from-scratch-part-2/), we built an HTTP/3 server and client in Go. We wrote the variable-length integer framing, QPACK header compression, unidirectional control streams, and `http.Handler` / `http.Client` adapters on top of a `Conn` and `Stream` interface backed by `quic-go`.

In this final part, we'll remove `quic-go` from `go.mod`, look at how QUIC works on the wire, write a small QUIC packet and `STREAM` framing layer over UDP sockets (`net.PacketConn`), and run our HTTP/3 server and client from Part 2 on top of it.

### QUIC Foundations

Before writing the UDP socket code, it's worth reviewing why QUIC ([RFC 9000](https://www.rfc-editor.org/rfc/rfc9000.html)) is structured this way.

#### Why UDP?
If TCP had problems with head-of-line blocking and handshake round trips, why not update TCP in the OS kernel instead of building a new protocol on UDP?

In practice, two things get in the way: middleboxes and OS upgrade cycles. Many firewalls, NATs, and routers inspect TCP headers and drop packets with unfamiliar TCP options or new IP protocol numbers. Even without middleboxes, rolling out kernel-level TCP changes across every server, phone, and laptop takes years.

UDP avoids both issues. A UDP header is just 8 bytes (`Source Port`, `Dest Port`, `Length`, `Checksum`) followed by payload bytes. UDP doesn't provide connection state, ordering, retransmissions, or congestion control, which means network hardware mostly forwards UDP packets without meddling with their contents. That lets QUIC implement stream multiplexing, reliability, and encryption in user space.

#### Independent Streams
In HTTP/2 over TCP, multiple HTTP streams share a single TCP byte stream. If Packet 1 (carrying data for Stream 0) is dropped on the network while Packet 2 (carrying data for Stream 4) arrives, the OS kernel holds Packet 2 in its buffer until Packet 1 is retransmitted:

{{< image src="holb.png" width="420px" class="center" >}}

In QUIC, each UDP datagram carries one or more **QUIC frames**, and `STREAM` frames include both a `Stream ID` and a byte `Offset`.

If a UDP packet for `Stream 0` is lost, only `Stream 0` has to wait for retransmission. `Stream 4` and `Stream 8` can still be read by the HTTP/3 layer right away. Within a single stream, if chunk 2 (`Offset=1000`) arrives before chunk 1 (`Offset=0`), the stream buffer stores chunk 2 at offset 1000 and hands both chunks to `Read` once chunk 1 arrives.

#### Faster Handshakes
As I showed in [gRPC Over HTTP/3](/posts/grpc-over-http3/), TCP with TLS requires a 3-way TCP handshake (`SYN`, `SYN-ACK`, `ACK`) before the TLS handshake (`ClientHello`, `ServerHello`) can start.

QUIC combines these steps by putting TLS 1.3 handshake messages inside QUIC `CRYPTO` frames (`0x06`) in the first `Initial` UDP packet. That reduces a new connection handshake to **1-RTT**. When a client reconnects to a server where it already has a cached session ticket and transport parameters, it can also send **0-RTT** packets carrying encrypted HTTP/3 `HEADERS` and `DATA` frames along with the `Initial` packet.

### Debugging QUIC

With HTTP/1.1, you can test a server using `telnet` or `nc`. With QUIC over UDP, the wire format is binary and encrypted from the first packet.

When I'm debugging real QUIC traffic, I usually rely on Wireshark with `SSLKEYLOGFILE`. Both `quic-go` and browsers support exporting TLS keys to a file if you set `SSLKEYLOGFILE=/tmp/keys.log` in your environment. Once you point Wireshark at that file under *Preferences -> Protocols -> TLS*, it decrypts the QUIC packets and lets you inspect the `CRYPTO`, `ACK`, `STREAM`, and `HTTP/3` frames inside each datagram. Another useful tool is `qlog`: if you set `QLOGDIR=/tmp/qlog` when running a `quic-go` program, it writes JSON trace files that you can load into [qvis.quictools.info](https://qvis.quictools.info/) to see packet timelines and flow control state. And for our own QUIC framing code here, unit tests that push out-of-order `STREAM` frames into a stream buffer are the fastest way to check reassembly edge cases.

### Packet and Frame Layout

[RFC 9000 Section 17](https://www.rfc-editor.org/rfc/rfc9000.html#name-packet-formats) defines two packet header formats, selected by the top bit of the first byte (`Header Form`):

{{< image src="quic-packets.svg" width="820px" class="center" inline="true" >}}

Notice that QUIC packets include a **Connection ID** even though UDP already has a source IP and port. This supports **Connection Migration**. If your phone moves from Wi-Fi to cellular, your source IP address and UDP port change. A TCP connection breaks because it is tied to the 4-tuple `(SrcIP, SrcPort, DstIP, DstPort)`. A QUIC server routes packets by their `Destination Connection ID`, so an existing connection and its open HTTP/3 streams can continue across the new network path.

The payload of a QUIC packet contains one or more **QUIC Frames** ([RFC 9000 Section 19](https://www.rfc-editor.org/rfc/rfc9000.html#name-frame-types-and-formats)). Here are the structs we use in `quic_frame.go`:

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
	Type     uint64 // PING (0x01), ACK (0x02), CRYPTO (0x06), STREAM (0x08..0x0f), CONNECTION_CLOSE (0x1d)
	StreamID uint64
	Offset   uint64
	Fin      bool
	Data     []byte
	// ... ACK, CRYPTO, and CONNECTION_CLOSE fields ...
}
```

Most of the work happens in the **`STREAM` frame (`0x08` to `0x0f`)**. The lowest 3 bits of the frame type act as flags (`OFF = 0x04`, `LEN = 0x02`, `FIN = 0x01`), and the lowest 2 bits of the `Stream ID` indicate who opened the stream and whether it is bidirectional or unidirectional:

{{< image src="quic-stream-frame.svg" width="820px" class="center" inline="true" >}}

When encoding a `STREAM` frame in `AppendQUICFrame`, we set the low bits on the type byte whenever `Offset > 0` or `Fin == true`:

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

| Low 2 Bits | Meaning | Stream IDs |
| :--- | :--- | :--- |
| `0b00` (`0x0`) | Client Bidirectional (Requests) | `0, 4, 8...` |
| `0b01` (`0x1`) | Server Bidirectional | `1, 5, 9...` |
| `0b10` (`0x2`) | Client Unidirectional (Control) | `2, 6, 10...` |
| `0b11` (`0x3`) | Server Unidirectional (Control) | `3, 7, 11...` |

Encoding the initiator and direction in the lowest two bits of the `Stream ID` lets the client and server open new streams concurrently without coordinating IDs. That's why our client's first request in Part 1 used `Stream 0`, its unidirectional control stream used `Stream 2`, and the server's unidirectional control stream used `Stream 3`.

*(Note: A full QUIC implementation like `quic-go` encrypts packet payloads and protects header bits using keys derived from the TLS 1.3 handshake in `CRYPTO` frames. In our toy transport here, we leave the frames unencrypted over UDP so we can focus on packet headers, connection IDs, `ACK` frames, and stream multiplexing without adding a full TLS 1.3 packet protector.)*

### Streams Over UDP

Now let's look at how `quic_conn.go` builds ordered streams on top of `net.ListenPacket("udp", addr)`.

Each `toyQUICStream` stores incoming slices of data in a map keyed by their byte `Offset` (`chunks map[uint64][]byte`), along with `readOffset` and `finOffset`:

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

When the HTTP/3 server or client calls `stream.Read(p)`, it checks whether a chunk is available at `s.readOffset`:

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

If two UDP datagrams arrive out of order so `offset=6 ("world!")` shows up before `offset=0 ("hello ")`, `s.chunks[6]` stays in the map until `offset=0` arrives. Once `offset=0` is received, `Read` returns `"hello world!"` in order and then returns `io.EOF`. Other streams on the connection (`s.conn.streams[otherStreamID]`) have their own `chunks` map and `readOffset`, so they aren't blocked while one stream waits on a missing offset.

At the connection level, `DialToyQUIC` opens a UDP socket, picks a 64-bit Connection ID, and sends a Long Header `Initial` packet with a `CRYPTO` frame (`CLIENT_HELLO_ALPN_H3`). On the server side, `toyQUICListener.readLoop()` reads UDP datagrams and routes packets for existing connections by `pkt.DestConnID`. When an `Initial` packet arrives for a new connection, the listener creates a `toyQUICConn`, sends back a Long Header `Handshake` packet (`ACK` + `CRYPTO` frame), and hands the connection to `Accept()`. Inside the connection, `handlePacket` dispatches incoming `STREAM` frames by `f.StreamID`, initializes peer-opened streams based on the low 2 bits of `StreamID`, and replies with `ACK` frames.

### Plugging in HTTP/3

Here is our `go.mod` for Part 3:

```go
module http3-part-3

go 1.25.5

require golang.org/x/net v0.55.0
```

`github.com/quic-go/quic-go` is no longer in `go.mod`. The only external dependency is `golang.org/x/net` for the Huffman table in `qpack.go` that we also used in the HTTP/2 posts.

We copy `frame.go`, `qpack.go`, `server.go`, and `client.go` as-is from Part 2, wire `ListenQUIC` and `DialQUIC` to `ListenToyQUIC` and `DialToyQUIC` in `main.go`, and run the tests and demo:

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
[POST /echo] Status: 200 OK | Body: Zero quic-go imports! Pure UDP -> QUIC -> QPACK -> HTTP/3!
```

When `client.Do(req)` runs here, `Client` encodes the `*http.Request` headers with QPACK in `qpack.go`, writes those headers and the body into HTTP/3 `HEADERS` (`0x01`) and `DATA` (`0x00`) frames in `frame.go`, and sends a `SETTINGS` (`0x04`) frame on unidirectional Control Stream `2`. Underneath, `toyQUICConn` wraps those stream writes into `STREAM` (`0x08..0x0f`) frames with stream IDs, byte offsets, and `FIN` bits inside QUIC Short Header packets and sends them over a `net.PacketConn` UDP socket. On the server side, `toyQUICListener` matches the incoming UDP packets by `Destination Connection ID`, sends `ACK` (`0x02`) frames, reassembles the stream chunks in `toyQUICStream`, and passes the streams up to our HTTP/3 `Server`, which decodes the QPACK headers, calls the `http.ServeMux` handler, and writes the HTTP/3 response back across the QUIC streams over UDP.

### Wrapping Up

Across this series ([HTTP/0.9](/posts/http0.9-from-scratch/), [HTTP/1.0](/posts/http1.0-from-scratch/), [HTTP/1.1](/posts/http1.1-from-scratch/), [HTTP/2](/posts/http2-from-scratch-part-1/), and now **HTTP/3**), we went from sending a single line of text (`GET /index.html`) over a TCP socket to multiplexing binary frames and QPACK headers over UDP streams.

Building the HTTP/3 framing and QPACK layer first before writing the QUIC transport underneath turned out to be a much nicer way to learn both protocols. Working with QUIC streams from the HTTP/3 side first makes it much clearer why QUIC's packet headers, stream IDs, and offset framing work the way they do.

The complete runnable code and tests for this article are available on GitHub in [http3-from-scratch-part-3/go](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/http3-from-scratch-part-3/go).
