---
categories: ["article"]
tags: ["golang", "http3", "quic", "protocols", "networking"]
date: "2026-11-19T10:00:00Z"
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
---

In [Part 1](/posts/http3-from-scratch-part-1/), we opened and closed streams over QUIC, wrote our HTTP/3 variable-length integer (`varint`) framing, and adapted our HTTP/2 HPACK code into a QPACK encoder and decoder.

At the end of Part 1, our test client and server could send `HEADERS` and `DATA` frames over a single QUIC stream. In this post, we'll turn that into a working HTTP/3 server and client that can talk to `curl`, `quic-go`'s `http3` package, and a web browser.

Before we get to the server loop, there's an obvious question to answer: since web traffic has defaulted to TCP port 443 for decades, how does a browser know when to connect to a server over UDP instead?

### Negotiating HTTP/3

With HTTP/2, protocol negotiation was easy because HTTP/1.1 and HTTP/2 both live on top of TCP and TLS. As we saw in [HTTP/2 From Scratch: Part 1](/posts/http2-from-scratch-part-1/), the client passes `NextProtos: []string{"h2", "http/1.1"}` in its TLS `ClientHello` on TCP port 443 and the server picks `h2` during the handshake.

We can't do that for HTTP/3 since QUIC uses UDP instead of TCP, so browsers have two ways to learn that a host supports HTTP/3.

#### The Alt-Svc Header
If you scroll back to the `openssl s_client` dump in [HTTP/2 From Scratch: Part 1](/posts/http2-from-scratch-part-1/), you might have noticed this header in the response from `kmcd.dev`:

```http
alt-svc: h3=":443"; ma=86400
```

That's [HTTP Alternative Services](https://www.rfc-editor.org/rfc/rfc7838.html). When you visit `kmcd.dev` for the first time over TCP, that header tells your browser that port 443 also speaks `h3` over UDP for the next 86,400 seconds (24 hours), so the browser can switch to QUIC on the next request:

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

Since TCP and UDP ports are separate, a Go server can listen on TCP `:8443` and UDP `:8443` at the same time in the same binary.

#### DNS HTTPS Records
Cloudflare also publishes an `HTTPS` DNS record ([RFC 9460](https://www.rfc-editor.org/rfc/rfc9460.html)) for `kmcd.dev`. If you run `dig kmcd.dev HTTPS`, you'll see `alpn="h3,h2"` in the answer. Browsers check for that `HTTPS` record when resolving the domain name, so they can connect to UDP port 443 on the very first request without doing a TCP round trip first:

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

Pointing Chrome at our own Go server on `localhost:8443` has two catches. On TCP, we can use a self-signed certificate and click past Chrome's warning screen, but Chrome's QUIC client rejects self-signed certs outright. And without a DNS `HTTPS` record or an `Alt-Svc` header for `localhost:8443`, Chrome has no reason to try UDP on that port.

To get a certificate that Chrome actually trusts on `localhost`, I added [`mkcert`](https://github.com/FiloSottile/mkcert) to `.mise.toml` and generated a local cert:

```shell
$ mkcert -install
$ mkcert localhost 127.0.0.1
```

Testing with `curl` is much simpler since `curl --http3-only -k` still allows self-signed certs over QUIC (note that macOS's built-in `/usr/bin/curl` doesn't have HTTP/3 enabled, so I used the `curl` binary from `mise`):

```shell
$ curl --http3-only -k -v https://127.0.0.1:8443/headers
```

And to make Chrome talk HTTP/3 directly to `localhost:8443` without running a separate TCP server for `Alt-Svc`, you can pass `--origin-to-force-quic-on`:

```shell
$ open -a "Google Chrome" --args \
    --enable-quic \
    --origin-to-force-quic-on=localhost:8443 \
    https://localhost:8443/
```

### Wrapping the QUIC Connection

In `transport.go`, I put a small `Conn` interface around `quic-go`. Right now those methods just call into `quic-go`, and in Part 3 we'll swap in our own UDP implementation behind the same interface:

```go
type Conn interface {
	AcceptStream(ctx context.Context) (Stream, error)
	OpenStreamSync(ctx context.Context) (Stream, error)
	AcceptUniStream(ctx context.Context) (ReceiveStream, error)
	OpenUniStreamSync(ctx context.Context) (SendStream, error)
}
```

`AcceptStream` and `OpenStreamSync` are the bidirectional stream methods we used in Part 1, and `AcceptUniStream` and `OpenUniStreamSync` open the one-way streams that we'll use for the HTTP/3 control stream.

### The Server Loop

Back in [HTTP/2 From Scratch: Part 2](/posts/http2-from-scratch-part-2/), we sent and received SETTINGS frames on Stream 0. QUIC doesn't reserve stream 0 for connection settings—in fact, we already used stream 0 for our echo request in Part 1. Instead, [RFC 9114 Section 6.2.1](https://www.rfc-editor.org/rfc/rfc9114.html#name-control-streams) says that both the client and server have to open a separate one-way control stream when they connect. The first byte written to a one-way stream is a varint identifying its stream type (`0x00` for a control stream), followed immediately by a SETTINGS frame:

{{< image src="h3-server-streams.svg" width="820px" class="center" inline="true" >}}

When I first wrote `sendControlStreamSettings`, I put `defer uni.Close()` at the top of the function out of habit, and `quic-go` immediately dropped the connection with `H3_CLOSED_CRITICAL_STREAM`. Unlike a request stream, the control stream has to stay open for the entire connection because closing it sends a FIN bit that the peer treats as a connection error:

{{< image src="control-stream-meme.jpg" width="480px" class="center" >}}

For each new QUIC connection in `server.go`, we open our control stream to send our settings, start a background goroutine to read the client's unidirectional streams, and then accept bidirectional request streams in a loop:

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

In `sendControlStreamSettings`, we open a one-way stream, write the `0x00` control stream type varint, and write a `SETTINGS` frame (`0x04`) with `SettingsQPACKMaxTableCapacity` set to `0`:

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

We set `SettingsQPACKMaxTableCapacity` to `0` because our QPACK decoder from Part 1 only supports the static table and string literals, so we don't want curl or Chrome sending us dynamic table updates.

#### Implementing http.Handler

In `handleRequestStream`, we read the first HEADERS frame from the stream, decode the QPACK headers into an `*http.Request`, and pass it to `s.Handler.ServeHTTP`:

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

Here `h3BodyReader` reads DATA frames from the QUIC stream as the handler reads the request body, and `h3ResponseWriter` encodes the response headers and body back out to the stream.

### Building the Client

For `client.go`, I wanted the same `Do(req *http.Request) (*http.Response, error)` signature that we built in [HTTP/2 From Scratch: Part 4](/posts/http2-from-scratch-part-4/).

The client does the exact same dance as the server, just in reverse: dial QUIC, open a control stream to send our settings, open a bidirectional stream for the request, write the HEADERS and DATA frames, and close the write side of the stream so the server sees `EOF`:

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

We reuse `h3BodyReader` for `resp.Body` too. Just like the bug we hit in the HTTP/2 client, we can't `defer conn.Close()` inside `Do` because the caller hasn't read the response body yet. Instead, we pass `closer: conn` into `h3BodyReader` so the QUIC connection closes when the caller calls `resp.Body.Close()`.

### Running It

In `main.go`, we set up an `http.NewServeMux()` with three routes (`/`, `/headers`, and `/echo`) and call them with our `Client`:

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

### What's Next?

Our HTTP/3 server and client can now talk to `quic-go` and `curl`, but we are still relying on `quic-go` to open the UDP sockets and manage the QUIC streams underneath.

In **Part 3**, we'll see why HTTP/2 still suffers from head-of-line blocking at the TCP layer, how QUIC fixes it, how QUIC packets and stream frames are laid out on the wire, and swap out `quic-go` for our own toy QUIC transport.

See all of the code mentioned in this article in [http3-from-scratch-part-2/go](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/http3-from-scratch-part-2/go).
