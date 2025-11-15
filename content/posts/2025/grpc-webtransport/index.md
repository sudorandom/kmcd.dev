---
categories: ["article"]
tags: ["webtransport", "http3", "grpc", "go"]
date: "2025-10-15T10:00:00Z"
description: "A hands-on experiment exploring how WebTransport over HTTP/3 could enable full-duplex gRPC streaming in the browser and why it’s not quite there yet, at least not with Go."
cover: "cover.png"
images: ["/posts/grpc-webtransport/cover.png"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Attempting gRPC Streaming with WebTransport"
slug: "grpc-webtransport"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-webtransport/
draft: true
---

WebSockets have powered real-time web apps for over a decade, but they’re starting to show their age. Let's talk about WebTransport, a new API standard built on HTTP/3 and QUIC. It's currently an **IETF draft proposal** ([spec](https://datatracker.ietf.org/doc/html/draft-ietf-webtrans-overview)), meaning the standard is still evolving, but it promises to fix WebSockets' biggest limitations and finally bring full-duplex gRPC to the browser. It also brings some new features like unreliable data transfer that make it better for real-time communications where retransmissions are typically useless. You can follow its browser implementation status and API details on the [Mozilla Developer Network](https://developer.mozilla.org/en-US/docs/Web/API/WebTransport).

In this article, I want to explore WebTransport by attempting to reimplement gRPC's streaming RPCs using WebTransport using Go.

## The Caveats of gRPC-Web

Before diving into WebTransport, let’s talk about why gRPC-Web exists and why it falls short for real-time communication.

Standard **gRPC** is a high-performance framework that leverages modern protocols like HTTP/2 to enable powerful communication patterns, especially between backend services. Its full power lies in its native support for streaming, which comes in four flavors:
1. **Unary:** The classic request/response.
2. **Server streaming:** The client sends a single request and gets a stream of responses back.
3. **Client streaming:** The client sends a stream of requests and gets a single response back.
4. **Bidirectional streaming:** The client and server can send messages to each other in any order.

The problem? Browsers can't speak native gRPC. That’s why gRPC-Web was created: a compatibility layer that translates gRPC calls into browser-friendly requests. However, this translation comes with a major compromise. Because gRPC-Web is fundamentally limited to the request/response nature of older HTTP semantics, **it cannot support client streaming or bidirectional streaming**.

This is a major drawback for applications that require true real-time, two-way communication, like chats, collaborative editors, or live data dashboards. In short, the promise of gRPC-Web falls short in the same places that gRPC excels at.

## What is WebTransport?

Enter WebTransport. It's a new web API that offers low-latency, bidirectional, client-server messaging. Crucially, WebTransport is still an active IETF draft proposal, not a finalized web standard. This means its specification can—and does—change, requiring both browsers and server libraries to stay in sync. This "moving target" is a key factor in its current adoption challenges.

Unlike WebSockets, which are built on a single, ordered stream of messages, WebTransport is built on top of HTTP/3 and QUIC. This foundation gives it some powerful capabilities:

* **Multiple streams:** Open several independent streams of data.
* **Unidirectional and bidirectional streams:** Flexible communication patterns.
* **Out-of-order delivery:** Data from one stream doesn't block another.
* **Reliable and Unreliable Data Transfer:**
  * **Reliable (Streams):** WebTransport streams (`createBidirectionalStream()` or `createUnidirectionalStream()`) are reliable and ordered, just like TCP. When you send data over a stream, you can be sure it will arrive in the correct order, without any missing pieces. This is ideal for critical data like chat messages, file transfers, or the initial state of an application.
  * **Unreliable (Datagrams):** WebTransport also provides a `datagrams` API, which works more like UDP. Data sent as a datagram is not guaranteed to arrive, nor is it guaranteed to arrive in any particular order. This might sound problematic, but it's extremely useful for latency-sensitive data where speed is more important than perfect accuracy. For example, in a real-time game, you'd rather drop an old packet with a player's previous position than delay the stream to re-transmit it. The same applies to live video or audio, where a lost frame is preferable to a frozen stream. This "choose your own adventure" for data reliability is a powerful feature that WebSockets lack.

These features make WebTransport a natural candidate to finally bring **full-featured gRPC streaming** to the web.

## Possible Alternatives

Before diving into an example, it's worth considering the alternatives.

### WebRTC
While WebRTC also provides real-time, peer-to-peer streaming, it operates outside of HTTP semantics, making it harder to monitor, debug, and integrate with existing observability tools. Additionally, WebRTC was created for peers; WebTransport is for clients and servers.

### WebSockets

While WebRTC serves a different purpose (peer-to-peer), the more direct comparison for WebTransport is the technology it's poised to replace: WebSockets. For over a decade, WebSockets have been the standard for client-server real-time communication. However, they are built on a single TCP connection, which introduces limitations like **head-of-line blocking**. A closer look reveals where WebTransport really shines:

| Feature | WebSockets | WebTransport |
|----------|-------------|--------------|
| Transport | TCP | QUIC (HTTP/3) |
| Multiplexing | ❌ Single stream | ✅ Multiple streams |
| Reliability | Always reliable | Reliable *and* unreliable modes |
| Ordered messages | Always ordered | Optional |
| Built-in backpressure | ❌ | ✅ |

The ability to manage multiple streams without head-of-line blocking, choose between reliable and unreliable delivery, and handle backpressure natively are game-changers. These aren't just incremental improvements; they are the *exact* features needed to overcome the limitations that have held back protocols like gRPC in the browser.

## Why is Adoption Stalling?

So, where does the gRPC team stand on this? The official [streaming roadmap for gRPC-Web](https://github.com/grpc/grpc-web/blob/master/doc/streaming-roadmap.md) has long been a source of discussion. However, the outlook is not promising. A key [GitHub issue](https://github.com/grpc/grpc-web/issues/24) tracking bidirectional streaming over WebTransport was recently closed with the status "not planned," as the gRPC-Web project has decided not to pursue new major features.

It's frustrating that a solution seems so close, yet so far. The gRPC project, for all its technical brilliance, has historically focused on the needs of large-scale microservice architectures. Browser developers have often felt like second-class citizens, as the organization's priorities seem geared towards large enterprise use-cases, leaving the web's potential untapped.

On top of this, full browser support is still pending. While Chrome, Edge, and Firefox are onboard, Safari remains stubbornly behind. Until Apple joins the table, WebTransport won’t be ready for wide adoption. You can always check the latest browser support on [caniuse.com/webtransport](https://caniuse.com/webtransport).

As I said before, WebTransport is still in the draft specification phase. There are only drafts of the spec, so the recommended implementation may change significantly before it's adopted. Understandably, this likely makes many developers less likely to invest time into it and it makes even less sense to bet a business on this technology at this stage.

Despite these setbacks, this shouldn't stop the community from experimenting. For internal tools, native applications, or contexts where you control the client environment, WebTransport is a viable and exciting option today.

## WebTransport in Go

The most reliable and supported implementation of WebTransport in Go, [`quic-go/webtransport-go`](https://github.com/quic-go/webtransport-go), is currently maintained as an unfunded hobby project by Marten Seemann. This isn't the first time I've relied on his excellent work; my own experience with the underlying `quic-go` library highlights its importance to the ecosystem. I previously contributed trailer support to `quic-go`, which was a critical step in enabling [gRPC Over HTTP/3](/posts/grpc-over-http3/), a journey I detailed in a previous series. I hope that at the end of this investigation I will end up helping this effort in some small way. The unfunded nature of the WebTransport library might be an issue, as Marten explained:

> webtransport-go has been unfunded since the beginning of 2024... as of June 2024, I will be ceasing maintenance work on the project.

This is a stark reminder that much of the modern web still runs on unpaid passion.

## Where Does This Leave Us?
So where are we now? WebTransport is stable in Chrome, supported in Firefox, and inching toward production-readiness. It’s not quite universal, but it’s mature enough to experiment with for internal apps, native clients, closed ecosystems, or for applications where Safari support is not a requirement.

If your users are internal or your browser matrix doesn’t include Safari, you can start experimenting with WebTransport today.

## Let's Try It: An Experiment in Go

To put this all to the test, let's build a simple experiment: a client-server application in Go that uses WebTransport to stream messages. The server will echo whatever the client sends, and the client will send a timestamped message each second. If you've ever written a small WebSocket app, this will look very familiar.

### The Server

The server sets up an HTTP/3 server and upgrades incoming requests on the `/webtransport` endpoint to a WebTransport session. It then enters a loop, accepting new streams and echoing back any messages it receives.

```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

func main() {
	server := &webtransport.Server{
		H3: http3.Server{
			Addr: ":4433",
		},
	}

	http.HandleFunc("/webtransport", func(rw http.ResponseWriter, r *http.Request) {
		conn, err := server.Upgrade(rw, r)
		if err != nil {
			log.Printf("upgrading failed: %s", err)
			rw.WriteHeader(500)
			return
		}

		go func() {
			log.Printf("accepted session: %s", conn.RemoteAddr())
			for {
				stream, err := conn.AcceptStream(r.Context())
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						log.Printf("accepting stream failed: %s", err)
					}
					return
				}
				log.Printf("accepted stream: %d", stream.StreamID())

				go func() {
					for {
						buf := make([]byte, 1024)
						n, err := stream.Read(buf)
						if err != nil {
							log.Printf("read finished with error: %s", err)
							return
						}
						log.Printf("read %d bytes: %s", n, buf[:n])

						_, err = stream.Write(buf[:n])
						if err != nil {
							log.Printf("write finished with error: %s", err)
							return
						}
						log.Printf("wrote %d bytes: %s", n, buf[:n])
					}
				}()
			}
		}()
	})

	log.Println("Starting server on :4433")
	if err := server.ListenAndServeTLS("localhost.pem", "localhost-key.pem"); err != nil {
		log.Fatal(err)
	}
}
```

**Note:** This code requires local TLS certificates. I use the amazing [mkcert CLI](https://github.com/FiloSottile/mkcert) for this:

```bash
mkcert -install
mkcert localhost
```

### The Client

The client dials the server, opens a bidirectional stream, and then starts two goroutines: one to send a message every second, and another to listen for incoming messages from the server.

```go
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quic-go/webtransport-go"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dialer := &webtransport.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	_, conn, err := dialer.Dial(ctx, "https://localhost:4433/webtransport", nil)
	if err != nil {
		return err
	}
	defer conn.CloseWithError(0, "graceful shutdown")

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return err
	}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-gctx.Done():
				log.Println("shutting down writer")
				return gctx.Err()
			case t := <-ticker.C:
				msg := fmt.Sprintf("Hello! The time is now %v", t.Format(time.DateTime))
				_, err = stream.Write([]byte(msg))
				if err != nil {
					return err
				}
				log.Printf("Wrote: %s", msg)
			}
		}
	})

	g.Go(func() error {
		for {
			buf := make([]byte, 1024)
			n, err := stream.Read(buf)
			if err != nil {
				log.Printf("shutting down reader: %v", err)
				return err
			}
			log.Printf("Read: %s", buf[:n])
		}
	})

	go func() {
		<-gctx.Done()
		stream.CancelRead(0)
		stream.Close()
	}()

	log.Println("Running, press CTRL+C to stop...")
	defer log.Println("shutting down")

	return g.Wait()
}
```

In principle, this example demonstrates how easily you can implement WebTransport with the right libraries. The API is clean and familiar, showing its potential as a foundation for a gRPC transport layer.

## The Results: A Roadblock

Here's where our experiment hit the wall. I couldn't get the Go example to work with modern browsers.

It appears the `quic-go/webtransport-go` implementation is based on an older draft of the WebTransport spec, which browsers apparently no longer support. This is a direct consequence of the project being unfunded: without active maintenance, it's fallen out of sync with the evolving standard, rendering it functionally broken for our experiment.

I've tried other solutions, like the Rust implementation [wtransport](https://github.com/BiagioFesta/wtransport), and it works just fine, confirming the issue is with the Go library, not the concept.

The server logs confirm this incompatibility. After the initial connection, the browser rejects the session:
```
2025/10/17 21:40:09 Adding connection ID 17db7129.
2025/10/17 21:40:09 server 	<- &wire.PingFrame{}
2025/10/17 21:40:09 server -> Sending coalesced packet (2 parts, 1280 bytes) for connection 9d62c944ad1538f4
2025/10/17 21:40:09 server 	Long Header{Type: Initial, DestConnectionID: (empty), SrcConnectionID: a9ea2138, Token: (empty), PacketNumber: 0, PacketNumberLen: 2, Length: 595, Version: v1}
2025/10/17 21:40:09 server 	-> &wire.AckFrame{LargestAcked: 1, LowestAcked: 1, DelayTime: 0s}
2025/10/17 21:40:09 server 	-> &wire.CryptoFrame{Offset: 0, Data length: 90, Offset + Data length: 90}
2025/10/17 21:40:09 server 	Long Header{Type: Handshake, DestConnectionID: (empty), SrcConnectionID: a9ea2138, PacketNumber: 0, PacketNumberLen: 2, Length: 615, Version: v1}
2025/10/17 21:40:09 server 	-> &wire.CryptoFrame{Offset: 0, Data length: 593, Offset + Data length: 593}
2025/10/17 21:40:09 server 	Short Header{DestConnectionID: (empty), PacketNumber: 0, PacketNumberLen: 2, KeyPhase: 0}
2025/10/17 21:40:09 server 	-> &wire.NewConnectionIDFrame{SequenceNumber: 1, RetirePriorTo: 0, ConnectionID: 17db7129, StatelessResetToken: 0xc2d7ba94e16551f57f9d9ddd5d61bf36}
2025/10/17 21:40:09 server -> Sending packet 1 (35 bytes) for connection 9d62c944ad1538f4, 1-RTT (ECN: ECT(0))
2025/10/17 21:40:09 server 	Short Header{DestConnectionID: (empty), PacketNumber: 1, PacketNumberLen: 2, KeyPhase: 0}
2025/10/17 21:40:09 server 	-> &wire.StreamFrame{StreamID: 3, Fin: false, Offset: 0, Data length: 14, Offset + Data length: 14}
2025/10/17 21:40:09 server Parsed a coalesced packet. Part 1: 1037 bytes. Remaining: 213 bytes.
2025/10/17 21:40:09 server <- Reading packet 2 (1037 bytes) for connection a9ea2138, Initial
2025/10/17 21:40:09 server 	Long Header{Type: Initial, DestConnectionID: a9ea2138, SrcConnectionID: (empty), Token: (empty), PacketNumber: 2, PacketNumberLen: 1, Length: 1023, Version: v1}
2025/10/17 21:40:09 server 	<- &wire.ConnectionCloseFrame{IsApplicationError:false, ErrorCode:0x12e, FrameType:0x6, ReasonPhrase:"199:TLS handshake failure (ENCRYPTION_HANDSHAKE) 46: certificate unknown. SSLErrorStack:[handshake.cc:297] error:1000007d:SSL routines:OPENSSL_internal:CERTIFICATE_VERIFY_FAILED"}
2025/10/17 21:40:09 server Closing connection with error: CRYPTO_ERROR 0x12e (remote) (frame type: 0x6): 199:TLS handshake failure (ENCRYPTION_HANDSHAKE) 46: certificate unknown. SSLErrorStack:[handshake.cc:297] error:1000007d:SSL routines:OPENSSL_internal:CERTIFICATE_VERIFY_FAILED
```

The key message in the logs is CERTIFICATE_VERIFY_FAILED. In plain English, the browser and server failed to agree on the security handshake, which is a classic symptom of protocol incompatibility between a client and server running different draft versions of a standard.

## A Universal gRPC Proxy?

One powerful idea is a proxy that can terminate all three types of gRPC connections:

1.  **Standard gRPC:** From other backend services.
2.  **gRPC-Web:** From current web browsers.
3.  **gRPC + WebTransport:** From modern browsers and native clients.

This proxy would inspect the incoming connection and route the gRPC calls to the appropriate backend service.

```mermaid
graph LR
    A[gRPC Client] --> P{Universal Proxy};
    B[gRPC-Web Client] --> P;
    C[gRPC+WebTransport Client] --> P;
    P --> S[gRPC Backend];
```

Such a proxy would allow you to write your services in standard gRPC and let the proxy handle the complexity of supporting different client types. This would be similar to gRPC-Web's proxy-based design.

It could be built from scratch or it could leverage existing functionality in a project like [Vanguard](https://github.com/connectrpc/vanguard-go), which I've used to enable support for gRPC, gRPC Web, ConnectRPC and REST in [FauxRPC](https://fauxrpc.com) or [Envoy](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/other_protocols/grpc), which can already terminate and translate gRPC and/or gRPC-Web connections.

Building such a proxy could bridge the gap between legacy browsers and future protocols. This would be a practical path while the standards catch up. But if I'm going to write this in Go, I need the WebTransport library to catch up to the latest WebTransport draft.

## Final Thoughts

WebTransport isn’t just “WebSockets but faster.” It’s a shift toward giving web developers the same transport-level flexibility backend engineers have enjoyed for years. Once it’s universally supported (and if the gRPC ecosystem embraces it), we could finally see browser apps communicating with backend systems using the same rich streaming semantics as microservices.

WebTransport presents a clear and powerful evolution for real-time web communication. Its native support for multiple streams and unreliable data transfer makes it a technically superior successor to WebSockets for many use cases, especially as a transport for full-duplex gRPC.

However, its future is not yet certain. The lack of official adoption by the gRPC-Web team and the absence of support in Safari are significant roadblocks. Furthermore, the maintenance state of key libraries, like `quic-go` for Go developers, adds another layer of uncertainty. As we've seen, when a core implementation falls out of sync with browser drafts, it can render the technology unusable. This lack of forward movement has led some to speculate that gRPC-Web itself may be an abandoned project. For now, WebTransport remains a technology of the near future. It is viable for controlled environments and internal tools, but not yet ready for the mainstream web. The technology is promising, but as our experiment shows, its real-world use depends on the entire ecosystem—from browser vendors to library maintainers—staying in sync.
