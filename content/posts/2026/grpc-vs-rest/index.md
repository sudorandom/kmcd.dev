---
categories: ["article"]
tags: ["api", "grpc", "protobuf", "rest", "restful"]
date: "2026-09-21T10:00:00Z"
description: "The traditional advice is REST for public APIs and gRPC for microservices. But is this a false dichotomy?"
cover: "cover.svg"
images: ["/posts/grpc-vs-rest/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC vs REST Is the Wrong Question"
slug: "grpc-vs-rest"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-vs-rest/
---

I've seen the same argument over and over on Twitter, LinkedIn, Reddit, etc. Where should you use JSON/HTTP APIs and where should you use gRPC? The same exact advice is almost always given: use HTTP/JSON for public or browser-facing APIs, and gRPC for microservices and infrastructure such as etcd and containerd.

The reasons for this advice are valid: HTTP/JSON *is* trivial to set up, inspect in DevTools, cache, proxy, and hit from a browser. gRPC and Protobuf *do* give you schemas, generated clients, compact and fast binary payloads, and streaming.

Why can't we have the best of both?

{{< image src="rest-vs-grpc-both.png" class="center" >}}

You <u>*can*</u> get the best of both worlds. With [ConnectRPC](https://connectrpc.com).

*(Disclosure: I work at Buf, which maintains ConnectRPC, but I absolutely had all of these opinions beforehand.)*

---

## Browser support

gRPC relies on HTTP/2 features that browser APIs refuse to expose to JavaScript, including [trailers](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Trailer). So gRPC can't be used with web browsers.

[gRPC-Web](https://github.com/grpc/grpc-web) tried to bridge the gap by putting trailers as another frame type inside the body of the HTTP responses. gRPC-Web ultimately failed as a project. Its own roadmap says it can no longer deliver new modern solutions, will not add new features, and recommends gRPC-Gateway instead. I cover the reasons in detail in [gRPC-Web Failed the Web](https://buf.build/blog/grpc-web-failed-the-web).

Transcoding is the other common workaround that enables you to have gRPC in the backend and an HTTP/JSON API in the frontend leveraging tools like [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway) and [Envoy](https://www.envoyproxy.io/). You annotate Protobuf files with `google.api.http` rules, and the proxy translates inbound requests to gRPC and the outgoing responses back to HTTP/JSON. This works fine, but now the browser-facing API goes through another representation. The Protobuf schema defines the gRPC service, `google.api.http` defines how that service maps onto HTTP, and tooling may then generate an OpenAPI description and another client from that.

{{% columns %}}
<div>

```d2
direction: down

"Transcoding via Proxy": {
  "Browser Client" -> "gRPC-Gateway / Envoy": REST / JSON
  "gRPC-Gateway / Envoy" -> "Your Backend Service": gRPC / Proto
}
```

</div>
<div style="align-self: center;">

- An extra required hop between the browser and the gRPC backend.
- Another HTTP mapping to define and maintain alongside the RPC service.
- Browser clients no longer talk directly to the RPC service described by the Protobuf schema.

</div>
{{% /columns %}}

### The Connect response

Connect sidesteps the proxy layer entirely by having the application server handle three protocols natively: standard gRPC, gRPC-Web, and Connect's own [HTTP-based protocol](https://connectrpc.com/docs/protocol/). Existing gRPC clients can keep talking standard gRPC, while browsers talk directly to the application server over plain HTTP with JSON or binary Protobuf.

```d2
direction: down

"Direct Multi-Protocol (ConnectRPC)": {
  "Browser Client" -> "Your Backend Service": Connect / JSON
  "Random curl script" -> "Your Backend Service": Connect / JSON
  "Go Client" -> "Your Backend Service": gRPC / Proto
  "Existing gRPC-Web Implementation" -> "Your Backend Service": gRPC-Web / Proto
}
```

Connect-Web also supports server-streaming RPCs right in the browser using the Fetch API. Outside the browser, the protocol handles client and bidirectional streaming over HTTP/2.

## Ad-hoc requests

A dead simple requirement for any API is the "send a coworker a curl command" test. This should be trivial. Standard gRPC fails this spectacularly. gRPC-Web also fails it.

Generated SDKs are actually pretty awesome, but they can get in the way when you want to quickly reproduce an issue or verify a change from the terminal. With gRPC, you have to install [grpcurl](https://github.com/fullstorydev/grpcurl), verify reflection is enabled in that environment, or manually pass proto files before calling an API dynamically.

### The Connect way

A unary Connect request is just an HTTP POST. For quick debugging, you can use [ProtoJSON](https://protobuf.dev/programming-guides/json/):

```http
Content-Type: application/json
```

That makes terminal testing straightforward:

```bash
curl -X POST \
  https://demo.connectrpc.com/connectrpc.eliza.v1.ElizaService/Say \
  -H "Connect-Protocol-Version: 1" \
  -H "Content-Type: application/json" \
  -d '{"sentence":"Hello"}'
```

No special CLIs, no need for server reflection. The server handles JSON deserialization according to the Protobuf schema, and the response prints clean, readable ProtoJSON. This example actually works, by the way. You can copy/paste it into your terminal and try.

Errors behave predictably too. Like many RPC systems, standard gRPC returns an HTTP `200 OK` for every response and puts the actual failures away in `grpc-status` trailers, which means edge proxies and access logs often report failed calls as perfectly healthy traffic. Connect uses standard HTTP statuses for unary calls: `NotFound` maps to `404`, `InvalidArgument` maps to `400`, and existing observability tools see unary call failures without custom parsers.

The same pragmatism applies to HTTP verbs. gRPC mandates POST for everything, even simple lookups or queries. Connect supports GET requests for methods flagged as [side-effect free](https://connectrpc.com/docs/go/get-requests-and-caching/):

```protobuf
rpc Say(SayRequest) returns (SayResponse) {
  option idempotency_level = NO_SIDE_EFFECTS;
}
```

Connect clients can call these endpoints over GET by placing serialized parameters into the query string:

```bash
curl "https://demo.connectrpc.com/connectrpc.eliza.v1.ElizaService/Say?message=%7B%22sentence%22%3A%22Hello%22%7D&encoding=json&connect=v1"
```

The payload lives in the URL (as JSON or base64-encoded Protobuf), which means standard web caching actually works. Return a standard [`Cache-Control`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control) header, and any standard CDN or browser cache will respect it.

You also aren't locked into Protobuf on the client side. Tools like [`protoc-gen-connect-openapi`](https://github.com/sudorandom/protoc-gen-connect-openapi), which I created, can generate an OpenAPI spec from your proto definitions. If another team needs OpenAPI specs for their tooling, you can export them without rebuilding your backend architecture around them.

## Use ConnectRPC

The usual REST vs gRPC discourse assumes you have to choose between ordinary HTTP semantics and a schema-driven RPC API. Connect doesn't make you choose.

Instead of asking whether to use REST or gRPC, why not ConnectRPC?
