---
categories: ["article"]
tags: ["streaming", "webtransport", "http3", "grpc", "go"]
date: "2026-09-01T10:00:00Z"
description: "A practical look at streaming data in the browser, from response streams and WebSockets to WebTransport over HTTP/3."
cover: "cover.png"
images: ["/posts/grpc-webtransport/cover.png"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Let's Talk About Streaming Data on the Web"
slug: "grpc-webtransport"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-webtransport/
draft: true
---

“Streaming data” on the web might mean consuming a response as bytes arrive, receiving server events, using a long-lived bidirectional channel, or sharing one connection between independent streams. These workloads have different requirements, but we often reduce the choice to HTTP requests or WebSockets.

I want to look at the available options and then experiment with WebTransport. I came at this from gRPC: could WebTransport make client and bidirectional streaming practical in a browser? That question deserves its own follow-up, so I am starting with the transport.

## The Problem

The browser handles server-to-client streaming reasonably well. Things get harder when the client and server both need to send data independently, or when we need several concurrent streams without opening several connections.

gRPC is what led me here because its client and bidirectional streaming calls are not available through gRPC-Web. It is only one example. Chats, collaborative editors, games, live dashboards, media, and remote control interfaces all eventually run into some version of the same problem: we need data moving in both directions, often across several independent flows, but the browser does not expose a general-purpose TCP or QUIC socket.

## What We Have Today

### Fetch Streams and Server-Sent Events

For downloads, generated content, event feeds, and other server-to-client flows, streaming `fetch()` or Server-Sent Events may be enough. Fetch exposes a `ReadableStream`, while Server-Sent Events add an event-oriented channel with reconnection. Both keep ordinary HTTP semantics and remain visible in browser network tools.

{{< d2 width="100%" >}}
direction: right

classes: {
  endpoint: {
    style: { fill: "#102a2e"; stroke: "#2dd4bf"; font-color: "#e6fffb"; stroke-width: 2 }
  }
}

browser: Browser
browser.class: endpoint
server: Server
server.class: endpoint

browser -> server: Request { style: { stroke: "#2dd4bf"; font-color: "#2dd4bf"; stroke-width: 3 } }
server -> browser: Response stream { style: { stroke: "#2dd4bf"; font-color: "#2dd4bf"; stroke-width: 3 } }
{{< /d2 >}}

They are not enough when the client also needs to stream data, or when both sides need to communicate independently over the same call.

### WebRTC

WebRTC gives us bidirectional media and data channels, but it was designed for communication between peers. Signaling, NAT traversal, and ICE make sense for calls and peer-to-peer applications but are awkward for a normal client-server API. It also sits outside ordinary HTTP infrastructure, making it harder to operate alongside a web backend.

### WebSockets

WebSockets are the best general-purpose option we have today for bidirectional client-server communication in a browser. They are widely supported, well understood, and visible in browser developer tools. If an application needs one long-lived, ordered stream of messages, a WebSocket is often exactly the right answer.

{{< d2 width="100%" >}}
direction: right

classes: {
  endpoint: {
    style: { fill: "#102a2e"; stroke: "#2dd4bf"; font-color: "#e6fffb"; stroke-width: 2 }
  }
}

browser: Browser
browser.class: endpoint
server: Server
server.class: endpoint

browser <-> server: One ordered stream { style: { stroke: "#2dd4bf"; font-color: "#2dd4bf"; stroke-width: 3 } }
{{< /d2 >}}

The problems start when we need several independent streams. We can multiplex all of them onto one WebSocket, but then we have to build a protocol with custom frames and stream IDs, plus routing, lifecycle, cancellation, backpressure, and error isolation on top of a single ordered connection. At that point, we are reimplementing a rough version of HTTP/2 in application code. It is a substantial and complex piece of infrastructure. Because every logical stream ultimately shares one ordered TCP byte stream, packet loss stalls unrelated streams until the missing data is retransmitted.

The other option is to open a separate WebSocket for every stream or RPC. That means another connection and another WebSocket handshake each time, which wastes resources and makes short-lived calls unnecessarily slow.

So WebSockets are still the best thing we have today, but they make us choose between a complicated application-level multiplexer and a pile of separate connections.

## Enter WebTransport

WebTransport is a browser API for low-latency client-server communication over HTTP/3 and QUIC. Instead of giving us one ordered channel, it gives us a session that can carry multiple independent streams and datagrams:

* A single session can carry multiple unidirectional and bidirectional streams. Data waiting on one stream does not block the others.
* Streams are reliable and ordered. They work well for chat messages, file transfers, application state, or an RPC that cannot lose data.
* Datagrams are unreliable and unordered. They work well when old data has little value, such as player positions or individual frames of live audio and video.

{{< d2 width="100%" >}}
direction: right

classes: {
  endpoint: {
    style: { fill: "#102a2e"; stroke: "#2dd4bf"; font-color: "#e6fffb"; stroke-width: 2 }
  }
  stream: {
    style: { stroke: "#2dd4bf"; font-color: "#2dd4bf"; stroke-width: 3 }
  }
}

browser: Browser
browser.class: endpoint
server: Server
server.class: endpoint

browser <-> server: Stream A { class: stream }
browser <-> server: Stream B { class: stream }
browser <-> server: Stream C { class: stream }
browser <-> server: Datagrams {
  class: stream
  style.stroke-dash: 4
}
{{< /d2 >}}

This directly addresses the WebSocket tradeoff. Independent streams share one connection without sharing ordered delivery or backpressure, and we do not have to invent the multiplexing layer ourselves.

At a high level, this is QUIC streams exposed to JavaScript. If HTTP/2 streams already make sense to you, the model should feel familiar. The important difference is that QUIC prevents packet loss on one stream from holding up data on the others.

Of course, none of these capabilities help us if browsers do not expose them. Until recently, browser support was the obvious reason not to build around WebTransport.

That is no longer the case. [Safari 26.4 enabled WebTransport](https://webkit.org/blog/17862/webkit-features-for-safari-26-4/), joining current Chrome, Edge, and Firefox releases. The version number is not a typo: Apple renumbered Safari alongside iOS, iPadOS, and macOS in 2025. The current [Can I Use support report](https://caniuse.com/webtransport) is green across modern major browsers. There are still old and legacy browsers without support, but browser coverage is no longer a reason to avoid testing WebTransport.

{{< figure src="caniuse.png" caption="Source: [caniuse.com/webtransport](https://caniuse.com/webtransport)" >}}

## Browser DevTools Lack WebTransport Inspection

WebTransport works. Debugging it does not. As of July 2026, none of the built-in developer tools in Chrome, Firefox, or Safari provides a WebSocket-style view of WebTransport traffic. You cannot select a session and inspect its streams or datagrams, view individual messages or frames, search their payloads, or download the exchanged application data as a response body.

There is no single response body for DevTools to reveal after the handshake. A WebTransport session can carry multiple independent streams and datagrams for its entire lifetime. A useful inspector needs to understand those primitives, reconstruct the streams, and expose their decrypted application bytes.

Chrome's [Network domain](https://chromedevtools.github.io/devtools-protocol/tot/Network/) emits only `webTransportCreated`, `webTransportConnectionEstablished`, and `webTransportClosed`. These notifications show that a session existed; they do not provide any visibility into what crossed it. There are no corresponding events for streams, datagrams, frames, or payload bytes. Chrome's Network panel provides dedicated message views for [WebSockets and event streams](https://developer.chrome.com/docs/devtools/network/reference/), while WebTransport has none. The [Firefox Network Monitor](https://firefox-source-docs.mozilla.org/devtools-user/network_monitor/) and [Safari Network tab](https://webkit.org/web-inspector/network-tab/) have the same fundamental limitation: neither provides built-in inspection of WebTransport traffic.

This is not only a concern for native gRPC. Connect and gRPC-Web are more likely starting points for browser RPC work, and they already layer envelopes, Protobuf messages, compression, status, and streaming semantics over HTTP. Moving either protocol onto WebTransport would add session and stream multiplexing beneath those layers while taking away the request and response body visibility developers currently depend on.

Without corresponding browser tooling, diagnosing a single RPC may require correlating application messages, protocol envelopes, WebTransport streams, and server-side transport logs by hand.

When a connection will not establish, Chrome's [`chrome://net-export/`](https://new.chromium.org/for-testers/providing-network-details/) records browser network events. Wireshark or a QUIC-aware proxy can inspect setup when TLS keys are available. These help with connection failures, but decrypted QUIC still leaves us reconstructing streams and decoding the application protocol.

WebTransport is still in the draft specification phase, so the recommended implementation may change before it is finalized. That is a reason to be cautious about betting a business on it, but not a reason to avoid experimenting with it.

Production applications also need to account for networks that block UDP, which is still common on corporate and managed networks. The specification includes a draft for [WebTransport over HTTP/2](https://datatracker.ietf.org/doc/draft-ietf-webtrans-http2/), but browser and server support is much more limited than HTTP/3. It also cannot preserve QUIC's unreliable delivery or stream independence. The Go example in this article only supports HTTP/3, so an application using it needs a fallback to WebSockets, streaming Fetch, or Server-Sent Events.

## What About gRPC?

Standard gRPC supports unary, server-streaming, client-streaming, and bidirectional-streaming calls over HTTP/2. Browsers cannot speak native gRPC, so gRPC-Web translates those calls into requests the browser can make. That translation supports unary and server-streaming calls, but it cannot provide client streaming or bidirectional streaming.

The official [gRPC-Web streaming roadmap](https://github.com/grpc/grpc-web/blob/master/doc/streaming-roadmap.md) has discussed this for years, but the project is no longer pursuing major features. Its issue tracking [bidirectional streaming](https://github.com/grpc/grpc-web/issues/24) is closed as "not planned."

Connect-ES has an open [WebTransport transport issue](https://github.com/connectrpc/connect-es/issues/1106) motivated by bringing client and bidirectional streaming to browsers. It has no assignee, milestone, or implementation yet, but the interest is there.

WebTransport supplies the bidirectional streams, but not gRPC message framing, metadata, status codes, deadlines, cancellation, or the mapping between RPCs and streams. Those decisions still need an implementation.

The next experiment is to find out how much of gRPC, gRPC-Web, or Connect maps cleanly onto WebTransport without creating an incompatible browser-only protocol.

## Trying WebTransport in Go

To put this all to the test, let's build a simple experiment: a client-server application in Go that uses WebTransport to stream messages. The server will echo whatever the client sends, and the client will send a timestamped message each second. If you've ever written a small WebSocket app, this will look very familiar.

### The Server

The server sets up an HTTP/3 server and upgrades incoming requests on the `/webtransport` endpoint to a WebTransport session. It then enters a loop, accepting new streams and echoing back any messages it receives.

```go
cert, err := tls.LoadX509KeyPair("localhost.pem", "localhost-key.pem")
if err != nil {
	return err
}

tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
h3 := &http3.Server{
	Addr:            ":4433",
	Handler:         mux,
	TLSConfig:       tlsConfig,
	EnableDatagrams: true,
}

server := &webtransport.Server{H3: h3}
webtransport.ConfigureHTTP3Server(h3)

mux.HandleFunc("/webtransport", func(rw http.ResponseWriter, r *http.Request) {
	session, err := server.Upgrade(rw, r)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	go func() {
		for {
			stream, err := session.AcceptStream(context.Background())
			if err != nil {
				return
			}
			go io.Copy(stream, stream) // Echo this stream independently.
		}
	}()
})

// The complete sample also starts the HTTPS page server and handles shutdown.
log.Fatal(server.ListenAndServe())
```

**Note:** This code requires local TLS certificates. I install the amazing [mkcert CLI](https://github.com/FiloSottile/mkcert) and generate the certificate pair that the sample server loads:

```bash
mkcert -install
mkcert localhost
```

Run these commands from the Go sample directory. `mkcert` creates `localhost.pem` and `localhost-key.pem`, which are loaded by both the Go and Rust servers.

### Testing WebTransport in Chrome and Firefox

WebTransport can fail even when HTTPS works. Installing the mkcert CA makes the certificate valid for normal HTTPS, but browsers apply additional restrictions to QUIC and HTTP/3.

#### Firefox

For Firefox, open `about:config`, set `network.http.http3.disable_when_third_party_roots_found` to `false`, and restart Firefox. Firefox otherwise disables HTTP/3 when a user-installed root CA such as mkcert is present. Use a separate development profile because this preference affects the whole profile.

#### Chrome, Edge, and Brave

Chrome uses the system trust store, where `mkcert -install` installs the local CA. WebTransport over HTTP/3 applies an additional requirement that the certificate be issued by a publicly known root. Chromium-based browsers provide a developer mode that relaxes this additional requirement for local testing:

1. Open `chrome://flags/#webtransport-developer-mode`.
2. Set **WebTransport Developer Mode** to **Enabled**.
3. Relaunch the browser.

The browser still validates the certificate against the system trust store; this flag allows that trusted root to be a locally installed CA such as mkcert. It is intended only for development. After relaunching, open `http://localhost:8080`.

### The Client

The Go client needs only a WebTransport session and a bidirectional stream. The complete sample wraps this in signal handling and separate read/write goroutines, but the transport-specific part is small:

```go
var dialer webtransport.Dialer

_, session, err := dialer.Dial(
	ctx,
	"https://localhost:4433/webtransport",
	nil,
)
if err != nil {
	return err
}
defer session.CloseWithError(0, "done")

stream, err := session.OpenStreamSync(ctx)
if err != nil {
	return err
}

if _, err := stream.Write([]byte("Hello, WebTransport!")); err != nil {
	return err
}

reply := make([]byte, 1024)
n, err := stream.Read(reply)
if err != nil {
	return err
}
log.Printf("Received: %s", reply[:n])
```

The browser side follows the same shape. This example opens three bidirectional streams concurrently and sends one message over each:

```javascript
const session = new WebTransport("https://localhost:4433/webtransport");
await session.ready;

async function echo(message) {
	const stream = await session.createBidirectionalStream();
	const writer = stream.writable.getWriter();
	const reader = stream.readable.getReader();

	await writer.write(new TextEncoder().encode(message));
	const { value } = await reader.read();
	return new TextDecoder().decode(value);
}

const replies = await Promise.all([
	echo("first stream"),
	echo("second stream"),
	echo("third stream"),
]);
```

Each stream is accepted independently. The sample gives every stream its own goroutine, so if work on the first stream stalls, the second and third can continue. This is the part that would require a custom multiplexer over one WebSocket.

The JavaScript version still looks simpler, although removing every `if err != nil {` line from the Go example does give it a slight head start.

## Conclusion

Streaming `fetch()` and Server-Sent Events fit server-to-client data. WebSockets provide a widely deployed bidirectional channel when one ordered stream is enough. WebTransport fits applications that need independent streams, per-stream backpressure, unreliable datagrams, or isolation from cross-stream head-of-line blocking.

The experiment worked: a browser and a Go client connected to the same Go server over HTTP/3, opened independent bidirectional streams, and exchanged data without routing everything through one ordered WebSocket channel. The programming model is surprisingly simple, and the transport provides primitives that would otherwise require a substantial application-level multiplexer.

Local development also exposed an unexpectedly sharp edge: trusting a mkcert CA for HTTPS does not automatically make every browser accept it for WebTransport over QUIC. Firefox and Chromium both require an explicit development setting. That is fine for an experiment, but it is not something I would want to explain in a production onboarding guide.

WebTransport is the first browser API that exposes modern transport primitives directly enough to solve the independent-stream problem cleanly. Browser support is here. Tooling still lags behind: DevTools cannot inspect the traffic, local certificates need browser-specific setup, and production deployments still need fallbacks.

WebTransport deliberately stops at the transport layer. It does not define an application protocol. A follow-up can take Connect, gRPC-Web, or gRPC framing beyond the transport demo, map an RPC onto WebTransport streams, and test whether the result preserves useful semantics without creating an entirely new protocol in disguise.
