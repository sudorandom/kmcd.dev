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

At the end of [HTTP/2 From Scratch: Part 4](/posts/http2-from-scratch-part-4/), I talked about why HTTP/2 only solved head-of-line blocking at the application layer. Since HTTP/2 multiplexes all of its streams over a single TCP connection, it still inherits TCP's strict sequential packet ordering. If a single TCP packet is lost on Stream 0, the operating system holds back data for Stream 4 in the kernel's TCP receive buffer until that lost packet is retransmitted:

{{< image src="holb.png" width="420px" class="center" >}}

QUIC ([RFC 9000](https://www.rfc-editor.org/rfc/rfc9000.html), which is a pretty sweet RFC number) fixes this by moving off TCP entirely and running over UDP (`net.ListenPacket("udp", addr)`).

### Inside a Packet

What's inside a QUIC UDP datagram?

When we read a datagram from `net.PacketConn`, we check the highest bit (`0x80`) of the first byte to see which of the two [RFC 9000 packet headers](https://www.rfc-editor.org/rfc/rfc9000.html#name-packet-formats) we're looking at (`1` for the Long Header used during the handshake, or `0` for the compact Short Header used once 1-RTT data starts flowing):

{{< image src="quic-packets.svg" width="820px" class="center" inline="true" >}}

Inside the decrypted payload of either packet is a slice of [QUIC frames](https://www.rfc-editor.org/rfc/rfc9000.html#name-frame-types-and-formats). Here is how we model packets and frames in `quic_frame.go`:

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

In the HTTP/2 posts, we passed a TCP `net.Conn` to `tls.Client` and let Go's `crypto/tls` package handle the handshake and record encryption. With UDP, we don't have a stream connection to hand to `tls.Client`; QUIC transports the raw TLS 1.3 handshake records inside `CRYPTO` frames (which I wrote about in [gRPC Over HTTP/3](/posts/grpc-over-http3/)) and encrypts each UDP packet itself.

Go's `crypto/tls` package exposes `tls.QUICClient` and `tls.QUICServer` for this exact job. The cipher suite tells it which hash and AES key size to use; the labels (`quic key`, `quic iv`, and `quic hp`) keep each value separate, even though they come from the same secret:

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

To encrypt a packet in `protectPacket`, we XOR the 12-byte `iv` with the packet number (`makeNonce`) and pass the unencrypted header as Additional Authenticated Data (AAD) into `aead.Seal` so the header cannot be tampered with in transit (then `headerProtectionMask` encrypts a 16-byte sample of the ciphertext with `hpKey` to mask the packet number on the wire).

For the very first packet (padded out to 1,200 bytes under [RFC 9000 Section 14.1](https://www.rfc-editor.org/rfc/rfc9000.html#name-initial-datagram-size)), `deriveInitialKeys` hashes the client's Destination Connection ID with the 20-byte salt from [RFC 9001 Section 5.2](https://www.rfc-editor.org/rfc/rfc9001.html#name-initial-secrets). Inside these packets, the client and server exchange public keys as part of the TLS handshake (`drainTLSEvents` polls `c.tlsConn.NextEvent()` to write `CRYPTO` frames and install the resulting keys). Each side uses its own private key and the other side's public key to derive a shared secret, which is used to generate keys for protecting subsequent traffic.

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
            c.mu.Lock()
            c.cryptoReadChunks[level][f.CryptoOffset] = bytes.Clone(f.CryptoData)
            var contiguous []byte
            for {
                chunk, ok := c.cryptoReadChunks[level][c.cryptoReadOffset[level]]
                if !ok {
                    break
                }
                delete(c.cryptoReadChunks[level], c.cryptoReadOffset[level])
                c.cryptoReadOffset[level] += uint64(len(chunk))
                contiguous = append(contiguous, chunk...)
            }
            c.mu.Unlock()

            if len(contiguous) > 0 && c.tlsConn != nil {
                _ = c.tlsConn.HandleData(tlsLevel, contiguous)
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

When our HTTP/3 parser calls `Read` on a `quicStream`, we check `s.chunks` at `s.readOffset` (if offset 6 arrives before offset 0, `s.chunks[0]` won't exist yet, so `Read` blocks on `s.cond.Wait()` while other streams on the connection keep running):

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

Now `quic-go` is completely gone from `go.mod`! The only thing left in `go.mod` is `golang.org/x/net`, which we brought in back in [HTTP/2 From Scratch: Part 4](https://www.google.com/search?q=/posts/http2-from-scratch-part-4/) for its HPACK/QPACK Huffman table implementation.

```shell
\$ go test -v ./...
=== RUN   TestOutOfOrderStreamReassembly
--- PASS: TestOutOfOrderStreamReassembly (0.00s)
=== RUN   TestEndToEndHTTP3OverQUIC
--- PASS: TestEndToEndHTTP3OverQUIC (0.01s)
PASS
ok      http3-part-3    0.861s

\$ go run .
URL:          [https://kmcd.dev/](https://kmcd.dev/)
Proto:        HTTP/3.0
Status:       200 OK
Server:       cloudflare
Content-Type: text/html; charset=utf-8
Alt-Svc:      h3=":443"; ma=86400
Body Length:  28672 bytes
Body Preview: <!doctype html><html lang=en><head><meta name=generator content="Hugo 0.152.2"><meta charset=UTF-8><meta http-equiv=X-UA-Compatible content="ie=edge"><meta name=viewport content="width=device-width,in

\$ go run . [https://cloudflare-quic.com/](https://cloudflare-quic.com/)
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

Before you get any ideas: this is a terrible QUIC implementation. Remember at the beginning of the post when I talked about how QUIC avoids stalling other streams when a packet is lost? If a packet actually gets dropped with our code, we don't retransmit it. That stream just sits in `s.cond.Wait()` until the `10*time.Second` context timeout in `main.go` happens. We also have zero congestion control, we ignore flow-control limits after advertising 1 MB in `marshalTransportParams`, and our `ACK` frames only report `LargestAcked` instead of tracking ranges. We implemented just enough to convince Cloudflare to hand over the contents of `kmcd.dev`.

Still, we went all the way from a one-line `GET /` over TCP in [HTTP/0.9](https://www.google.com/search?q=/posts/http0.9-from-scratch/) through [HTTP/1.0](https://www.google.com/search?q=/posts/http1.0-from-scratch/), [HTTP/1.1](https://www.google.com/search?q=/posts/http1.1-from-scratch/), [HTTP/2](https://www.google.com/search?q=/posts/http2-from-scratch-part-1/), and now **HTTP/3** over raw UDP sockets, which brings us to the end of the "HTTP from Scratch" series.

See all of the code mentioned in this article in [http3-from-scratch-part-3/go](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/http3-from-scratch-part-3/go).
