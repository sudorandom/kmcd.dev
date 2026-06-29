---
categories: ["article"]
tags: ["grpc", "grpc-web", "protobuf", "connectrpc"]
date: "2026-07-14T10:00:00Z"
description: "gRPC-Web should have been the pressure that made gRPC simpler, more inspectable, and better suited for the web."
title: "gRPC-Web Should Have Fixed gRPC"
slug: "grpc-web-should-have-fixed-grpc"
type: "posts"
draft: true
---

gRPC did a lot right. It turned Protocol Buffers into an API design tool, delivering typed messages, generated clients, and strict contracts. For backend networks where you control every hop, it is a fantastic tool.

But gRPC tied itself too tightly to HTTP/2 internals. Specifically, it relied on response trailers to deliver final application status codes like `grpc-status`.

While browsers support HTTP/2 at the network layer, frontend JavaScript does not get full low-level access. Web apps can benefit from multiplexing, but `fetch` and XHR expose a much narrower API that historically lacked trailer support.

That left gRPC in an awkward position: the browser could use HTTP/2 underneath, but JavaScript could not make a native gRPC call.

## The Compromise: gRPC-Web

gRPC-Web was the official patch. It adjusted the wire format by encoding trailer-like data directly into the response body, allowing browser clients to participate.

It worked, but the deployment story became a proxy nightmare. Because standard backend gRPC servers did not speak this variant natively, you had to run Envoy or a transcoder like gRPC-Gateway just to translate browser requests.

We accepted a specialized protocol for backend-to-backend efficiency, but bringing that baggage to the web created a mountain of moving parts. gRPC-Web was treated as a fallback browser variant instead of a mandate to simplify gRPC itself.

## Unary Calls Should Be Boring

The real mistake was letting the hardest engineering cases define the common case. Most RPCs are not bidirectional streams. They are ordinary request-response operations: create a thing, fetch a thing, update a thing.

For these unary calls, the protocol should have used standard HTTP semantics.

You should still define your schema in protobuf, generate your clients, and get type safety. But the network layer should look like this:

```sh
curl \
  -H 'Content-Type: application/json' \
  -d '{"userId":"123"}' \
  [https://api.example.com/UserService/GetUser](https://api.example.com/UserService/GetUser)

```

If a user exists, return an HTTP `200`. If they do not, return a `404`. If the server crashes, return a `500`.

Boring HTTP works. Browser dev tools understand it, load balancers log it correctly, and tired engineers can debug it at 2 AM without specialized tools.

A practical protocol split would have been straightforward:

* **Unary calls:** Standard HTTP request-response semantics.
* **Streaming calls:** Framed messages over a stream-friendly transport.

## The Web-Native Evolution

This protocol isn't hypothetical; it is exactly how [ConnectRPC](https://connectrpc.com/) works. Connect preserves the protobuf service model and client generation, but treats unary calls as standard HTTP requests. It proves that you can have type-safe contracts without forcing simple calls to cosplay as complex streams.

gRPC-Web had the right ingredients but locked them in the wrong box. It was framed as gRPC compromised for the browser, when it should have been treated as gRPC corrected for the web.
