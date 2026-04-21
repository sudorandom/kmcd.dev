---
categories: ["article"]
tags: ["connectrpc", "grpc", "protobuf", "api", "rpc", "go", "golang", "http3", "openapi"]
date: "2026-05-05T10:00:00Z"
description: "Reflecting on two years of ConnectRPC: How it evolved from a gRPC alternative to a complete API ecosystem."
cover: "cover.svg"
images: ["/posts/connectrpc-where-is-it-now/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "ConnectRPC: Where is it now?"
slug: "connectrpc-where-is-it-now"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/connectrpc-where-is-it-now/
---

Two years ago, I wrote [Making gRPC more approachable with ConnectRPC](/posts/connectrpc/). At the time, ConnectRPC was the "new kid on the block", a library promising to fix the "gRPC tax" by supporting HTTP/1.1 and JSON without an extra proxy.

Today, ConnectRPC isn't just a library. It is the core of a toolchain that makes traditional `protoc` workflows look completely dated. Companies like Anthropic are using it in production to power their SDKs, even maintaining [their own ConnectRPC library in Rust](https://github.com/anthropics/connect-rust). 

Let's look at how far things have come and how tools like Buf Remote Plugins, Protobuf SDKs, FauxRPC, and native HTTP/3 are changing API development.

## Code Generation

One of my biggest complaints in [Working with Protobuf in 2024](/posts/working-with-protobuf-in-2024/) was the compatibility matrix from hell. Managing local installations of `protoc`, `protoc-gen-go`, and half a dozen other plugins was a miserable onboarding experience. If one person had a slightly different version of a plugin, the generated code drifted, and the CI build would fail for reasons that took twenty minutes to track down.

We can finally stop doing that. Buf Remote Plugins effectively killed the "it works on my machine" version of `protoc`. By pointing `buf.gen.yaml` to remote plugins on the [Buf Schema Registry (BSR)](https://buf.build), we get deterministic, zero-install code generation.

```yaml
# buf.gen.yaml in 2026
version: v2
plugins:
  - remote: buf.build/connectrpc/go:v1.19.1
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/protocolbuffers/go:v1.34.1
    out: gen/go
    opt: paths=source_relative
```

Your CI pipeline doesn't need a bloated custom Docker image packed with binaries anymore. You just need the `buf` CLI. New hires clone the repo, run one command, and they’re done. It’s the level of "it just works" that we should have had a decade ago.

## First-Class IDE Support

Writing Protobuf used to feel like coding in a glorified Notepad. We lacked the basic editor intelligence that almost every other major language enjoys. 

That changed in early 2026 when Buf released a production-grade Language Server Protocol (LSP) server for Protobuf. It’s bundled directly into the `buf` CLI, which means whether you use VSCode or Neovim, you finally get go-to-definition and reference finding that actually works.

The LSP is workspace-aware, too. You can cmd-click an imported message from a third-party library and jump straight to the definition on the BSR without manually syncing files. It also catches syntax errors and duplicate modifiers before you even try to compile, which saves you from that annoying "context switch to terminal, run build, see error, switch back" loop.

## Format, Lint, and Breaking Changes

The Buf CLI also provides the kind of guardrails that keep a team from moving into "legacy debt" territory too quickly.

If you’ve ever sat through a PR review where someone spent ten comments arguing about whether a field should be `camelCase` or `snake_case`, `buf fmt` and `buf lint` are for you. They end the debate. You run the command, the code is formatted, and the team moves on to actually solving problems.

The real winner is `buf breaking`. In a microservices setup, accidentally deleting a field or changing a data type in your schema is a great way to wake up the on-call engineer. By running `buf breaking` in CI, you verify the current schema against previous commits. It catches destructive changes before they hit the main branch, ensuring your contracts stay stable without requiring a human to manually audit every `.proto` change.

## Data Validation

Validation has historically been a tedious chore. Writing endless `if req.Age < 0` or `if req.Email == ""` checks in every single handler is a waste of time and a magnet for bugs.

[protovalidate](https://protovalidate.com/) (which recently hit v1.0) moves those rules directly into the Protobuf schema. Since it’s built on Google's Common Expression Language (CEL), you can do more than just check for nulls; you can write complex cross-field logic, like ensuring a "start date" is always before an "end date."

By dropping the `protovalidate` interceptor into your server, requests are automatically validated before they touch your business logic. But the real "aha!" moment is the frontend. Your TypeScript client can run these same rules in the browser before the request even leaves. No more maintaining a separate Zod or Yup schema that inevitably gets out of sync with the backend. One source of truth, enforced everywhere.

## Docs and Mocks

Sharing a gRPC endpoint used to be a pain; you couldn't just hand someone a cURL command and expect it to work. ConnectRPC solved that fundamental issue by supporting standard HTTP/1.1 and JSON. But to truly treat these services like REST APIs, we needed the documentation tooling to match. That is why I spent part of 2024 working on [**protoc-gen-connect-openapi**](/posts/protoc-gen-connect-openapi/).

Now, [Self-Documenting Connect Services](/posts/self-documenting-connect-services/) are essentially the default for me. Because ConnectRPC skips binary framing for unary calls and uses standard HTTP status codes, we can generate an OpenAPI spec directly from the Protobuf definitions. You can spin up a Swagger UI directly from your server, let external users test with JSON, and keep your strict internal contracts intact.

We’ve also mostly solved the "waiting for the backend" bottleneck. [**FauxRPC**](/posts/fauxrpc/) uses your Protobuf descriptors to spin up a mock server in seconds. When you pair it with [**protovalidate**](/posts/fauxrpc-protovalidate/), the fake data is actually realistic enough to build a frontend against. Some teams are even running [FauxRPC in Testcontainers](/posts/fauxrpc-testcontainers/) for integration tests, which is much cleaner than trying to manage a "staging" backend for every test run.

## Why gRPC-Web Failed

To understand why ConnectRPC won the frontend, you have to look at the history of the protocol. Native gRPC relies on HTTP/2 trailers for status codes, but browsers do not expose those trailers to JavaScript. This originally made gRPC effectively unusable on the web.

The official solution to this problem was gRPC-Web. Its intended goal was straightforward: allow developers to use gRPC directly from web applications. As far as that specific goal goes, it was a success. You could finally make gRPC calls from a browser.

But there is a big difference between a technical success and a widely adopted standard. gRPC-Web never truly took off for a number of reasons. First, it was fundamentally unfriendly to modern infrastructure. It required a separate proxy (usually Envoy) just to translate the frontend requests into something the gRPC backend could understand. This added immediate operational overhead to every project.

Worse, it preserved the most frustrating parts of gRPC. Every single request returned a **200 OK**, regardless of whether the server crashed or the resource was missing. It was a baffling design choice that broke the internet's existing contract for observability. You could not rely on standard load balancer metrics, standard browser dev tools, or your generic APM to see if your site was actually healthy. You were forced to use specialized, protocol-aware tooling just to perform basic debugging. If I have to open a dedicated "gRPC-aware" network tab just to see why a login failed, I feel like it hasn't actually earned the "web" part of gRPC-Web name.

ConnectRPC stepped in and completely bypassed the proxy requirement. Beyond just dropping Envoy, it fixed the foundational web integration issues. A unary JSON request in ConnectRPC acts exactly like a standard REST call. If a resource is missing, you get a real **404 Not Found**, and your existing monitoring stack just works. It gave frontend developers the familiar, straightforward debugging experience they actually wanted while keeping the strict schema safety that backend teams need.

## Why it's my default choice

In 2024, ConnectRPC was about making gRPC more approachable. Now, the underlying protocol is almost an implementation detail. We get the benefits of typed schemas and code generation, but the friction of the "gRPC tax" is gone.

If you’re still hand-rolling JSON/REST APIs or wrestling with legacy gRPC-go stubs and Envoy proxies, it’s time to move on. The tools are ready, the workflow is better, and your on-call engineer will thank you.
