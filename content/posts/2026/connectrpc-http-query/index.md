---
title: "It's Time for ConnectRPC to Adopt HTTP QUERY"
date: "2026-07-28T10:00:00Z"
tags: ["connectrpc", "http", "networking", "api", "protobuf", "golang"]
categories: ["Backend Engineering"]
slug: "connectrpc-http-query"
cover: "cover.svg"
images: ["/posts/connectrpc-http-query/cover.svg"]
type: "posts"
devtoSkip: true
---

HTTP QUERY is now standardized in [RFC 10008](https://datatracker.ietf.org/doc/html/rfc10008). For backend engineers working with schema-first APIs, this provides a clean solution for a problem we have been working around for years: cacheable requests that require complex, structured input.

That matters for a protocol like ConnectRPC. Connect already tries to map RPCs onto ordinary HTTP instead of fighting the grain of the web. It supports HTTP GET for side-effect-free unary RPCs, which is great for caching, but GET forces structured request payloads into query parameters.

ConnectRPC should support QUERY as the body-carrying counterpart to GET for those same side-effect-free unary calls. Caching support across the broader web can mature over time, but the protocol shape is already useful today: structured request data belongs in the request body, not squeezed into a URL.

### Why Connect Got It Right

One of the best design decisions in the Connect protocol is that it leans into standard HTTP semantics. Unlike traditional gRPC, which demands HTTP/2 and relies heavily on trailing headers, Connect maps naturally onto the HTTP infrastructure most teams already run.

That pays off operationally:

* **Meaningful HTTP Status Codes:** Errors map directly to standard HTTP statuses, allowing metrics to work without a specialized gRPC proxy.
* **Standard Compression:** Traffic relies on standard `Content-Encoding` headers (like gzip or brotli) already built into your infrastructure.
* **No Trailers Required:** Requests pass cleanly through standard load balancers, firewalls, and HTTP/1.1 proxies without requiring end-to-end HTTP/2.
* **Native Ecosystem Integration:** The protocol plugs directly into Go's standard `net/http` stack, allowing you to reuse standard middleware, multiplexers, and observability tools.

Building on this foundation, Connect allows any unary RPC marked as side-effect free (`NO_SIDE_EFFECTS` in Protobuf) to be invoked via HTTP GET, unlocking caching at the CDN or proxy layer.

### The Problem with GET and Query Parameters

To make GET work with complex schema definitions, the protocol has to perform significant gymnastics. Because GET request bodies have no defined semantics and are routinely ignored or rejected by intermediate proxies, Connect is forced to cram structured payloads into the URL.

For a simple JSON request, the client must serialize the payload, URL-encode it, and append it as a query parameter:

```http
GET /connectrpc.greet.v1.GreetService/Greet?connect=v1&encoding=json&message=%7B%22name%22%3A%22Buf%22%7D HTTP/1.1
Host: demo.connectrpc.com
```

If you use binary Protobuf or compression, the overhead increases further, requiring base64 encoding along with additional control parameters. Shoving these complex payloads into URLs creates immediate practical problems for production systems:

* **Bloated URLs:** Complex requests easily hit maximum URL length limits enforced by load balancers, reverse proxies, and older browsers.
* **Leaky Logs:** Query parameters show up in plain text in standard [Nginx](https://nginx.org/) or [Apache](https://httpd.apache.org/) access logs, WAF dashboards, and observability tools. If a request contains sensitive filter criteria, teams are forced to write custom masking rules to scrub their logs.
* **Encoding Friction:** Maintaining a separate serialization path just for GET requests introduces branching logic into the codebase. Clients and servers must implement special handling to treat this specific verb entirely differently than the rest of the API surface.

### Enter HTTP QUERY

QUERY gives HTTP the method shape this use case has been missing. Semantically, it is defined as a safe, idempotent method. Mechanically, it operates like a POST, allowing a standard request body.

If ConnectRPC adopts QUERY, the entire query parameter encoding scheme can be dropped. A QUERY request looks exactly like a POST request on the wire, keeping the payload in the HTTP body natively encoded as `application/json` or `application/proto`.

Here is how a proposed JSON-based QUERY wire request looks:

```http
QUERY /connectrpc.greet.v1.GreetService/Greet HTTP/1.1
Host: demo.connectrpc.com
Content-Type: application/json
Connect-Protocol-Version: 1

{"name":"Buf"}
```

And for binary Protobuf:

```http
QUERY /connectrpc.greet.v1.GreetService/Greet HTTP/1.1
Host: demo.connectrpc.com
Content-Type: application/proto
Connect-Protocol-Version: 1

<binary protobuf>
```

This is the core protocol win: **QUERY lets Connect reuse the normal unary POST body format instead of maintaining a specialized GET query-encoding path.**

### Caching Demands QUERY-Aware Infrastructure

From a network caching perspective, QUERY is not simply "GET with a body." RFC 10008 specifies that for a QUERY response to be cached, the cache key must incorporate the request body content alongside related metadata.

Most existing HTTP caching infrastructure is built strictly around request metadata such as the method, scheme, host, path, query string, and selected headers. Supporting QUERY requires intermediate proxies, gateways, and CDNs to inspect and hash request bodies. Until major CDNs document first-class support for body-aware cache keys, QUERY caching should be treated as experimental outside infrastructure you control directly.

### The Browser Story

Browser support is not a reason to avoid QUERY; it is a reason to implement it deliberately.

Because `QUERY` is not a CORS-safelisted method, cross-origin browser clients require the server or gateway to allow it in preflight responses. This is a standard deployment requirement for modern APIs. Many Connect-Web deployments already require custom CORS configuration for protocol headers, content types, and credentials.

Adopting QUERY gives web clients a clean way to express search forms, filtered list views, reporting queries, and batch reads without resorting to URL hacks.

### A Pragmatic Rollout Strategy

We cannot flip a switch and expect a new HTTP verb to work across the public internet immediately. The web is built on middleboxes, load balancers, strict firewalls, and managed WAFs that are inherently suspicious of unfamiliar traffic and will likely reject QUERY requests as malformed for some time.

However, many modern backend servers accept arbitrary verbs without complaint. You can verify how your current stack handles unfamiliar methods right now using [httpbin.io](https://httpbin.io):

```bash
$ curl -X QUERY [https://httpbin.io/status/200](https://httpbin.io/status/200) -w "%{http_code}"
200
```

To prove the server simply accepts the string token, you can test an arbitrary string:

```bash
$ curl -X YEET [https://httpbin.io/status/200](https://httpbin.io/status/200) -w "%{http_code}"
200
```

While this does not prove full RFC 10008 compliance, it confirms that the initial deployment barrier at the server application layer is low. The harder problem is teaching intermediaries, caches, and client libraries what QUERY actually means.

The most effective early deployments will target paths where engineering teams control the entire network hop: internal service-to-service traffic and browser-facing APIs behind configurable API gateways. Protocols like Connect are the ideal starting point because they can expose QUERY as an opt-in transport mechanism while the broader networking ecosystem catches up.

### What QUERY Should Not Replace Yet

While QUERY is the cleaner protocol shape, a practical rollout requires clear boundaries:

* **GET remains useful for small, URL-friendly requests:** For simple payloads that fit naturally in a URI, GET continues to offer mature browser integration, native CDN caching, and effortless manual debugging.
* **POST remains the compatibility fallback:** When requests have side effects, or when traffic must route through uncooperative legacy intermediaries, POST remains the universal standard.
* **QUERY starts as opt-in:** Support should initially be opt-in for unary RPCs explicitly marked `NO_SIDE_EFFECTS`. This option should be exposed to both server-side and web clients, backed by clear deployment guidance for CORS, gateways, and cache behavior.

The QUERY method solves a real, persistent networking problem. It is time to put it to work.
