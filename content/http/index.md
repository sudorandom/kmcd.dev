---
description: "What version of HTTP are you connecting with?"
linktitle: ""
title: "What version of HTTP are you connecting with?"
titleIcon: "fa-globe"
cover: "cover.jpg"
subtitle: ""
devtoSkip: false
layout: http
---

### HTTP/1.0
HTTP/1.0 Introduced status codes, headers for content negotiation, and support for various media types.


Each request/response required a new TCP connection, leading to overhead and performance issues.

- [RFC 1945](https://www.rfc-editor.org/rfc/rfc1945)


### HTTP/1.1

HTTP/1.1 introduced persistent connections (keeping a TCP connection open for multiple requests/responses), chunked transfer encoding, virtual hosting,and caching mechanisms.


HTTP/1.1 has limited support for concurrency and requires opening multiple connections to perform requests in parallel. It is also still text-based so the overhead for requests is rather large. Headers also cannot be compressed.

- [RFC 2616 (obsoleted)](https://www.rfc-editor.org/rfc/rfc2616)
- [RFC 7230](https://www.rfc-editor.org/rfc/rfc7230) - [RFC 7235](https://www.rfc-editor.org/rfc/rfc7235)


### HTTP/2

HTTP/2 introduced binary framing, header compression (HPACK), multiplexing (multiple requests/responses concurrently over a single connection), and server push.


HTTP/2 still relies on TCP, which can suffer from head-of-line blocking and performance issues on lossy networks.

- [RFC 7540](https://www.rfc-editor.org/rfc/rfc7540)


### HTTP/3

HTTP/3 uses QUIC as its transport protocol which is built on UDP, providing faster connection establishment, improved congestion control, and multiplexedstreams that are independently reliable.


HTTP/3 is relatively new, so adoption is still growing. However, [](https://caniuse.com/http3)all major browsers support HTTP/3so for the web, adoption is mostly held up with server support.

- [RFC 9114](https://www.rfc-editor.org/rfc/rfc9114)


I also wrote a post about this page. Feel free to [read it here](/posts/http-tool/).