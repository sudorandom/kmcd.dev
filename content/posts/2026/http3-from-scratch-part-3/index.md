---
categories: ["article"]
tags: ["golang", "http3", "quic", "udp", "protocols", "networking"]
date: "2026-11-26T10:00:00Z"
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
---

In [Part 1](/posts/http3-from-scratch-part-1/) and [Part 2](/posts/http3-from-scratch-part-2/), we built our HTTP/3 framing, QPACK encoder and decoder, control streams, and HTTP/3 server and client in Go. At the end of Part 2, our code could talk to curl, `quic-go`, and Chrome, but we were still relying on `quic-go` to handle the UDP sockets and QUIC streams.

In this post, we'll remove `quic-go`, build a minimal QUIC layer directly on top of UDP, and use it to fetch `https://kmcd.dev/` over HTTP/3.

### Why UDP?

At the end of [HTTP/2 From Scratch: Part 4](/posts/http2-from-scratch-part-4/), I talked about why HTTP/2 only solved head-of-line blocking at the application layer. Since HTTP/2 multiplexes all of its streams over a single TCP connection, it still inherits TCP's strict sequential packet ordering. Drop a single TCP packet on Stream 0, and the OS sits on data for Stream 4 in the kernel's TCP receive buffer while waiting for the retransmit:

{{< image src="holb.png" width="420px" class="center" >}}

QUIC ([RFC 9000](https://www.rfc-editor.org/rfc/rfc9000.html), which is a pretty sweet RFC number) ditches TCP altogether and runs straight over UDP (`net.ListenPacket("udp", addr)`).

### Inside a Packet

What's inside a QUIC UDP datagram?

Look at the highest bit (`0x80`) of the first byte we read from `net.PacketConn` to see which of the two [RFC 9000 packet headers](https://www.rfc-editor.org/rfc/rfc9000.html#name-packet-formats) is on the wire (`1` for a handshake Long Header, `0` for a 1-RTT Short Header):

{{< image src="quic-packets.svg" width="820px" class="center" inline="true" >}}

Here is how we model those packet headers and their decrypted [QUIC frames](https://www.rfc-editor.org/rfc/rfc9000.html#name-frame-types-and-formats) in `quic_frame.go`:

```go
type Packet struct {
    IsLongHeader bool
    LongType     uint8
    Version      uint32
    DestConnID   []byte
    SrcConnID    []byte
    PacketNumber uint32
    Frames       []QUICFrame
}

type QUICFrame struct {
    Type         uint64
    StreamID     uint64
    Offset       uint64
    Fin          bool
    Data         []byte
    LargestAcked uint64
    CryptoOffset uint64
    CryptoData   []byte
    ErrorCode    uint64
    ReasonPhrase string
}
```

### Stream Frames

Notice that HTTP/3 has frames (`HEADERS`, `DATA`, `SETTINGS`), and QUIC *also* has frames (`CRYPTO`, `ACK`, `STREAM`). The lowest three bits of the `STREAM` frame type byte act as presence flags for the fields that follow (`0x04` for `OFF`, `0x02` for `LEN`, and `0x01` for `FIN`):

{{< image src="quic-stream-frame.svg" width="820px" class="center" inline="true" >}}

When writing a STREAM frame in `quic_frame.go`, we set those flags on the base type (`0x08`) and encode the fields using the varint helpers from Part 1:

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

Look at the Stream ID field in the diagram above too. Remember in Part 1 and Part 2 how client request streams were numbered 0, 4, 8 while the control streams were 2 and 3? The lowest two bits of a QUIC stream ID are bitflags for who initiated the stream (0 for client, 1 for server) and whether it's bidirectional (0) or unidirectional (1). Counting up client bidirectional streams by one actually increments the wire ID by four.

### The Handshake

If we want to talk to `https://kmcd.dev/` over UDP, we can't send plaintext packets. Every QUIC packet on the wire, including the very first packet, is encrypted under [RFC 9001](https://www.rfc-editor.org/rfc/rfc9001.html).

In the HTTP/2 posts, we just handed a TCP `net.Conn` to `tls.Client` and let `crypto/tls` do the rest. With UDP there *is* no `net.Conn` to pass in, so we use Go's `tls.QUICClient` and `tls.QUICServer` instead and carry the TLS 1.3 handshake bytes ourselves inside `CRYPTO` frames (which I wrote about in [gRPC Over HTTP/3](/posts/grpc-over-http3/)).

As the handshake steps through each encryption level, `crypto/tls` hands us a secret and a cipher suite. In `derivePacketKeys`, the cipher suite picks the hash and AES key size, while the `"quic key"`, `"quic iv"`, and `"quic hp"` labels derive three separate values from that one secret:

```go
func derivePacketKeys(suite uint16, secret []byte) (*packetKeys, error) {
    var hashFunc func() hash.Hash = sha256.New
    keyLen := 16
    if suite == tls.TLS_AES_256_GCM_SHA384 {
        hashFunc = sha512.New384
        keyLen = 32
    }

    key := hkdfExpandLabel(hashFunc, secret, "quic key", nil, keyLen)
    iv := hkdfExpandLabel(hashFunc, secret, "quic iv", nil, 12)
    hpKey := hkdfExpandLabel(hashFunc, secret, "quic hp", nil, keyLen)

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    return &packetKeys{aead: aead, iv: iv, hpKey: hpKey}, nil
}
```

When `protectPacket` sends a packet out, it XORs that 12-byte `iv` with the packet number in `makeNonce`, runs `aead.Seal` over the frames (passing the unencrypted header in as the `additionalData` parameter), and calls `headerProtectionMask` with `hpKey` to mask the packet number in the header.

Wait: how do we get a `secret` for the very first packet before the TLS handshake has even started? Both sides feed the client's random Destination Connection ID and a hardcoded 20-byte salt from [RFC 9001 Section 5.2](https://www.rfc-editor.org/rfc/rfc9001.html#name-initial-secrets) into `derivePacketKeys` so they can read and write those initial `CRYPTO` frames.

### Building Streams

In `quic_conn.go`, `handlePacket` processes the frames from each decrypted packet:

```go
func (c *quicConn) handlePacket(pkt Packet) {
    var shouldAck bool
    for _, f := range pkt.Frames {
        switch f.Type {
        case QUICFramePing, QUICFrameHandshakeDone, QUICFrameNewConnectionID:
            shouldAck = true

        case QUICFramePathChallenge:
            shouldAck = true
            _ = c.sendPacketAtLevel(level, []QUICFrame{{
                Type: QUICFramePathResponse,
                Data: f.Data,
            }})

        case QUICFrameCrypto:
            shouldAck = true
            if data := c.readCrypto(level, f.CryptoOffset, f.CryptoData); len(data) > 0 {
                _ = c.tlsConn.HandleData(tlsLevel, data)
                _ = c.drainTLSEvents()
            }

        case QUICFrameStreamBase:
            shouldAck = true
            c.getOrCreateStream(f.StreamID).pushFrame(f.Offset, f.Data, f.Fin)
        }
    }

    if shouldAck {
        _ = c.sendPacketAtLevel(level, []QUICFrame{{
            Type:         QUICFrameAck,
            LargestAcked: uint64(pkt.PacketNumber),
        }})
    }
}
```

Each stream in `quic_conn.go` implements Go's `io.Reader` and `io.Writer` interfaces, storing incoming bytes in an `s.chunks` map by offset until our HTTP/3 parser calls `Read` at `s.readOffset`. If offset 6 shows up before offset 0, `Read` parks on `s.cond.Wait()` until offset 0 arrives, while other streams on the connection have their own `quicStream` state and keep running without waiting:

```go
func (s *quicStream) pushFrame(offset uint64, data []byte, fin bool) {
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

func (s *quicStream) Read(p []byte) (int, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    for {
        if chunk, ok := s.chunks[s.readOffset]; ok {
            n := copy(p, chunk)
            delete(s.chunks, s.readOffset)
            if n < len(chunk) {
                s.chunks[s.readOffset+uint64(n)] = chunk[n:]
            }
            s.readOffset += uint64(n)
            return n, nil
        }
        if s.finReceived && s.readOffset >= s.finOffset {
            return 0, io.EOF
        }
        if s.closed {
            return 0, io.ErrClosedPipe
        }
        s.cond.Wait()
    }
}
```

### The Result

Now `quic-go` is completely gone from `go.mod`! The only thing left in `go.mod` is `golang.org/x/net`, which we brought in back in [HTTP/2 From Scratch: Part 4](/posts/http2-from-scratch-part-4/) for its HPACK/QPACK Huffman table.

```shell
$ go run .
URL:          [https://kmcd.dev/](https://kmcd.dev/)
Proto:        HTTP/3.0
Status:       200 OK
Server:       cloudflare
Content-Type: text/html; charset=utf-8
Alt-Svc:      h3=":443"; ma=86400
Body Length:  28672 bytes
Body Preview: <!doctype html><html lang=en><head><meta name=generator content="Hugo 0.152.2"><meta charset=UTF-8><meta http-equiv=X-UA-Compatible content="ie=edge"><meta name=viewport content="width=device-width,in

$ go run . [https://cloudflare-quic.com/](https://cloudflare-quic.com/)
URL:          [https://cloudflare-quic.com/](https://cloudflare-quic.com/)
Proto:        HTTP/3.0
Status:       200 OK
Server:       cloudflare
Content-Type: text/html
Alt-Svc:      h3=":443"; ma=86400
Body Length:  125959 bytes
Body Preview: <!DOCTYPE html>
<html lang="en" itemscope="" itemtype="[http://schema.org/Article](http://schema.org/Article)"><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
    <title>QUIC | Cloudflare</title>
    <me
```

### Wrapping Up

Before you get any ideas: this is a terrible QUIC implementation. Remember at the beginning of the post when I talked about how QUIC avoids stalling other streams when a packet is lost? Well, our code doesn't actually retransmit lost packets; if a packet drops on the wire, that stream just hangs in `s.cond.Wait()` until the 10-second context timeout in `main.go` kills the connection. We also advertise a 1 MB flow-control window in `marshalTransportParams` without ever checking it. We implemented just enough to convince Cloudflare to hand over the contents of `kmcd.dev`.

Still, we went all the way from a one-line `GET /` over TCP in [HTTP/0.9](/posts/http0.9-from-scratch/) through [HTTP/1.0](/posts/http1.0-from-scratch/), [HTTP/1.1](/posts/http1.1-from-scratch/), [HTTP/2](/posts/http2-from-scratch-part-1/), and now **HTTP/3** over raw UDP sockets, which brings us to the end of the "HTTP from Scratch" series.

See all of the code mentioned in this article in [http3-from-scratch-part-3/go](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/http3-from-scratch-part-3/go).
