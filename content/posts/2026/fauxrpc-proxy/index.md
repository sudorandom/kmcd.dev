---
categories: ["article"]
tags: ["fauxrpc", "connectrpc", "grpc", "protobuf", "api", "rpc", "go", "golang", "http3"]
date: "2026-06-29T10:00:00Z"
description: "Stop writing mock stubs by hand. How FauxRPC uses smart proxying, reflection, and CEL to automate your API testing."
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Smart Proxying and Auto-Recording in FauxRPC"
slug: "fauxrpc-proxy"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/fauxrpc-proxy/
draft: true
---

Mocking APIs is one of those tasks that starts simple and quickly turns into a chore. You begin with the best intentions, hand-crafting a few JSON fixtures for your frontend tests. But microservices evolve, payloads change, and soon you're maintaining a massive directory of stale mock files. You find yourself trying to remember if user ID `42` was the one that returns a `404`, or the one that simulates a slow response.

When you're building with gRPC or ConnectRPC, this problem gets both easier and harder. It's easier because you have a strict schema (the Protobuf contract) to guide you. It's harder because writing binary payloads or mock servers that conform to those schemas by hand is tedious.

In [FauxRPC](/posts/fauxrpc/), I built a tool to generate fake data from Protobuf schemas dynamically. But I wanted to go further. I wanted to make mocks a natural byproduct of running your development environment. That is why I introduced **Proxy Mode** and **Auto-Recording**.

By placing FauxRPC in front of a real upstream service, it acts as a smart proxy: intercepting traffic, forwarding it to the upstream server, and writing out reusable mock stubs to disk. It even generates intelligent matching rules automatically.

Here is how it works under the hood.

```d2
direction: down

client: Client {
  shape: person
}

fauxrpc: FauxRPC Proxy {
  style.fill: "#1e1e2e"
  style.stroke: "#cba6f7"
}

upstream: Backend Server {
  style.fill: "#11111b"
  style.stroke: "#f38ba8"
}

client -> fauxrpc: "Request" {
  style.stroke: "#89b4fa"
}

# Scenario A: Implemented endpoints
fauxrpc -> upstream: "Forward" {
  style.stroke: "#89b4fa"
}
upstream -> fauxrpc: "Response" {
  style.stroke: "#a6e3a1"
}
fauxrpc -> client: "Return Response" {
  style.stroke: "#a6e3a1"
}

# Scenario B: Unimplemented endpoints (Fallback)
fauxrpc -> client: "Return Fake/Stub Response\n(if backend returns Unimplemented)" {
  style.stroke: "#f9e2af"
}
```

## Zero-Configuration Reflection

Normally, running a mock server requires you to supply the schema files. You have to pass paths to your `.proto` files or compiled descriptor sets (`.binpb`). That’s fine for a CI environment, but during local development, dragging files around and keeping them in sync is a pain.

If you start FauxRPC in proxy mode without specifying a schema:
```bash
fauxrpc run --proxy-to=localhost:8080 --record-dir=stubs/
```
FauxRPC uses **gRPC Server Reflection** to discover the schema dynamically from the upstream server on startup.

Behind the scenes, FauxRPC performs a few key steps during this schema discovery phase:
1. It connects to the upstream and queries the reflection service using `ListServices` to list all endpoints.
2. For each service, it fetches the file descriptors using `FileContainingSymbol`.
3. It recursively collects the raw file descriptors (`FileDescriptorProto`), deduplicating common dependencies (like `google/protobuf/timestamp.proto`).
4. It compiles the gathered descriptors into a `FileDescriptorSet` and registers them in FauxRPC's registry.

This means FauxRPC is ready to handle, translate, and mock any method defined on the upstream server with absolutely zero configuration.

## The Multi-Protocol Proxy Engine

FauxRPC is designed to work as a multi-protocol translator. A frontend might communicate using ConnectRPC (over JSON), while the upstream service is a pure gRPC service using binary Protobuf.

To achieve this, FauxRPC sets up a ConnectRPC client using a custom `dynamicProtoCodec` for proxying. This codec uses Go's protobuf reflection (`dynamicpb.Message`) to encode and decode payloads on the fly using the schemas retrieved during the reflection phase.

* **Unary calls:** The proxy reads the request, forwards it using the client, intercepts the response, and writes it back.
* **Streaming calls (client, server, and bidi):** To handle streams, FauxRPC uses `FrameTracker` objects. The tracker forwards frames in real time so there is no added latency, but it keeps a copy of the sequence in memory for logging and recording.
* **Metadata and Headers:** When forwarding headers via `copyHeaders`, FauxRPC filters out protocol-level headers (like `content-type`, `grpc-status`, and Connect-specific protocol headers) to let the underlying transport layer handle them cleanly. It also automatically masks sensitive headers (e.g. `Authorization`, `Cookie`, `X-API-Key`) with `*****` to avoid leaking production keys or user credentials into logs or stubs.

## Graceful Fallbacks for Parallel Teams

In a fast-moving team, schemas are often updated before the code is ready. A backend engineer might merge a `.proto` change adding a new endpoint, but the actual implementation won't land for days. Normally, this blocks the frontend engineers who need that endpoint to build the UI.

FauxRPC's **Fallback Mode** solves this problem directly.

If the upstream server returns an `Unimplemented` error code, the proxy doesn't just pass that error to the client. Instead, it enters fallback mode:
1. It checks the local stub database to see if a mock stub matches the request.
2. If no stub is found, it automatically generates realistic fake data (leveraging `protovalidate` annotations if they are present in the schema).
3. The fake response is returned to the client.

To the frontend, the endpoint looks and acts like it's fully implemented. The frontend team keeps coding, completely unblocked by backend delays.

## Auto-Recording Stubs

Writing mock stubs by hand is the worst part of API mocking. In proxy mode, FauxRPC can automate this entirely when you pass the `--record-dir` flag:

```
stubs/
└── connectrpc.eliza.v1.ElizaService/
    ├── Say.json
    └── Introduce.json
```

As you interact with your upstream service, FauxRPC intercepts the requests and responses, automatically writing them to disk as structured JSON or YAML files matching your service hierarchy.

Because these files are saved directly in your codebase, you can commit them to Git and immediately use them as your mock suite in CI/CD pipelines or integration tests. There's no extra setup—just run your application, click around your frontend, and your API mock stubs are generated for you.

## Generating Intelligent CEL Matchers

A recorded stub is only useful if it matches the right request parameters. If you search for user ID `12`, you want the stub that returns Alice. If you search for user ID `42`, you want the stub that returns Bob.

FauxRPC automatically compiles request shapes into Common Expression Language (CEL) matching rules by examining the request metadata and payload:
1. It ranges over the fields set on the request message.
2. It ignores complex fields (nested messages, lists, and maps) to keep the generated rules readable.
3. For primitive fields (like strings, booleans, and integers), it formats the values into CEL literals.
4. It joins these field checks with `&&`.

For example, if you send a request where `name` is "Alice" and `age` is 30, FauxRPC automatically generates:
```yaml
active_if: req.name == "Alice" && req.age == 30
```
When a subsequent request comes in, FauxRPC compiles and evaluates this expression against the request payload. If it evaluates to `true`, the stub is served.

## Recorded Stub Formats

FauxRPC records three kinds of stubs based on the call type:

### Unary / Client-Streaming Success
For successful unary calls, it records the response payload:
```json
{
  "id": "c85d8869-ad10-449e-ba63-2287f7401c10",
  "target": "connectrpc.eliza.v1.ElizaService/Say",
  "active_if": "req.sentence == \"hello\"",
  "content": {
    "sentence": "Hello! How can I help you today?"
  },
  "priority": 10
}
```

### Unary / Client-Streaming Error
If the upstream returned an error, FauxRPC records the status code and message so you can mock error states:
```json
{
  "id": "18fd7d9e-108c-4a37-bcfc-fa8d7a12ad4f",
  "target": "connectrpc.eliza.v1.ElizaService/Say",
  "active_if": "req.sentence == \"trigger error\"",
  "error_code": 3,
  "error_message": "invalid sentence structure",
  "priority": 10
}
```

### Server-Streaming / Bidirectional Streaming
For streaming APIs, it records the sequence of frames, mimicking latency with a default delay, and captures trailing errors:
```json
{
  "id": "a97df11b-7a31-4b10-8b4b-6f81e3a1f810",
  "target": "connectrpc.eliza.v1.ElizaService/Introduce",
  "active_if": "req.name == \"Bob\"",
  "stream": {
    "items": [
      { "content": { "sentence": "Hi Bob!" }, "delay": "100ms" },
      { "content": { "sentence": "I am Eliza." }, "delay": "100ms" },
      { "error": { "code": 5, "message": "session lost" } }
    ]
  },
  "priority": 10
}
```

## A Protobuf-Native Dashboard with Protodocs

The developer dashboard in FauxRPC (served at `http://localhost:6660/fauxrpc` when running with `--dashboard`) has also received a major upgrade. 

Previously, I relied on converting Protobuf schemas to OpenAPI and rendering a standard Swagger UI using my plugin, [protoc-gen-connect-openapi](https://github.com/sudorandom/protoc-gen-connect-openapi). It worked, but OpenAPI is fundamentally designed for REST. Translating Protobuf concepts into OpenAPI constructs is like fitting a square peg in a round hole—you lose the rich semantics of your schemas.

To fix this, I built **[Protodocs](https://protodocs.dev/)**, a "protobuf-native" documentation tool, and integrated it directly into FauxRPC.

{{< figure src="protodoc-in-fauxrpc.png" alt="Protodocs rendered natively inside the FauxRPC developer dashboard" >}}

Protodocs takes raw protobuf descriptors and renders a documentation page built specifically for Protobuf. Because it understands the schema natively, it includes features you’d expect from an IDE:
* **Go-to-definition** and **find-references** for messages, fields, and services.
* **Interactive API explorer**: You can make test calls directly from the browser using gRPC, gRPC-Web, or ConnectRPC (similar to Swagger, but with native protocol support).

### Testing gRPC from the Browser

One major hurdle with browser-based gRPC clients is that browsers don't expose HTTP/2 trailers, which native gRPC requires for status codes. 

To solve this, Protodocs uses a websocket-based proxy under the hood when integrated with a Go server like FauxRPC. When you trigger a gRPC call from the browser, the request is channeled through a WebSocket connection to FauxRPC, which acts as a bridge to translate and send native gRPC frames to the backend. You get full browser-based testing for actual gRPC APIs without having to deploy Envoy or configure complex gRPC-Web proxies.

And, of course, the FauxRPC dashboard still captures your live traffic history. When you open a logged request, you can view the raw JSON payload or copy a pre-compiled FauxRPC YAML stub with its generated `active_if` matcher directly to your clipboard. This completely bypasses the manual stub-writing process.

{{< figure src="fauxrpc-request-log-stub.png" alt="Viewing the FauxRPC request log history and copying a pre-generated stub" >}}

## Summary

FauxRPC’s proxy and recording capabilities bridge the gap between static mock servers and real-world backend services. By discovering schemas through reflection, handling dynamic protocol translation, falling back to fake data, and generating intelligent CEL matchers automatically, it takes the busywork out of API mocking.

It turns mocks from a maintenance chore into a natural byproduct of running your development environment.

