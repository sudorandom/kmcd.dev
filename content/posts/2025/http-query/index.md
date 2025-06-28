---
categories: ["article"]
tags: ["http", "protocols", "api", "webdev", "go"]
date: "2025-06-04T10:00:00Z"
cover: "cover.png"
images: ["/posts/wordseq/wordseq.svg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "HTTP QUERY and Go"
description: "We need another HTTP verb. I'll explain why."
slug: "http-query"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/http-query/
---

You're likely familiar with the HTTP methods, **GET** and **POST**, the workhorses of HTTP. These have both worked surprisingly well and have provided well-defined caching behavior for over a quarter of a century. However, neither of these solves the problem of complex request parameters without completely throwing out the caching semantics of `GET`. This is where a concept like a **QUERY** method comes in, and it's not just a thought experiment; it's an area actively being [explored by the IETF HTTP Working Group](https://httpwg.org/http-extensions/draft-ietf-httpbis-safe-method-w-body.html).

### But, why?

The standard HTTP methods serve us well, except the ones that don't... but that's a topic for another day. **GET** is great for simple data retrieval, but its reliance on URL parameters makes it cumbersome for complex queries or large sets of input parameters. URLs have practical length limits, and embedding deeply nested structures is awkward.

This often leads developers to use **POST** for what are semantically read-only query operations. Many widely-used protocols effectively do this:

  * **GraphQL** typically uses POST requests with JSON bodies to send queries.
  * **gRPC** and **gRPC-Web** typically rely exclusively on POST.
  * Older protocols like **SOAP** and **XML-RPC** almost exclusively use POST to encapsulate their operations, including data retrieval. This has been an issue for a long time!

While using POST works, it's a compromise. POST traditionally implies an action that might change state on the server, which means intermediaries (read: caches) and even client-side logic might treat these "query-via-POST" requests with undue caution, forgoing caching or automatic retries that would be safe for a truly read-only operation.

This is precisely the problem that a dedicated safe method with a request body aims to solve. The IETF HTTP Working Group is discussing such a method in a draft titled ["A Safe HTTP Method with a Request Content"](https://httpwg.org/http-extensions/draft-ietf-httpbis-safe-method-w-body.html). As the draft states:

> The QUERY method provides a solution that spans the gap between the use of GET and POST. As with POST, the input to the query operation is passed along within the content of the request rather than as part of the request URI. Unlike POST, however, the method is explicitly safe and idempotent, allowing functions like caching and automatic retries to operate.

While the draft uses "QUERY" as a candidate name (among others), the core idea is what we're exploring: a method for safe, idempotent data retrieval that can carry a payload. For the rest of this article, we'll continue to use "QUERY" to represent this concept and show how you can implement such a custom method in Go today.

It's crucial to remember that until such a method is formally standardized and widely adopted, **using custom HTTP methods can impact interoperability**. However, for internal APIs, tightly controlled systems, or as a forward-looking experiment, they can be very useful.

### Server-Side with Go

Go 1.22 introduced enhancements to `http.ServeMux` that allow you to register handlers for specific HTTP methods and paths more directly. Let's build a server that handles our custom **QUERY** method:

{{% render-code file="go/server.go" language="go" %}}

In this server:

1.  We use `mux.HandleFunc("QUERY /data", queryHandler)` to directly associate the `queryHandler` with our custom **QUERY** HTTP method for the `/data` path.
2.  The `queryHandler` reads the request body, where the complex query parameters would reside.

Congratulations, we've made an API that uses the QUERY method. Now let's write a client to pair with this server.

### Client-Side, also with Go

Here's how a Go client can send a **QUERY** request:

{{% render-code file="go/client.go" language="go" %}}

The client uses `http.NewRequest("QUERY", ...)` to specify the custom method and sends the `queryPayload` in the request body.

### Client-Side with cURL

Although web browsers won't randomly make QUERY calls, many other tools can use QUERY. Many also support arbitrary HTTP verbs, although this isn't typically leveraged due to filters from load balancers, firewalls, etc.

Anyway, here's what that request looks like with cURL:

```shell
$ curl -X QUERY -d '{"filters": {"status": "active", "category": "electronics"}, "
fields": ["name", "price"]}' http://localhost:8080/data -v
* Host localhost:8080 was resolved.
* IPv6: ::1
* IPv4: 127.0.0.1
*   Trying [::1]:8080...
* Connected to localhost (::1) port 8080
> QUERY /data HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/8.7.1
> Accept: */*
> Content-Length: 89
> Content-Type: application/x-www-form-urlencoded
> 
* upload completely sent off: 89 bytes
< HTTP/1.1 200 OK
< Date: Sat, 31 May 2025 15:47:45 GMT
< Content-Length: 129
< Content-Type: text/plain; charset=utf-8
< 
Received your QUERY request with body: {"filters": {"status": "active", "category": "electronics"}, "fields": ["name", "price"]}
* Connection #0 to host localhost left intact
```

### QUERY vs. GET and POST - A Clearer Separation

Let's summarize the distinctions with our (potentially future-standard) QUERY method in mind:

  * **GET**: Used for retrieving data. Parameters are typically sent in the URL. GET requests *must* be safe and idempotent. They generally don't have a body.
  * **POST**: Used for submitting data to be processed, often resulting in a change of state or side effects on the server (e.g., creating a new resource). Parameters are sent in the request body. Not necessarily safe or idempotent.
  * **QUERY (custom/proposed)**: Used to *request data* (like GET) but with the ability to send a *complex query in the request body* (like POST). Crucially, it is defined as **safe and idempotent** (like GET). This explicitly tells intermediaries and clients that the request has no side effects and can be cached or retried automatically.

### Caching Behavior: GET, POST, and QUERY

HTTP caching is vital for performance. A method's characteristics (especially safety and idempotency) directly influence its cacheability.

#### 1. GET

  * **Behavior**: GET requests are inherently cacheable. Caches readily store and serve responses to GET requests if caching headers (like `Cache-Control`, `Expires`, `ETag`) allow. This is a foundational aspect of HTTP performance.

#### 2. POST

  * **Behavior**: POST requests are generally *not* cacheable by default. Since POST can have side effects, caching responses could lead to unintended consequences or stale data if the action isn't repeated.

#### 3. QUERY

The IETF draft emphasizes that a method like QUERY is "explicitly safe and idempotent, allowing functions like caching and automatic retries to operate." This is key.

  * **Behavior (Current Custom Method)**: Because our **QUERY** method is non-standard *today*, caches will **not cache it by default**. They typically only automatically consider standard safe methods like GET.
    To make responses to a custom QUERY request cacheable:
      * The **server must send explicit caching headers** (e.g., `Cache-Control: public, max-age=3600`). These headers signal the cacheability of the response.
      * Intermediary caches (CDNs, reverse proxies) might need **specific configuration** to recognize and cache responses for this custom method.
      * The `Vary` HTTP header is important if the response depends on the request body content.
  * **Behavior (Future Standardized Method)**: If a method like QUERY becomes a recognized HTTP standard, caches would likely treat it similarly to GET for caching purposes, provided it's implemented according to its safe and idempotent semantics. This would be a major advantage, allowing complex queries with bodies to be cached as effectively as GET requests.

### Key Takeaway for QUERY Caching

The *intent* of a QUERY method is to be cacheable. If using a custom method today, you must provide explicit caching directives. The ongoing standardization effort aims to make this caching behavior more automatic and universally understood, unlocking performance benefits for complex, body-inclusive queries that are currently often forced into less cache-friendly POST requests. How long will this process take? Who knows! These kinds of changes can take decades.

## What's Next?

The discussion around a safe HTTP method with a request body is an exciting development. By understanding its purpose and experimenting with custom methods like QUERY in Go, we can better appreciate the nuances of HTTP and prepare for potential future standards that will make our APIs more robust and performant.

Some projects have already taken to adding support for the QUERY method. Take a look at [NodeJS: Support for 'QUERY' method](https://github.com/nodejs/node/issues/51562), which added support last year. As demonstrated earlier in this article, using `QUERY` is already possible in Go. This general support is often true for other languages and libraries as well, since custom HTTP methods are a feature of the HTTP specification.

The other aspect of moving this forward is a bit more nebulous: bureaucracy. There is still work and review to be done to graduate the draft into an official RFC from the IETF. But fear not. There's actually steady changes being made to the draft document. You can tell this from the [git history](https://github.com/httpwg/http-extensions/commits/main/draft-ietf-httpbis-safe-method-w-body.xml) on the httpwg's repo. At the time of writing May 19th was when the last change was made, so I'm certain that this hasn't been forgotten about.
