---
title: "It's Time for ConnectRPC to Adopt HTTP QUERY"
date: 2026-08-04T15:45:12+02:00
tags: ["connectrpc", "http", "networking", "api", "protobuf", "golang"]
categories: ["Backend Engineering"]
slug: "connectrpc-http-query"
cover: "cover.svg"
images: ["/posts/connectrpc-http-query/cover.svg"]
type: "posts"
devtoSkip: true
---

HTTP QUERY is now standardized in [RFC 10008](https://datatracker.ietf.org/doc/html/rfc10008). For backend engineers working with schema-first APIs, this is a much bigger deal than it might seem at first glance. It finally provides a clean solution for a problem we have been working around for years: cacheable requests that require complex, structured input.

That matters for a protocol like ConnectRPC. Connect already tries to map RPCs onto ordinary HTTP instead of fighting the grain of the web. It supports HTTP GET for side-effect-free unary RPCs, which is great for caching, but GET forces structured request payloads into query parameters.

ConnectRPC should support QUERY as the body-carrying counterpart to GET for those same side-effect-free unary calls. That should include web clients. Caching support can mature over time, but the protocol shape is already useful: structured request data belongs in the request body, not squeezed into a URL.

### Why Connect Got It Right

One of the best design decisions in the Connect protocol is that it leans into standard HTTP semantics. Unlike traditional gRPC, which demands HTTP/2 and relies heavily on trailing headers, Connect maps much more naturally onto the HTTP infrastructure most teams already run.

That pays off operationally:

* **Meaningful HTTP Status Codes:** Mapping errors directly to standard HTTP statuses so metrics work without a gRPC proxy.
* **Standard Compression:** Relying on standard `Content-Encoding` headers (like gzip or brotli) already built into your infrastructure.
* **No Trailers Required:** Allowing traffic to pass through standard load balancers, firewalls, and HTTP/1.1 proxies without requiring end-to-end HTTP/2.
* **Native Ecosystem Integration:** Plugging directly into Go's standard `net/http` stack, allowing you to reuse standard middleware, multiplexers, and observability.

Building on this foundation, Connect allows any unary RPC marked as side-effect free (`NO_SIDE_EFFECTS` in Protobuf) to be invoked via HTTP GET, unlocking caching at the CDN or proxy layer.

### The Problem with GET

But to make GET work with complex schema definitions, the protocol has to perform some gymnastics. Because GET request bodies have no generally defined semantics and are often ignored or rejected by intermediate proxies, Connect is forced to cram the structured request payload into the URL.

This results in a completely different encoding path. For a simple JSON request, the client must serialize the payload, URL-encode it, and append it as a query parameter, yielding a request that looks like this:

```http
GET /connectrpc.greet.v1.GreetService/Greet?connect=v1&encoding=json&message=%7B%22name%22%3A%22Buf%22%7D HTTP/1.1
Host: demo.connectrpc.com
```

If you are using the binary Protobuf format or compression, the overhead is even higher, requiring the client to base64-encode the binary bytes before appending them, along with additional control parameters. It works, but it brings a lot of baggage, creates split serialization paths, and turns URLs into protocol envelopes.

### The Pain Points of Query Parameters

Shoving a complex payload into a URL creates immediate practical problems for production systems:

* **Bloated URLs:** Complex requests can produce very large URIs. You can easily hit the maximum URL length limits enforced by load balancers, reverse proxies, or older browsers.
* **Leaky Logs:** Query parameters are notoriously visible. They show up in plain text in standard [Nginx](https://nginx.org/) or [Apache](https://httpd.apache.org/) access logs, WAF dashboards, and observability tools. If your request contains any sensitive information, you are forced to write custom masking rules to scrub your logs.
* **Encoding Friction & Special Handling:** Maintaining a separate encoding and decoding path just for GET requests forces branching logic into the codebase. Clients and servers must implement special handling to treat this specific verb entirely differently than the rest of the API surface.

### Enter HTTP QUERY

QUERY gives HTTP the method shape this use case has been missing. Semantically, it is defined as a safe, idempotent method. Mechanically, it operates like a POST, allowing you to include a standard request body.

If ConnectRPC adopts QUERY, the benefits are immediate and obvious. The entire query parameter encoding scheme can be dropped. A QUERY request would look exactly like a POST request on the wire. The payload stays in the HTTP body where it belongs, encoded natively as `application/json` or `application/proto`.

Here is how a proposed JSON-based QUERY wire request would look:

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

This is the important protocol-level win: **QUERY lets Connect reuse the normal unary POST body format instead of maintaining the special GET query-encoding path.**

Your URLs remain clean and readable. Your server logs stop filling up with giant base64 strings. You no longer have to worry about hitting arbitrary URI length limits when a client sends large filter criteria.

### Caching Demands QUERY-Aware Infrastructure

It is vital to recognize that QUERY is not "GET, but with a body" from a network caching perspective. RFC 10008 specifies that for a QUERY response to be cached, the cache key must incorporate the request body content and related metadata.

Most HTTP caching infrastructure is built around request metadata such as the method, scheme, host, path, query string, and selected headers. QUERY changes that model by requiring caches to account for request content too. To support QUERY caching, intermediate proxies, gateways, and CDNs must be updated to inspect and hash request bodies. Until major CDNs document first-class QUERY support, especially body-aware cache keys, QUERY caching should be treated as experimental outside infrastructure you control.

### The Browser Story

Browser support is not a reason to avoid QUERY. It is a reason to implement it deliberately.

Because `QUERY` is not a CORS-safelisted method, cross-origin browser clients will need the server or gateway to allow it in preflight responses. That is a real deployment requirement, but it is not unusual for API protocols. Many Connect-Web deployments already need CORS configuration for protocol headers, content types, and credentials.

The important point is that QUERY gives browser clients a cleaner way to express safe, structured requests. A search form, filtered list view, reporting query, or batch read can send its request as JSON or binary Protobuf in the body instead of packing the whole thing into `?message=...`.

Caching does not have to work everywhere on day one for this to be worthwhile. CDNs and middleware can add body-aware QUERY caching later. Until then, QUERY still improves the wire format and removes the worst parts of GET-based payload encoding.

### Start Where You Can Control the Path

The easiest early deployments will be the paths where teams control the server, gateway, and proxy configuration. That includes internal service-to-service traffic, but it also includes browser-facing APIs behind modern gateways that can be configured to allow `QUERY`.

For internal RPCs, the benefits are obvious: large safe requests can use the normal unary body format without hitting URL length limits or leaking encoded payloads into access logs. For browser clients, the benefit is just as practical: complex filters, search requests, and batch reads can stop pretending to be URL parameters.

QUERY caching can arrive later. The first step is making the method available and giving clients a cleaner transport option.

### Setting Expectations

Of course, we cannot flip a switch and expect this to work everywhere tomorrow. The internet is built on middleboxes that are highly suspicious of anything new. 

Load balancers, strict firewalls, and managed WAFs will likely drop QUERY requests or reject them as malformed traffic for a while. It will take time for the networking infrastructure ecosystem to catch up and update their routing rules to support the new verb.

However, that does not mean you cannot start experimenting today. In fact, many modern backend servers will accept arbitrary verbs without complaint. You can test this right now with QUERY using [httpbin.io](https://httpbin.io):

```bash
$ curl -X QUERY https://httpbin.io/status/200 -w "%{http_code}"
200
```

To prove the server is just happily accepting it as a string, you can even use a completely ridiculous custom verb:

```bash
$ curl -X YEET https://httpbin.io/status/200 -w "%{http_code}"
200
```

This does not prove real QUERY support. It only proves that some backend stacks already treat the method token as an ordinary string. That is still useful: the first deployment barrier may be lower than it looks. The harder problem is teaching intermediaries, caches, security tools, and client libraries what QUERY means.

Protocols like Connect are exactly where this should start, because they can expose QUERY as an opt-in transport before the whole public internet is ready for it. We can test it internally, enable it for web clients behind gateways that allow `QUERY`, and be ready for the day when more of the broader internet natively understands it.

### What QUERY Should Not Replace Yet

While QUERY is the cleaner protocol shape, it is not a direct drop-in replacement for every use case today. A practical rollout needs clear boundaries:

* **GET remains useful for small, URL-friendly requests:** For simple payloads that fit naturally in the URL, GET continues to offer mature browser integration, native CDN caching, and easy manual debugging.
* **POST remains the compatibility fallback:** If a request has side effects or needs guaranteed routing through uncooperative intermediaries, POST is still the universal standard.
* **QUERY starts as opt-in:** Support for QUERY should initially be opt-in for unary RPCs explicitly marked `NO_SIDE_EFFECTS`. That opt-in should be available to both server-side clients and web clients, with deployment guidance for CORS, gateway configuration, and cache behavior.

The QUERY method solves a very real, very annoying problem. Let's start using it.
