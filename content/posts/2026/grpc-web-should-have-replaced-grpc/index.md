---
categories: ["article"]
tags: ["grpc", "grpc-web", "protobuf", "connectrpc"]
date: "2026-07-14T10:00:00Z"
description: "gRPC-Web should have been the pressure that made gRPC simpler, more inspectable, and better suited for the web."
title: "gRPC-Web Should Have Fixed gRPC"
slug: "grpc-web-should-have-fixed-grpc"
type: "posts"
---

gRPC did a lot right. It turned Protocol Buffers into an API design tool, delivering typed messages, generated clients, and strict contracts. For backend networks where you control every hop, it is a fantastic tool.

But gRPC tied itself too tightly to its HTTP/2 transport model. Specifically, it built core application behavior, like delivering final application status codes via response trailers, around protocol features that browser JavaScript could not fully expose.

Browsers support HTTP/2 at the network layer, but frontend JavaScript gets a much narrower API. Standard browser tools like `fetch` and XHR do not expose HTTP trailers to application code in the way native gRPC depends on. This left gRPC in an awkward position: the browser was using HTTP/2 underneath, but JavaScript could not make a native gRPC call.

## The Compromise: gRPC-Web

gRPC-Web was the official answer to this problem. It adjusted the wire format by encoding trailer-like data directly into the response body, allowing browser clients to function.

It worked, but the deployment story usually required a proxy, which added friction and complexity. Because ordinary gRPC servers did not speak this variant natively, you usually had to run Envoy or another gRPC-Web translation layer just to bridge browser requests into your backend.

We accepted a specialized protocol for backend-to-backend efficiency, but bringing gRPC to the web involved extra parts. gRPC-Web was treated as a weird browser variant on the side instead of a mandate to simplify gRPC itself.

## Unary Calls Should Be Boring

The real mistake was letting the hardest engineering cases define the common case. Most RPCs are not bidirectional streams. They are ordinary request-response operations: create a thing, fetch a thing, update a thing.

For these unary calls, the protocol should have used standard HTTP semantics.

That means using HTTP’s existing machinery instead of recreating it inside a custom message envelope. `Content-Type` should say whether the body is JSON or binary protobuf. For successful unary calls, the response can use the same format as the request. Developers should be able to use `application/json` in browsers or local `curl` calls for easy debugging, while production workloads can use a binary protobuf content type for better performance. `Accept-Encoding` should advertise supported compression formats, and `Content-Encoding` should describe the compression actually used. When the body size is known, `Content-Length` can say how large it is. When it is not known ahead of time, the underlying HTTP version already has ways to determine where the body ends.

For unary RPCs, gRPC’s extra length prefix and compression flag were not elegant protocol design. They were streaming machinery leaking into the boring case.

You should still define your schema in protobuf, generate your clients, and get type safety. But the network layer should look like this:

```sh
curl \
  -H 'Content-Type: application/json' \
  -d '{"userId":"123"}' \
  https://api.example.com/UserService/GetUser
```

If the operation naturally maps to an HTTP status, use it. If a user exists, return `200`. If they do not, return `404`. If the server crashes, return `500`.

The standard counterargument from RPC purists is that HTTP status codes are too coarse. A `404` could mean the URL path is missing, or it could mean the requested database resource is missing. To avoid that ambiguity, gRPC often returns `200 OK` for a successfully handled HTTP request and puts the RPC status in trailers.

But treating transport errors and application errors as completely separate worlds asks too much of the web. Load balancers, API gateways, CDNs, browser tools, and monitoring dashboards already understand `4xx` and `5xx` rates. Hiding application failures behind `200 OK` makes that infrastructure less useful unless every layer becomes protocol-aware.

A web-native design should use standard HTTP status codes for coarse outcomes, then put richer domain-specific error details inside the JSON or binary response body.

Boring HTTP works. Browser dev tools understand it, standard middleboxes log it correctly, and tired engineers can debug it at 2 AM without specialized tools.

A practical protocol split would have been straightforward:

* **Unary calls:** Standard HTTP request-response semantics.
* **Streaming calls:** Framed messages over a stream-friendly transport.

## The Web-Native Evolution

What I want from “the next version of gRPC” is not exotic. In fact, it already exists; it is exactly how [ConnectRPC](https://connectrpc.com/) works. Connect preserves the protobuf service model and client generation, but treats unary calls as standard HTTP requests without the complexities of binary framing on top of HTTP. It proves that you can have type-safe contracts without forcing simple calls to cosplay as complex streams.

gRPC-Web should have been the moment gRPC admitted that the web was not a weird edge case. It should have been gRPC v2: protobuf contracts and generated clients on top of boring, inspectable, web-native HTTP, with special framing saved for real streaming instead of forced onto every unary call.

Instead, it became another compatibility layer. Useful, yes. But far less ambitious than it should have been.
