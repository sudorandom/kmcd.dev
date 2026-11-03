---
categories: ["article"]
tags: ["golang", "http3", "quic", "protocols", "networking"]
date: "2026-11-17T10:00:00Z"
description: "Negotiating HTTP/3 with Alt-Svc and DNS, forcing local browsers onto QUIC, and building a full HTTP/3 server and client."
cover: "cover.svg"
images: ["/posts/http3-from-scratch-part-2/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "HTTP/3 From Scratch: Part 2"
slug: "http3-from-scratch-part-2"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/http3-from-scratch-part-2/
series: ["HTTP from Scratch"]
draft: true
---

In [Part 1](/posts/http3-from-scratch-part-1/), we opened and closed streams over QUIC, wrote our HTTP/3 variable-length integer (`varint`) framing, and adapted our HTTP/2 HPACK code into a QPACK encoder and decoder.

At the end of Part 1, our test client and server could send `HEADERS` and `DATA` frames over a single QUIC stream. In this post, we'll turn that into a working HTTP/3 server and client that can talk to `curl`, `quic-go`'s `http3` package, and a web browser.

Before we get to the server loop, there's an obvious question to answer: since web traffic has defaulted to TCP port 443 for decades, how does a browser know when to connect to a server over UDP instead?

### Negotiating HTTP/3

With HTTP/2, negotiation happens inside the same connection. Both HTTP/1.1 and HTTP/2 run over TCP and TLS. When a browser connects to TCP port 443, it sends an ALPN (Application-Layer Protocol Negotiation) list in its TLS `ClientHello` with `["h2", "http/1.1"]`. The server selects `"h2"` during the handshake and the connection continues as HTTP/2.

With HTTP/3, you can't negotiate a UDP protocol inside a TCP handshake without paying for a TCP+TLS connection first. Browsers use two main ways to discover HTTP/3 support.

#### The Alt-Svc Header
If you look back at the `openssl s_client` output in [HTTP/2 From Scratch: Part 1](/posts/http2-from-scratch-part-1/), you can see this response header from `kmcd.dev`:

```http
alt-svc: h3=":443"; ma=86400
```

[Alternative Services (`Alt-Svc`, RFC 7838)](https://www.rfc-editor.org/rfc/rfc7838.html) lets an HTTP/1.1 or HTTP/2 server over TCP advertise that the same origin is also available over `h3` on UDP port 443, and that the client can cache that hint for 86,400 seconds (24 hours). The first request still goes over TCP, and the browser opens a QUIC connection on UDP in the background for later requests:

```d2
shape: sequence_diagram

Browser
TCP: "TCP :443"
UDP: "UDP :443"

Browser -> TCP: "GET / (HTTP/2)"
TCP -> Browser: "200 OK + Alt-Svc: h3"
Browser -> UDP: "QUIC Handshake"
Browser -> UDP: "GET / (HTTP/3)"
UDP -> Browser: "200 OK"
```

Because TCP ports and UDP ports are separate in the OS, a single Go binary can listen on TCP `:8443` (serving HTTP/1.1 or HTTP/2 with an `Alt-Svc` header) and UDP `:8443` (serving HTTP/3) at the same time.

#### DNS HTTPS Records
Relying on `Alt-Svc` still means the first visit uses TCP. [RFC 9460](https://www.rfc-editor.org/rfc/rfc9460.html) added the `HTTPS` DNS record type so clients can find out about HTTP/3 during DNS resolution. When a browser looks up a domain, it queries `A`, `AAAA`, and `HTTPS` records together. If the `HTTPS` record includes `alpn="h3"`, the browser can start with a QUIC handshake on UDP right away:

```d2
shape: sequence_diagram

Browser
DNS
UDP: "UDP :443"

Browser -> DNS: "Query A + HTTPS"
DNS -> Browser: "IP + alpn=h3"
Browser -> UDP: "QUIC + HTTP/3 Request"
UDP -> Browser: "HTTP/3 Response"
```

### Testing Locally

Testing an HTTP/3 server on `localhost` in a browser can be frustrating if you don't know about Chrome's certificate rules. With HTTP/1.1 or HTTP/2, you can click through Chrome's self-signed certificate warning page and the browser will load the site. Even if you do that, Chrome will still refuse to use QUIC/HTTP/3 on an untrusted certificate and will stay on TCP without showing an error. And unless you pass a flag or serve `Alt-Svc` on TCP first, the browser won't try UDP on `localhost:8443` anyway.

To avoid certificate issues locally, I use [`mkcert`](https://github.com/FiloSottile/mkcert) (configured in `.mise.toml`) to create a locally trusted certificate:

```shell
$ mkcert -install
$ mkcert localhost 127.0.0.1
```

Once you have a valid cert, you can test with `curl` using `--http3-only` (which forces UDP instead of falling back to TCP):

```shell
$ curl --http3-only -k -v https://127.0.0.1:8443/headers
```

Or if you want to test with Chrome directly without running a companion TCP server for `Alt-Svc`, you can launch Chrome with `--origin-to-force-quic-on`:

```shell
$ open -a "Google Chrome" --args \
    --enable-quic \
    --origin-to-force-quic-on=localhost:8443 \
    https://localhost:8443/
```

### The Transport Interface

Since we are running on `quic-go` in this post and swapping in our own UDP QUIC code in Part 3, we define a small `Conn` interface in `transport.go`:

```go
type Conn interface {
	AcceptStream(ctx context.Context) (Stream, error)
	OpenStreamSync(ctx context.Context) (Stream, error)
	AcceptUniStream(ctx context.Context) (ReceiveStream, error)
	OpenUniStreamSync(ctx context.Context) (SendStream, error)
}
```

The interface has methods for both bidirectional streams (`AcceptStream` / `OpenStreamSync`) and unidirectional streams (`AcceptUniStream` / `OpenUniStreamSync`). We need the unidirectional streams for the HTTP/3 control stream.

### The Server Loop

In HTTP/2, connection-wide frames like `SETTINGS` and `GOAWAY` were sent on `Stream 0`. In HTTP/3, `Stream 0` is just the first client-initiated bidirectional request stream.

Instead, [RFC 9114 Section 6.2.1](https://www.rfc-editor.org/rfc/rfc9114.html#name-control-streams) specifies that both the client and server open a unidirectional control stream (`Stream Type = 0x00`) when a connection starts and send a `SETTINGS` frame (`0x04`) as its first frame. Each HTTP request and response uses a separate bidirectional stream:

{{< image src="h3-server-streams.svg" width="820px" class="center" inline="true" >}}

One important rule in the spec is that you cannot close the control stream while the connection is open. If you add `defer uni.Close()` after writing your `SETTINGS` frame, the other side treats that `FIN` bit as a fatal `H3_CLOSED_CRITICAL_STREAM` error and closes the connection:

{{< image src="control-stream-meme.jpg" width="480px" class="center" >}}

Here is how `Server.handleConnection` in `server.go` sets up a connection:

```go
func (s *Server) handleConnection(ctx context.Context, conn Conn) error {
	if err := sendControlStreamSettings(ctx, conn); err != nil {
		return err
	}

	go drainPeerUniStreams(ctx, conn)

	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return err
		}
		go s.handleRequestStream(ctx, conn, stream)
	}
}
```

In `sendControlStreamSettings`, we write the stream type varint (`0x00` for the control stream) followed by the `SETTINGS` frame (`0x04`) with `SettingsQPACKMaxTableCapacity (0x01) = 0`:

```go
func sendControlStreamSettings(ctx context.Context, conn Conn) error {
	uni, err := conn.OpenUniStreamSync(ctx)
	if err != nil {
		return err
	}

	var settingsPayload []byte
	settingsPayload = AppendVarint(settingsPayload, SettingsQPACKMaxTableCapacity)
	settingsPayload = AppendVarint(settingsPayload, 0)

	var buf []byte
	buf = AppendVarint(buf, StreamTypeControl)
	buf = AppendVarint(buf, FrameSettings)
	buf = AppendVarint(buf, uint64(len(settingsPayload)))
	buf = append(buf, settingsPayload...)

	_, err = uni.Write(buf)
	return err
}
```

Advertising `SettingsQPACKMaxTableCapacity = 0` tells the peer not to insert entries into the QPACK dynamic table and to stick to the 99-entry static table and string literals. That lets our `QPACKDecoder` from Part 1 handle headers from real clients like `curl` and Chrome without needing dynamic table state synchronization.

#### Handling Requests

For each bidirectional request stream, `handleRequestStream` reads the initial `HEADERS` frame, builds an `*http.Request`, and passes it to `s.Handler.ServeHTTP(w, req)`:

```go
func (s *Server) handleRequestStream(ctx context.Context, conn Conn, stream Stream) error {
	defer stream.Close()

	firstFrame, _ := ReadFrame(stream)
	headerFields, _ := NewQPACKDecoder().Decode(firstFrame.Payload)

	req := &http.Request{
		Proto:  "HTTP/3.0",
		Header: make(http.Header),
		Body:   &h3BodyReader{stream: stream},
	}
	populateRequestHeaders(req, headerFields)

	w := &h3ResponseWriter{stream: stream, headers: make(http.Header)}
	s.Handler.ServeHTTP(w, req.WithContext(ctx))
	if !w.sentHeaders {
		w.WriteHeader(http.StatusOK)
	}
	return nil
}
```

Rather than reading the entire request body into memory upfront, `h3BodyReader` reads `DATA` frames from the QUIC stream as the handler calls `req.Body.Read(p)`. On the write side, `h3ResponseWriter` implements `http.ResponseWriter`: `WriteHeader` encodes and writes the `HEADERS` frame, and `Write(p)` sends `p` inside an HTTP/3 `DATA` frame.

### Building the Client

On the client side (`client.go`), we want a `Client` with a standard `Do(req *http.Request) (*http.Response, error)` method, similar to what we built in [HTTP/2 From Scratch: Part 4](/posts/http2-from-scratch-part-4/).

`Client.Do` runs the same sequence from the client side:

```go
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	conn, _ := c.DialFunc(req.Context(), addr, c.TLSConfig)
	sendControlStreamSettings(req.Context(), conn)
	go drainPeerUniStreams(req.Context(), conn)

	stream, _ := conn.OpenStreamSync(req.Context())
	WriteFrame(stream, FrameHeaders, enc.Encode(headers))
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		WriteFrame(stream, FrameData, bodyBytes)
	}
	stream.Close()

	frame, _ := ReadFrame(stream)
	respHeaders, _ := NewQPACKDecoder().Decode(frame.Payload)

	resp := &http.Response{
		Proto:  "HTTP/3.0",
		Header: make(http.Header),
		Body:   &h3BodyReader{stream: stream, closer: conn},
	}
	populateResponseHeaders(resp, respHeaders)
	return resp, nil
}
```

We reuse `h3BodyReader` for `resp.Body` as well. When the caller closes `resp.Body`, it closes the underlying QUIC connection.

### The Result

In `main.go`, we register three routes (`/`, `/headers`, and `/echo`) on a standard `http.NewServeMux()` and call them using our HTTP/3 `Client`:

```shell
$ go run .
HTTP/3 server listening on UDP 127.0.0.1:64329

[GET /] Status: 200 OK | Body: Welcome to HTTP/3 from scratch! You requested GET /
[GET /headers] Status: 200 OK | Body: {"X-Custom-Header":["true"]}
[POST /echo] Status: 200 OK | Body: Echoing over HTTP/3 DATA frames!
```

In `server_test.go`, we also test our `Server` against `quic-go`'s `http3.Transport`, and test our `Client` against `quic-go`'s `http3.Server`:

```shell
$ go test -v ./...
=== RUN   TestInteropWithQuicGoHTTP3
--- PASS: TestInteropWithQuicGoHTTP3 (0.01s)
=== RUN   TestClientAgainstOfficialServer
--- PASS: TestClientAgainstOfficialServer (0.00s)
PASS
ok  	http3-part-2	0.398s
```

Both tests pass, confirming that our framing, control stream setup, and QPACK code interoperate with `quic-go`'s HTTP/3 implementation.

### What's Next?

Now that we have a working HTTP/3 server and client, we can look at the transport layer underneath it.

In **Part 3**, we'll drop down to **UDP**. We'll look at how QUIC avoids transport-layer head-of-line blocking, how 1-RTT and 0-RTT handshakes work, how QUIC frames packets and streams over UDP datagrams, and replace `quic-go` with our own QUIC transport layer.

The complete runnable code and tests for this article are available on GitHub in [http3-from-scratch-part-2/go](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/http3-from-scratch-part-2/go).
