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

QUIC ([RFC 9000](https://www.rfc-editor.org/rfc/rfc9000.html)) fixes this by moving off TCP entirely and running over UDP (`net.ListenPacket("udp", addr)`). Because UDP just delivers raw datagrams in whatever order they arrive without blocking in the kernel, QUIC can manage streams in user space—so a lost packet on one stream never stalls another.

Of course, ditching TCP means we lose everything TCP and TLS were doing for us underneath. To replace `quic-go` with our own UDP socket, we have to handle three jobs ourselves:

1. **Packets:** Parsing QUIC packet headers off each UDP datagram.
2. **Encryption:** Running the TLS 1.3 handshake inside UDP packets so we can encrypt and decrypt payloads.
3. **Streams:** Taking stream chunks that arrive over UDP and stitching them back into an ordered `io.Reader` and `io.Writer` for our HTTP/3 code.

### Inside a Packet

When we read a UDP datagram off the socket, what's inside?

A single UDP datagram can carry one or more QUIC packets (servers like Cloudflare often pack an Initial packet and a Handshake packet into the same datagram to save round trips). We check the highest bit of the first byte to see which of the two [RFC 9000 packet headers](https://www.rfc-editor.org/rfc/rfc9000.html#name-packet-formats) we're looking at:

{{< image src="quic-packets.svg" width="820px" class="center" inline="true" >}}

* **Long Header** (first bit `1`): Used during the handshake when both sides are still negotiating versions, connection IDs, and keys.
* **Short Header** (first bit `0`): A compact header used for normal HTTP/3 traffic once the handshake is done.

Both headers include a **Destination Connection ID** so endpoints can route packets to the right connection even if the client's IP address or port changes.

Once a packet is decrypted, its payload is simply a list of [QUIC frames](https://www.rfc-editor.org/rfc/rfc9000.html#name-frame-types-and-formats). Here is how we model packets and frames in `quic_frame.go`:

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

Notice that HTTP/3 has frames (`HEADERS`, `DATA`, `SETTINGS`), and QUIC *also* has frames (`CRYPTO`, `ACK`, `STREAM`). The layering looks like this:

* A **UDP datagram** carries one or more **QUIC packets**.
* A **QUIC packet** carries **QUIC `STREAM` frames**, which provide the byte stream for each stream ID.
* Inside that stream's bytes live the **HTTP/3 frames** we built in Part 1 and Part 2.

A QUIC **STREAM frame** has a type byte between `0x08` and `0x0f`. Its lowest three bits act as flags telling the parser whether the Offset field, Length field, and FIN bit are included:

{{< image src="quic-stream-frame.svg" width="820px" class="center" inline="true" >}}

When writing a STREAM frame in `quic_frame.go`, we set those flag bits on the base type (`0x08`) and encode the fields using the varint helpers from Part 1:

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

If we want to talk to `https://kmcd.dev/` over UDP, we can't send plaintext packets. Every QUIC packet on the wire—including the very first packet—is encrypted under [RFC 9001](https://www.rfc-editor.org/rfc/rfc9001.html).

In HTTP/2, we gave Go's `crypto/tls` package a TCP connection and let `tls.Client` handle the handshake and encryption automatically. Over UDP, there is no TCP connection to hand to `tls.Client`. Instead, QUIC carries the TLS 1.3 handshake messages inside `CRYPTO` frames (which I wrote about in [gRPC Over HTTP/3](/posts/grpc-over-http3/)) and encrypts each UDP packet itself.

Fortunately, Go's standard library includes `tls.QUICClient` and `tls.QUICServer` specifically for this. Instead of reading and writing to a socket, `tls.QUICClient` gives us raw TLS 1.3 handshake bytes to wrap in `CRYPTO` frames, and hands us the traffic secrets as each stage of the handshake completes. From those secrets, we derive the AES-GCM packet keys in `quic_crypto.go`:

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

That raises a chicken-and-egg question: before the TLS handshake has exchanged any keys with the server, how is the very first packet encrypted? Both sides derive an initial key from the client's random Destination Connection ID and a hardcoded 20-byte salt defined in [RFC 9001 Section 5.2](https://www.rfc-editor.org/rfc/rfc9001.html#name-initial-secrets). Anyone watching the network can decrypt an Initial packet using that public salt, which is fine—its job is just to protect the opening handshake against accidental packet corruption on the wire until real TLS keys take over.

One small gotcha when talking to real servers: we also have to pad our client's first UDP datagram out to 1,200 bytes ([RFC 9000 Section 14.1](https://www.rfc-editor.org/rfc/rfc9000.html#name-initial-datagram-size)), otherwise Cloudflare silently drops it to prevent UDP amplification attacks.

### Building Streams

Once packets are decrypted and we start receiving STREAM frames, we need to turn those frames into something our HTTP/3 code from Part 2 can read from.

Each stream in `quic_conn.go` implements Go's `io.Reader` and `io.Writer` interfaces. Because UDP datagrams can arrive out of order, every STREAM frame carries a byte `Offset` indicating where its data belongs in the stream. When a frame arrives, `pushFrame` stores the chunk in a map keyed by its byte offset and wakes up any waiting reader:

```go
func (s *quicStream) pushFrame(offset uint64, data []byte, fin bool) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if len(data) > 0 {
        copied := make([]byte, len(data))
        copy(copied, data)
        s.chunks[offset] = copied
    }
    if fin {
        s.finReceived = true
        s.finOffset = offset + uint64(len(data))
    }
    s.cond.Broadcast()
}
```

When our HTTP/3 parser calls `Read` on a stream, we check whether a chunk is waiting at the stream's current `readOffset`. If UDP datagrams arrive out of order—say offset 6 arrives before offset 0—offset 0 isn't in the map yet, so `Read` blocks on the condition variable until offset 0 arrives, without holding up any other stream on the connection:

```go
func (s *quicStream) Read(p []byte) (int, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    for {
        if chunk, ok := s.chunks[s.readOffset]; ok {
            n := copy(p, chunk)
            if n == len(chunk) {
                delete(s.chunks, s.readOffset)
            } else {
                s.chunks[s.readOffset+uint64(n)] = chunk[n:]
                delete(s.chunks, s.readOffset)
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

### Debugging with Wireshark

One downside of moving from HTTP/1.1 on TCP to QUIC on UDP is that you can't just use `nc` or `telnet` to see what your code is doing.

When I was working on my `quic-go` PRs, I used Wireshark with `SSLKEYLOGFILE=/tmp/keys.log` configured under *Preferences -> Protocols -> TLS* so it could decrypt the QUIC packets from curl and Chrome. You can also set `QLOGDIR` when running `quic-go` to dump `.qlog` trace files and inspect the packet timeline in [qvis.quictools.info](https://qvis.quictools.info/).

### The Result

Now we can plug our UDP transport into the `Conn` interface from Part 2 and drop `github.com/quic-go/quic-go` from `go.mod`. That leaves `golang.org/x/net` as our only external dependency (which we added back in [HTTP/2 From Scratch: Part 4](/posts/http2-from-scratch-part-4/) for its HPACK/QPACK Huffman table):

```go
module http3-part-3

go 1.25.5

require golang.org/x/net v0.55.0
```

When I pointed `main.go` at `https://kmcd.dev/` for the first time, the QUIC handshake succeeded and the HTML body came back right away, but the response headers included `Purpose: cloudflare` and no `Server` header at all. It turned out I had placed `x-frame-options` at index 62 of the QPACK static table instead of index 97, shifting entries 62 through 98 off by one. Fixing the table order decoded `Server: cloudflare` and `Alt-Svc` into their proper names.

Here is `main.go`, which fetches a URL over HTTP/3 using our custom QUIC transport and prints the response headers along with the first 200 bytes of the body:

```go
func main() {
    targetURL := "https://kmcd.dev/"
    if len(os.Args) > 1 {
        targetURL = os.Args[1]
        if !strings.Contains(targetURL, "://") {
            targetURL = "https://" + targetURL
        }
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client := NewClient()

    req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
    if err != nil {
        log.Fatal(err)
    }
    req.Header.Set("User-Agent", "kmcd-h3-scratch/1.0")

    resp, err := client.Do(req)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Fatal(err)
    }

    preview := body
    if len(preview) > 200 {
        preview = preview[:200]
    }

    fmt.Printf("URL:          %s\n", targetURL)
    fmt.Printf("Proto:        %s\n", resp.Proto)
    fmt.Printf("Status:       %s\n", resp.Status)
    fmt.Printf("Server:       %s\n", resp.Header.Get("Server"))
    fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
    fmt.Printf("Alt-Svc:      %s\n", resp.Header.Get("Alt-Svc"))
    fmt.Printf("Body Length:  %d bytes\n", len(body))
    fmt.Printf("Body Preview: %s\n", string(preview))
}
```

Running our test suite and `go run .` tests our local loopback server (`ListenQUIC`) as well as live HTTP/3 requests to `https://kmcd.dev/` and `https://cloudflare-quic.com/`:

```shell
$ go test -v ./...
=== RUN   TestOutOfOrderStreamReassembly
--- PASS: TestOutOfOrderStreamReassembly (0.00s)
=== RUN   TestEndToEndHTTP3OverQUIC
--- PASS: TestEndToEndHTTP3OverQUIC (0.01s)
PASS
ok  	http3-part-3	0.861s

$ go run .
URL:          https://kmcd.dev/
Proto:        HTTP/3.0
Status:       200 OK
Server:       cloudflare
Content-Type: text/html; charset=utf-8
Alt-Svc:      h3=":443"; ma=86400
Body Length:  28672 bytes
Body Preview: <!doctype html><html lang=en><head><meta name=generator content="Hugo 0.152.2"><meta charset=UTF-8><meta http-equiv=X-UA-Compatible content="ie=edge"><meta name=viewport content="width=device-width,in

$ go run . https://cloudflare-quic.com/
URL:          https://cloudflare-quic.com/
Proto:        HTTP/3.0
Status:       200 OK
Server:       cloudflare
Content-Type: text/html
Alt-Svc:      h3=":443"; ma=86400
Body Length:  125959 bytes
Body Preview: <!DOCTYPE html>
<html lang="en" itemscope="" itemtype="http://schema.org/Article"><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
    <title>QUIC | Cloudflare</title>
    <me
```

### Wrapping Up

That brings us to the end of the HTTP/3 posts and the "HTTP from Scratch" series ([HTTP/0.9](/posts/http0.9-from-scratch/), [HTTP/1.0](/posts/http1.0-from-scratch/), [HTTP/1.1](/posts/http1.1-from-scratch/), [HTTP/2](/posts/http2-from-scratch-part-1/), and now **HTTP/3**).

See all of the code mentioned in this article in [http3-from-scratch-part-3/go](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/http3-from-scratch-part-3/go).
