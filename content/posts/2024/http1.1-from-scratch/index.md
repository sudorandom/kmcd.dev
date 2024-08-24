---
categories: ["article"]
tags: ["networking", "http", "go", "golang", "tutorial", "web", "webdev"]
series: ["HTTP from Scratch"]
date: "2024-08-27"
description: "When the web became whole."
cover: "cover.jpg"
images: ["/posts/http1.1-from-scratch/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "HTTP/1.1 From Scratch"
slug: "http1.1-from-scratch"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/http1.1-from-scratch/
draft: true
---

The internet as we know it today wouldn't be possible without the continuous evolution of the underlying technologies that power it. One such pivotal advancement was the introduction of HTTP/1.1, a significant upgrade to the original HTTP/1.0 protocol.

## Introduction

In our previous posts, we explored the early days of the web with [HTTP/0.9](/posts/http0.9-from-scratch) and [HTTP/1.0](/posts/http1.0-from-scratch). We saw how HTTP/0.9 laid the groundwork for basic web communication, and how HTTP/1.0 introduced headers and status codes, paving the way for a more structured web experience. Now, it's time to delve into HTTP/1.1, the version that truly shaped the modern web.

Despite the advent of HTTP/2 and HTTP/3, HTTP/1.1 remains a cornerstone of the internet. It's far from obsolete and continues to power a significant portion of web traffic. This is wild speculation from me but I think its robustness and widespread adoption as the assumed default means it may even outlive HTTP/2.

The rapid release of HTTP/1.1, just months after HTTP/1.0, might lead some to believe that HTTP/1.0 was a failure. However, this rapid evolution is a testament to the success of HTTP/1.0 and the explosive growth of the web. HTTP/1.0 laid a solid foundation, and HTTP/1.1 built upon it to address the challenges of a rapidly expanding internet.

Following the previous format, I'm going to talk about the new features a bit before implementing them in Go.

## Key Improvements over HTTP/1.0
HTTP/1.1 introduced several critical enhancements that significantly improved the performance, efficiency, and flexibility of web communication:

### Persistent Connections
Unlike HTTP/1.0, where each request/response cycle required a new connection, HTTP/1.1 introduced persistent connections (Keep-Alive). This allowed multiple requests and responses to be sent over a single TCP connection, dramatically reducing overhead and latency.

TODO: Diagram

### Host Header
The Host header, now mandatory, enabled virtual hosting, allowing multiple websites to be served from a single server. This was crucial for the scalability of the web, as it allowed for more efficient use of server resources.

### Chunked Transfer Encoding
HTTP/1.1 introduced chunked transfer encoding, enabling the efficient transfer of dynamic content and responses of unknown size. This eliminated the need to know the entire content length beforehand, improving the handling of streaming data and large files.

```http
HTTP/1.1 200 OK
Content-Type: text/plain
Transfer-Encoding: chunked

8\r\n
kmcd.dev\r\n
12\r\n
 is awesome!\r\n
0\r\n
\r\n
```

As you can see, the message can be broken up into chunks. It first starts with the size of the next chunk and then the chunk itself, and after everything is `\r\n`, otherwise referred to as new line (`NL`) and carriage return (`CR`).

### Caching
HTTP/1.1 introduced the Cache-Control header, providing finer-grained control over caching mechanisms. This led to reduced bandwidth usage and faster page loads, as clients could cache resources more intelligently.

```shell
$ curl --http1.1 -I https://kmcd.dev
HTTP/1.1 200 OK
Date: Sat, 24 Aug 2024 10:12:55 GMT
Content-Type: text/html; charset=utf-8
Connection: keep-alive
Cache-Control: max-age=31536000, public
Server: cloudflare
```

### Additional Methods
HTTP/1.1 added new HTTP methods like PUT, PATCH, DELETE, CONNECT, OPTIONS, and TRACE, enabling more complex interactions with web resources and supporting the development of RESTful APIs.

### HTTP 100 Continue
This status code allowed for client-server negotiation before sending large request bodies, preventing unnecessary data transfer in case the server was going to reject the request anyway.

```http
PUT /images HTTP/1.1
Host: images.example.com
Content-Type: image/png
Content-Length: 341
Expect: 100-continue
```

The server responds:
```http
HTTP/1.1 100 Continue
```

And then the client will continue on with the raw PNG data.
```http
[image data]
```

`100 Continue` gives the server a chance to reject the request before the body is sent by the client. Maybe the image is too big, not a valid content type or maybe the user just doesn't have permission to upload to that path. Instead of a `100 Continue` response, the server could immediately return an appropriate error response.

### Failed Features
While HTTP/1.1 introduced many successful features, some didn't gain widespread adoption:

#### Pipelining
Pipelining allowed clients to send multiple requests without waiting for responses, potentially improving performance. However, its complexity and limited benefits led to its infrequent implementation.

#### Trailers
Trailers provided a way to send additional metadata at the end of a message. While part of the HTTP/1.1 spec, trailers were not widely adopted and are now more associated with HTTP/2.

## Building a Simple HTTP/1.1 Server
For the server, there's actually not that much new to add to enable HTTP/1.1 support.

TODO: Basically, add the features:
- Persistent Connections: support for Connection header, and based on that, don't close the connection. Add support for timeouts.
- Enforce the host header
- Chunked transfer encoding: this one is fun

## Testing
Once our HTTP/1.1 server is implemented, we can test it using various methods:

- Web Browsers: Any modern web browser can be used to send requests and interact with our server.
- CLI Tools: Command-line tools like curl can be used to send raw HTTP requests and inspect responses.
- Testing Frameworks: Consider using testing frameworks or libraries specifically designed for HTTP testing.

## Conclusion

HTTP/1.1 has played a crucial role in shaping the modern web. It powers a wide range of applications, including:

- **Web Browsing**: Every time you visit a website, HTTP/1.1 is at work behind the scenes, facilitating the communication between your browser and the server.
- **APIs**: HTTP/1.1 is the foundation for RESTful APIs, enabling seamless communication between different software applications.
- **Other Applications**: HTTP/1.1 is also used in various other applications, such as file transfer, software updates, and even IoT devices.

As we've seen, HTTP/1.1's longevity and widespread adoption suggest it may outlive its successors, HTTP/2 and HTTP/3. Its key features, such as persistent connections, virtual hosting, and chunked transfer encoding, have significantly improved the performance, scalability, and flexibility of the web.

By understanding HTTP/1.1 from scratch, we gain a deeper appreciation for the inner workings of the internet and lay a solid foundation for exploring more advanced web technologies. So, let's continue our journey into the world of HTTP and uncover the secrets that power the modern web.
