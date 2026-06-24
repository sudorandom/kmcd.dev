---
title: "Introducing ProtoDocs"
date: "2026-06-23T10:00:00Z"
categories: ["article", "project"]
tags: ["protobuf", "grpc", "connectrpc", "documentation", "go", "rpc"]
description: "A protobuf-first documentation browser for APIs that need clearer generated docs."
cover: "cover.svg"
images: ["/posts/introducing-protodocs/cover.png"]
featuredalt: "A stylized ProtoDocs interface showing protobuf files, services, messages, and RPC calls."
featuredpath: "date"
slug: "introducing-protodocs"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/introducing-protodocs/
---

**Protocol Buffers are fantastic for defining API contracts.** You get a mature tooling ecosystem, strong compatibility habits, and schemas that are precise enough to build around. The documentation story is not nearly as good. If you have ever tried to hand generated protobuf docs to an API consumer, you have probably seen the gap.

Most tools force an awkward compromise. Some produce complete output that is hard to read. Others look polished only after custom templates, glue code, and project-specific maintenance. The fallback is to share the `.proto` files directly, which works for people already deep in protobuf but is rough for everyone else.

That frustration is why I originally built [protoc-gen-connect-openapi](https://github.com/sudorandom/protoc-gen-connect-openapi). The OpenAPI ecosystem has tools like Redocly and Swagger UI, and they make good-looking, browsable API sites easy to publish. Translating [Connect APIs](https://connectrpc.com) into OpenAPI made sense at the time, and it still works well. But it also means starting with protobuf, converting to another schema format, and then documenting that result.

I wanted the shorter path: start from protobuf, keep the schema visible, and still end up with something polished enough to publish.

So I made **[ProtoDocs](https://protodocs.dev/)**.

{{< figure src="eliza-sshot.png" caption="A screenshot of the ProtoDocs interface showing protobuf files, services, messages, and RPC calls.">}}

## Protobuf-first documentation

ProtoDocs is a web-based documentation browser for Protocol Buffer definitions. Its model follows protobuf directly: files, packages, services, RPCs, messages, enums, fields, oneofs, options, and comments.

What does "protobuf-first" actually mean? It means the tool treats the source as the primary interface. I think protobuf schemas are clean, expressive, and readable, especially compared with the nested JSON or YAML you often end up reading in OpenAPI. ProtoDocs renders the actual protobuf syntax, adds highlighting, and layers interactive tooltips over keywords, field names, type references, and custom options. You can read the source and explore it at the same time.

ProtoDocs also defines its own configuration as a Protocol Buffer schema, so the project can document its own config format. You can browse the official configuration options directly on [protodocs.dev](https://protodocs.dev/#/files/protodocs/v1/config.proto).

The goal is simple: make protobuf documentation readable, navigable, and useful without first turning it into a different API description format.

## What it does

The UI lets you explore an API from a few different angles:

*   **Browse files and packages** in a directory hierarchy that merges common path prefixes, which makes larger proto trees easier to scan.
*   **Inspect services and RPC methods**, including unary and streaming signatures, alongside custom method options.
*   **Drill into messages**, enums, custom options, oneof declarations, nested types, comments, and linked type references.
*   **Jump to definitions** and find references across the loaded files. Pinned type tooltips let you move from a field to its type declaration, or see where a type is used, without losing your place.
*   **Send live requests** from an interactive "Try it out" panel using Connect, [gRPC-Web](https://github.com/grpc/grpc-web), or proxied [gRPC](https://grpc.io), including streaming calls.
*   **Tunnel browser traffic** through a WebSocket-based proxy to avoid CORS restrictions and connect directly to services.
*   **Read highlighted protobuf source** with keyword tooltips that explain the underlying syntax.
*   **Load schemas several ways**, including pre-built descriptor sets from `buf` or `protoc`, live server reflection, or an in-memory Go registry for embedded usage.

As you edit request parameters in the client panel, ProtoDocs generates command snippets for Connect, `grpcurl`, and `buf curl`. The browser interaction becomes a reproducible terminal command instead of a one-off experiment.

## Built for Connect, but not only Connect

ProtoDocs is designed with ConnectRPC servers in mind.

A Connect server can expose Connect, gRPC, and gRPC-Web from the same API surface, which makes it a natural fit for browser-based exploration. ProtoDocs uses that shape to keep the interactive client close to the real RPC contract.

The interactive client supports:

*   Connect
*   gRPC-Web
*   gRPC through proxy mode

The documentation side only needs protobuf descriptors, so schema browsing works beyond Connect. The richer interactive path is where ConnectRPC servers shine.

{{< figure src="eliza-tryitout-sshot.png" caption="A screenshot of the ProtoDocs request panel for trying RPC calls from the browser.">}}

## Three ways to run it

There are three primary deployment modes:

- Static website
- Website with a backend
- Embedded Go application

Each one has a different tradeoff.

### Static website

For the static path, put the built frontend on Nginx, S3, Netlify, GitHub Pages, or any other static host, then point it at one or more descriptor files. That is enough for browsable protobuf documentation.

This is the simplest deployment. For interactive requests, the browser talks directly to the target API server, so that server needs CORS configured if you want the "Try it out!" panel to work.

{{< d2 width="100%" max-height="70vh" >}}
direction: down

ui: ProtoDocs UI

service: gRPC-Web / Connect Endpoint

ui -> service: Direct RPC, requires CORS
{{< /d2 >}}

### Website with a small backend

ProtoDocs can also run behind a small backend proxy.

Here, browser requests and streaming calls tunnel through the proxy. That avoids CORS problems and lets the backend forward requests to gRPC, gRPC-Web, or Connect services.

{{< d2 width="100%" max-height="70vh" >}}
direction: down

ui: ProtoDocs UI

proxy: HTTP and WebSocket Proxy

service: gRPC / Connect / gRPC-Web

ui -> proxy: Proxy RPC and streams, WebSockets
proxy -> service: Translate and forward, HTTP/2
{{< /d2 >}}

I expect this mode to be the best fit for internal tools and developer portals where "open the docs and call the API" needs to work out of the box.

### Embedded Go handler

The Go package can also embed ProtoDocs directly into an HTTP server.

That gives you the documentation UI, descriptor loading from the Go registry, and the proxy path from the same application. If you already have a Go service exposing Connect or gRPC handlers, mounting ProtoDocs alongside it makes the service self-documenting without a separate docs deployment.

At a high level, it looks like this:

```go
handler, err := protodocs.NewHandler(protodocs.Config{
    Title:    "My Service Documentation",
    LogoText: "My Service",
    Registry: protoregistry.GlobalFiles,
    Prefix:   "/docs/",
})
if err != nil {
    log.Fatal(err)
}

http.Handle("/docs/", handler)
```

## Themes, source, and sharp edges

ProtoDocs includes light, dark, and cyberpunk themes. They can follow system preferences or be toggled manually.

It renders source views, markdown comments, structured options, package trees, service lists, search results, and linked type details. The point is to make the schema readable without asking people to clone a repo, find the right proto directory, and mentally resolve every import by hand.

It is still early.

There are bugs, and parts of the UI and runtime behavior will change as the project gets more real-world use.

But it is far enough along that I want people to try it, point out what is confusing, and show me the protobuf schemas that break it. If you run into sharp edges, have feedback on the layout, or want to contribute a fix, please open an issue or send a PR on GitHub.

## Try it

You can try the live demo at **[protodocs.dev](https://protodocs.dev/)**.

{{< github-repo repo="sudorandom/protodocs" description="A protobuf-first documentation browser for APIs that need clearer generated docs." >}}

If you have a ConnectRPC, gRPC-Web, or descriptor-heavy protobuf project, point ProtoDocs at it and see where it falls over.
