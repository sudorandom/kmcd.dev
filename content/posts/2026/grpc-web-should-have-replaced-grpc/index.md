---
categories: ["article"]
tags: ["grpc", "grpc-web", "protobuf", "api", "rpc", "web", "http2", "connectrpc"]
date: "2026-06-28T10:00:00Z"
description: "gRPC-Web should have been the pressure that made gRPC simpler, more inspectable, and better suited for the web."
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC-Web Should Have Fixed gRPC"
slug: "grpc-web-should-have-fixed-grpc"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-web-should-have-fixed-grpc/
draft: true
---

gRPC did a lot right.

It turned Protocol Buffers from "how do I encode this struct?" into "how do I define this API?" You write a schema, generate clients and servers, and now you have typed messages, streaming, deadlines, cancellation, interceptors, and a cross-language contract that mostly works the way it says it will.

For backend systems, that bargain is great. If you control the clients, servers, load balancers, and deployment environment, gRPC is a very nice tool. I use it. I like it.

But gRPC also made a very expensive bet on HTTP/2-specific behavior, and that bet got awkward the moment browsers entered the picture.

Browsers could negotiate HTTP/2 just fine. That was not the problem. The problem was that frontend JavaScript did not get access to the HTTP/2 features gRPC needed. `fetch()` could not send raw HTTP/2 frames, manage streams the way native gRPC expected, or read trailers as normal response metadata.

So we ended up in a strange place: the browser could speak HTTP/2 under the hood, but your JavaScript app still could not make a standard gRPC call. Enter gRPC-Web, which is where I think the ecosystem made the wrong call.

gRPC-Web was treated as a browser compatibility layer. It should have been treated as a warning sign.

Maybe not the exact gRPC-Web protocol we ended up with, but the idea behind it: keep protobuf, keep generated clients, keep the RPC model, work with normal HTTP infrastructure, and stop making every API call behave like a custom protocol hiding inside HTTP.

In other words, gRPC-Web should have fixed gRPC.

## The original bargain

The original gRPC protocol is documented as [gRPC over HTTP/2](https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md). At a high level, a unary call usually looks like this:

1. Send an HTTP/2 `POST`.
2. Put the service and method in the path.
3. Send metadata as HTTP headers.
4. Send a length-prefixed protobuf message in the body.
5. Return a response body containing a length-prefixed protobuf message.
6. Put the final RPC result in HTTP trailers, especially `grpc-status` and `grpc-message`.

There are details and edge cases, including trailers-only responses, but the important part is the same: the final application status lives in gRPC metadata, not in the normal HTTP response status.

This design has some nice properties. The request and response are symmetric. Streaming fits naturally because both sides exchange framed messages. The RPC status is separate from the HTTP status. An HTTP `200 OK` can mean "the HTTP stream worked," while `grpc-status: 5` can mean "the application returned NOT_FOUND."

That model makes sense if you prioritize RPC purity. It makes much less sense if you look at how HTTP actually operates in the wild.

Most HTTP tooling does not think in trailers. If a resource is not found, it expects a `404`. If the caller is not authenticated, it looks for a `401` or `403`. If the server crashes, it needs a `500`. If the request body is too large, it expects a `413`. Load balancers, dashboards, browser dev tools, synthetic checks, reverse proxies, API gateways, and tired engineers at 2 AM all speak this language.

gRPC decided to speak a different language.

That choice bought a clean RPC abstraction, but it also made basic HTTP tooling worse. A normal `curl` request cannot easily hit a standard gRPC endpoint. Generic API clients do not understand the framing. For application-level failures, a load balancer looking only at HTTP status can see successful `200 OK` responses while the actual RPC status lives somewhere else.

You can solve all this, sure. The solution is usually adding more gRPC-aware tools.

The implicit bargain was simple: accept this specialized protocol because the backend benefits are worth the hassle. For backend-to-backend APIs? Absolutely. For the web? Not a chance.

## Then browsers happened

The browser problem was not subtle. Native gRPC relied on HTTP/2 behavior that frontend JavaScript simply could not access. The biggest offender was trailers, but the broader issue was that browsers do not expose the full HTTP/2 model to application code.

The result was awkward: browsers could open HTTP/2 connections, but JavaScript could not run normal gRPC calls over them. That is a pretty rough place for an API framework to land.

gRPC-Web was the official patch. The [gRPC-Web protocol](https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md) adjusted the wire format so browser clients could participate. Instead of relying on HTTP/2 trailers in the standard way, gRPC-Web encodes trailer-like information into the response body. It also supports HTTP/1.1, which was a practical concession to the infrastructure people actually run.

The goal made sense: let frontend code call services generated from the same `.proto` contracts the backend uses. Stop inventing a separate REST or JSON API just because the UI runs in JavaScript. I totally agree with that goal.

The mistake was treating gRPC-Web as another protocol variant instead of stopping to ask whether the original protocol had painted itself into a corner.

## The proxy-shaped warning sign

Historically, the standard gRPC-Web deployment story was proxy-shaped. A browser client would speak gRPC-Web to something like Envoy, and Envoy would translate that into native gRPC for the backend.

If you wanted a more traditional HTTP or JSON shape, the usual answer was another translation layer, like [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway).

It works, but the pitch is exhausting:

> We use gRPC because it gives us clean API contracts.
>
> But browsers cannot call it.
>
> So we use gRPC-Web.
>
> But our gRPC server may not speak that natively.
>
> So we put a proxy in front of it.
>
> And now the browser can call something that sort of resembles gRPC.

By the time you finish that explanation, your clean generated API contract is weighed down by a protocol variant, a proxy-shaped deployment model, and maybe a transcoder.

This is the part that gets to me. gRPC already had a reputation for requiring special handling. Making the official browser path feel like another moving part only made that reputation worse.

To be fair, some server frameworks eventually learned to speak gRPC-Web directly. That is good. It is also the giveaway.

gRPC-Web really was not that different from gRPC. It could have been treated as a first-class protocol mode much earlier. A browser-compatible RPC protocol should not have needed a translation layer as the default story.

That should have been the baseline from day one.

## What gRPC-Web should have become

Imagine if the gRPC team looked at the browser problem and asked a slightly different question:

> If the protocol is awkward for browsers, maybe the browser is not the only problem.

gRPC-Web solved reachability. Browser clients could finally play along, but it stopped short.

The better move would have been:

1. Use real HTTP status codes for unary RPCs.
2. Use standard HTTP compression headers.
3. Use `Content-Length` for unary responses when the length is known.
4. Make unary JSON and protobuf requests easy to send with standard HTTP tools.
5. Keep streaming as a specialized mode instead of making every call pay the complexity tax.
6. Make browser compatibility a baseline feature, not a quirky variant.

In short, gRPC-Web should have been the point where gRPC cleaned up its HTTP story.

Call it gRPC over HTTP, gRPC 2.0, or whatever else. The name is not the important part. The important part is letting HTTP do the job when HTTP is already enough.

Most RPCs are unary anyway. A client sends a request and gets a response. HTTP already handles that beautifully:

```http
POST /acme.user.v1.UserService/GetUser HTTP/1.1
Content-Type: application/json
Accept: application/json
Content-Length: 16
Accept-Encoding: gzip

{"userId":"123"}
```

If the user does not exist, return a `404`. If the request is invalid, send a `400` or `422`. Authentication failed? `401`. If the response is compressed, declare it with `Content-Encoding`. If the body size is known, use `Content-Length`. If the method succeeds, return a `200`.

You can still embed rich error details in the response body. You can still use protobuf. You can still generate clients. You can still have deadlines, metadata, interceptors, and typed service definitions. None of that requires pretending HTTP status codes are beneath you.

That is the core issue for me: building a web-friendly gRPC did not require abandoning gRPC's best ideas. It just required being less stubborn about the envelope.

## What using it would feel like

In this alternate reality, you still define an API like this:

```protobuf
service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}
```

You still generate a Go server and a TypeScript client. You still get typed requests and responses. You still get protobuf on the wire for compact binary encoding.

But you also get to do the boring thing:

```sh
curl \
  -H 'Content-Type: application/json' \
  -d '{"userId":"123"}' \
  https://api.example.com/acme.user.v1.UserService/GetUser
```

If the user exists, you get a `200`. If they do not, you get a `404`. If your request is malformed, you get a `400`. And if the server crashes, your load balancer, browser dev tools, API gateway, and monitoring stack all recognize a `500` without needing a decoder ring for a protobuf-specific status trailer.

It is still an RPC framework. You are still treating `.proto` as the source of truth. You still get the generated code. But your first layer of debugging is just HTTP.

That matters more than protocol purists like to admit.

## The tooling tax

One of the best tests for an API protocol is painfully simple:

Can I make a request with the tools already installed on my machine?

With REST-ish JSON APIs, the answer is almost always yes. The API might be poorly designed, or the auth might be annoying, but the basic mechanics are familiar:

```sh
curl \
  -H 'Content-Type: application/json' \
  -d '{"id":"123"}' \
  https://api.example.com/users/get
```

With native gRPC, you generally need specialized tools like `grpcurl`, `buf curl`, generated clients, reflection, descriptor sets, or local `.proto` files. These tools are fantastic. I use them constantly. But needing them for basic inspection is a real tax.

gRPC-Web does not really fix this. It makes browser calls possible, which is important, but the request still is not quite normal HTTP. The response may contain framed data. The status may be buried in the body. If you are debugging in the browser, the Network tab gets you part of the way there, then leaves you hanging right when things get interesting.

That is a weird outcome for a tool with "Web" in its name.

The web is not always elegant, but it is deeply inspectable. You can read requests in dev tools, replay them with `curl`, route them through proxies, and troubleshoot a lot before reaching for protocol-specific gear. gRPC-Web should have leaned hard into that.

Instead, it usually feels like gRPC wearing a browser costume.

## Streaming made the simple case worse

The strongest defense for native gRPC's design is streaming. Once you factor in client streaming, server streaming, and bidirectional streaming, the custom framing and trailer-based status model makes more sense. You need to know when a stream ends. You may need final status after a sequence of messages. You need a model that works across every RPC shape.

Fair enough.

But that logic allowed the hardest edge cases to dictate the most common case.

Most API calls are not bidirectional streams. They are boring unary operations: create a thing, fetch a thing, update a thing, search for things. These calls should not be taxed with the full complexity of a streaming protocol.

This is where gRPC could have split the model:

* Unary calls use standard HTTP semantics.
* Server streaming uses a stream-friendly response format.
* Client and bidirectional streaming use HTTP/2, WebTransport, or whatever transport provides the necessary primitives.

Would that have been less pure? Yes. Would it have been more honest? Absolutely.

Instead, gRPC forced one model onto every shape, leaving gRPC-Web to patch over the browser mismatch without questioning the original architectural tradeoff.

## The reveal

This whole concept sounds hypothetical, but it already exists.

It is called [ConnectRPC](https://connectrpc.com/).

Connect keeps the protobuf service model, generated clients and servers, and compatibility with gRPC concepts. But for unary calls, it acts like normal HTTP. JSON works. Protobuf works. Status codes mean something. Compression uses standard headers. `curl` works. Browser clients work.

Connect does not pretend streaming is free either. Streaming still needs framing, because streams are streams. The difference is that unary calls are not forced to cosplay as streaming calls just because the protocol wants one perfect model for everything.

Crucially, Connect also does not force you to torch the old world. A Connect server can expose the Connect protocol, gRPC, and gRPC-Web from the same handlers. That is why it does not feel like a competitor to gRPC as much as the evolution path we always deserved.

The thing I wanted gRPC-Web to become is essentially the blueprint Connect followed:

* Web-compatible by default.
* Unary calls that behave like standard HTTP.
* Protobuf schemas as the source of truth.
* Generated clients and servers.
* JSON for easy debugging.
* Binary protobuf for compact encoding.
* Compatibility with gRPC and gRPC-Web for existing clients.
* No mandatory proxy just to let a browser say hello.

I am not saying every existing gRPC deployment should be ripped out overnight. Native gRPC is deeply entrenched, and it handles backend systems beautifully. Connect is not flawless either, and not every API needs to be public-web-friendly.

My point is narrower: the official gRPC ecosystem stumbled onto the right ingredients with gRPC-Web, then locked them in the wrong box.

gRPC-Web was treated as "gRPC, but compromised for browsers."

It should have been treated as "gRPC, corrected for the web."

## What Google could have done

The roadmap did not need to be complicated. Keep native gRPC for the places where it shines: internal systems, high-performance service-to-service traffic, and streaming-heavy APIs. Then make the web-friendly version the boring default for unary APIs.

That would have meant:

1. Official servers speak it directly.
2. Official clients generate support for it.
3. Unary calls use normal HTTP status codes.
4. Compression uses normal HTTP headers.
5. JSON and protobuf both work without ceremony.
6. Browsers are first-class clients, not a weird special case hanging off the side.

That would have changed how gRPC felt to use. Instead of "great for microservices, annoying everywhere else," it could have become the obvious choice for protobuf-defined APIs across backend services, CLIs, mobile apps, and browsers.

That perception matters. Developers do not pick protocols strictly because of throughput benchmarks. They pick them because the first hour is not miserable, debugging makes sense, and deployment does not require a whiteboard diagram before lunch.

gRPC had the hard parts. It had schemas. It had code generation. It had a cross-language ecosystem. It even identified the browser problem correctly. Then, when the web forced the protocol toward a simpler HTTP shape, the ecosystem treated that shape like a side quest.

That was the missed opportunity.

## A browser variant was not enough

I do not think gRPC-Web is a bad project. It was a practical response to real browser limitations, and it unlocked protobuf-based browser clients for countless teams. It deserves credit for that.

But it also dragged along too much of the original protocol's baggage. You still need specialized clients. The default deployment story often involved a proxy. You still miss out on the clean ergonomics of standard HTTP. And because gRPC-Web was framed as a fallback browser variant, it never had the mandate to simplify gRPC itself.

That is why the name "gRPC-Web" has always felt slightly backwards to me. The web should not have been an add-on to gRPC. Browser support should have forced gRPC to become simpler, more inspectable, and far more compatible with the infrastructure the world already runs on.

The best version of gRPC-Web would not have been a bridge back to gRPC; it would have been a web-native replacement.

If that model sounds like what you actually want to build, you should take a serious look at [ConnectRPC](https://connectrpc.com/). That is basically the shape: protobuf-defined APIs, generated clients and servers, ordinary HTTP for unary calls, and compatibility with gRPC and gRPC-Web when you still need the old protocols.
