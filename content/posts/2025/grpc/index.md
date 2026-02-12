---
categories: ["article"]
tags: ["grpc", "protobuf", "api", "webdev"]
date: "2025-05-11T10:00:00Z"
description: "Protovalidate, HTTP/3, Head-of-Line blocking, and why strict schemas save sanity."
cover: "cover.jpg"
images: ["/posts/grpc/cover.jpg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "A Deep Dive into gRPC and Protobuf"
slug: "grpc"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc/
draft: true
---

{{< toc >}}

## **I. Introduction: What Fresh Hell is gRPC?**

If you've built APIs long enough, you've felt this tension:

- JSON is flexible… until it isn’t.
- REST is simple… until versioning and performance get ugly.
- Microservices are clean… until they need to talk to each other 10,000 times per second.

gRPC is a contract-first RPC framework built on HTTP/2 and Protocol Buffers.

It is not:
- “Just faster REST”
- “JSON but binary”
- “Only for Google-scale systems”

It *is*:
- A schema-driven API system
- A code generator
- A high-performance transport layer
- A governance mechanism for distributed systems

Born from Google’s internal RPC system (Stubby) and later open-sourced under the CNCF, gRPC was designed for environments where services evolve in lockstep and performance matters.

---

## **II. The Mental Model: Contract → Code → Transport**

Instead of starting with transport details, start with the real flow:

1. You define a `.proto` schema.
2. Code is generated for clients and servers.
3. The client calls a method like a local function.
4. The framework serializes the message.
5. It travels over HTTP/2.
6. The server deserializes and executes.
7. A response returns the same way.

Under the hood:

```d2
direction: right

Client -> GeneratedStub: Call SayHello()
GeneratedStub -> HTTP2: Serialize + frame
HTTP2 -> ServerTransport: Stream over connection
ServerTransport -> GeneratedServer: Deserialize
GeneratedServer -> Implementation: Invoke handler
Implementation -> GeneratedServer: Return response
GeneratedServer -> HTTP2: Serialize + trailer
HTTP2 -> GeneratedStub: Receive stream
GeneratedStub -> Client: Return result
```

Notice what’s missing:
- No manual routing
- No JSON string parsing
- No reflection-based field lookups

That’s deliberate.

---

## **III. Protocol Buffers: The Real Star**

Most people think they’re adopting gRPC.

They’re actually adopting **Protocol Buffers**.

Protobuf is:
- A schema language (IDL)
- A binary wire format
- A cross-language contract system

```protobuf
syntax = "proto3";

package greet.v1;

service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply);
}

message HelloRequest {
  string name = 1;
}

message HelloReply {
  string message = 1;
}
```

From this single file you generate:
- Go server interfaces
- TypeScript clients
- Java stubs
- Python models
- Even OpenAPI specs

One file. Multiple languages. Guaranteed alignment.

That’s not serialization. That’s governance.

---

### **A. Field Numbers and Binary Efficiency**

Protobuf does not transmit field names — only numeric tags.

```protobuf
string name = 1;
```

On the wire, only `1` matters.

This is why:

- You must never reuse field numbers.
- You must treat schema evolution carefully.
- You need tooling to enforce discipline.

---

### **B. Protovalidate: Validation Done Right**

Validation used to mean writing manual checks everywhere.

Now you can embed constraints directly in your schema using Protovalidate:

```protobuf
import "buf/validate/validate.proto";

message CreateUserRequest {
  string email = 1 [(buf.validate.field).string.email = true];
  int32 age = 2 [(buf.validate.field).int32.gte = 18];
}
```

Now:
- Validation rules live with the contract.
- They apply across languages.
- You stop duplicating logic.

---

## **IV. Tooling: The Difference Between Pain and Discipline**

Using raw `protoc` directly is fragile.

Modern teams use **Buf**.

- `buf lint` → Enforce style consistency.
- `buf breaking` → Detect breaking changes.
- `buf generate` → Declarative plugin execution.

If you care about API stability, `buf breaking` alone justifies the switch.

### **A. The Buf Schema Registry (BSR)**

Think of the BSR as "npm for Protobuf."

Instead of copying `.proto` files between repositories or using fragile Git submodules, you push your schemas to a central registry.

Clients can then:
- Depend on specific versions of your API.
- Generate code remotely without local dependencies.
- Browse documentation that is always in sync with the schema.

It turns API definitions into a managed dependency, just like any other library.

---

## **V. The Transport Layer: HTTP/2 and Performance**

gRPC runs over HTTP/2.

HTTP/2 provides:

- Multiplexed streams
- Header compression
- One TCP connection
- Reduced handshake overhead

Instead of opening multiple TCP connections, you open one and multiplex many RPC calls over it.

This reduces latency and improves throughput significantly in high-volume environments.

---

### **A. The TCP Head-of-Line Blocking Problem**

HTTP/2 multiplexes streams.

But they still share one TCP connection.

If a packet drops:
- The OS pauses the entire TCP connection.
- Every stream waits.
- Everything stalls.

This is TCP-level head-of-line blocking.

HTTP/3 (QUIC over UDP) fixes this by isolating streams at the transport layer.

If you operate over unreliable networks, this distinction matters.

---

## **VI. Streaming: Moving away from one request/one response

Most APIs are Unary (one request, one response). gRPC supports three more patterns thanks to HTTP/2's long-lived streams:

1. **Server Streaming:** Client sends one request; server sends many responses (e.g., a live stock ticker).
2. **Client Streaming:** Client sends many requests; server sends one response (e.g., uploading a large file in chunks).
3. **Bidirectional Streaming:** Both send many messages whenever they want (e.g., a chat application).

Streaming is powerful but adds complexity to load balancing and error handling. Only use it when a single request/response doesn't fit the data flow.

---

## **VII. Observability: The Status Code Lie**

If you come from REST, you trust HTTP status codes.

With gRPC, that’s dangerous.

gRPC often returns:

- HTTP Status: `200 OK`
- Actual Result: `grpc-status: 13 (INTERNAL)`

Why?

Because HTTP is just the transport.  
The real status is in the trailer.

If your monitoring only checks HTTP codes, your dashboards may lie to you.

You must:
- Parse `grpc-status`
- Instrument at the RPC layer
- Use OpenTelemetry interceptors

### **A. Metadata: The Sidecar for Data**

How do you pass Auth tokens, Request IDs, or Trace Context without polluting your Protobuf messages?

**Metadata.**

Metadata (known as "headers" and "trailers" in HTTP) allows you to pass key-value pairs outside the actual message body.

- **Headers:** Sent at the start of the call.
- **Trailers:** Sent at the very end (often containing the final status).

Interceptors (middleware) are the best way to handle metadata, ensuring that every call automatically carries the necessary context for security and tracing.

---

## **VII. The Browser Gap and gRPC-Web**

Browsers cannot directly speak raw gRPC because:

- They do not expose HTTP/2 framing.
- Trailer access is restricted.
- Binary framing conflicts with fetch/XHR constraints.

gRPC-Web solves this via protocol translation and usually requires a proxy (like Envoy).

It works — but it adds infrastructure.

---

## **VIII. ConnectRPC: The Practical Evolution**

ConnectRPC, built by the Buf team, fixes many usability issues.

It supports:

- gRPC
- gRPC-Web
- Connect protocol

From a single server implementation.

Advantages:

### **A. Real HTTP Status Codes**
Errors map to actual HTTP 4xx/5xx responses.

Your observability stack works normally.

### **B. JSON Without gRPC Framing**
You can test endpoints with plain curl:

```bash
curl [https://api.example.com/greet.v1.Greeter/SayHello](https://api.example.com/greet.v1.Greeter/SayHello) \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice"}'
```

No special client required.

### **C. One Implementation, Multiple Protocols**
You don’t need separate infrastructure for browsers and services.

For many teams, Connect is what they hoped gRPC would feel like.

---

## **IX. Generating OpenAPI from Protobuf**

One common objection:

> “Our frontend uses OpenAPI.”

You can still keep Protobuf as your source of truth.

Tools like:

- https://github.com/sudorandom/protoc-gen-connect-openapi

allow you to generate OpenAPI specs directly from Connect/Protobuf schemas.

That means:
- No duplicate API definitions.
- No schema drift.
- No maintaining REST by hand.

One contract. Multiple outputs.

---

## **X. Mocking and Contract-First Development**

Testing often becomes painful in RPC systems.

Instead of spinning up real backends, you can mock directly from schema.

- https://github.com/sudorandom/fauxrpc
- https://fauxrpc.com/

FauxRPC enables:

- Schema-driven mocks
- Parallel frontend/backend development
- Rapid prototyping before implementation

If you embrace contract-first design, mocking from `.proto` is the logical next step.

---

## **XI. Alternatives and Tradeoffs**

gRPC is not religion.

### Twirp
- Simpler RPC
- Protobuf-based
- Minimalist
- Less infrastructure complexity

### REST + JSON
- Universally compatible
- Easy debugging
- Weak contracts
- Larger payloads

gRPC shines when:
- You control clients
- You need high throughput
- You value strict contracts
- You operate in microservices

It’s overkill when:
- You need maximum public compatibility
- You can’t enforce schema discipline
- Your infra struggles with HTTP/2

---

## **XII. Conclusion**

gRPC is not a silver bullet.

It adds:
- Tooling complexity
- Binary debugging friction
- HTTP/2 infrastructure requirements

But when you combine:

- The speed of multiplexed streams
- The safety of schema validation
- The guardrails of Buf
- The ergonomics of Connect

You get a system that scales better than loose JSON-over-HTTP architectures.

The real power isn’t the binary encoding.

It’s the enforceable contract.

In distributed systems, contracts are everything.

And if you adopt them properly, you stop debugging stringly-typed chaos at 2AM.

Just remember to check your trailers.