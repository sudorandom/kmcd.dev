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

{{< diagram >}}
{{< image src="reuse.svg" width="300px" class="center" >}}
{{< /diagram >}}

As a reminder, this is what HTTP/1.0 looks like, or if you decide not to reuse connections in HTTP/1.1; every request requires a new TCP connection:
{{< diagram >}}
{{< image src="noreuse.svg" width="300px" class="center" >}}
{{< /diagram >}}

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
Pipelining allows clients to send multiple requests without waiting for responses, potentially improving performance by removing some latency. However, since requests need to be responded to in the same order they arrived at, the actual benefit is fairly limited. Add to the fact that earlier requests which hitting timeouts might impact later requests, the actual usage of this feature is pretty low.

{{< diagram >}}
{{< image src="pipelining.svg" height="500px" class="center" >}}
{{< /diagram >}}

Many servers support pipelining, but enough of them don't or don't support the feature well that many clients will avoiding utilizing the feature. This was a required feature in HTTP/1.1.

#### Trailers
Trailers provided a way to send additional metadata at the end of a message. This allows handlers on the server side to give statuses and other kinds of information like checksums at the end of a request instead of the beginning. While part of the HTTP/1.1 spec, trailers were not widely adopted in this version of HTTP and are now associated more with HTTP/2. Even in HTTP/2, trailers were never fully embraced by browsers, but I'm skipping ahead here!

## Building a Simple HTTP/1.1 Server
For the server, there's actually not that much new to add to enable HTTP/1.1 support. We need:

- Require the host header
- Persistent Connections
- Chunked transfer encoding

### Require the host header
This one is trivial. I just added a condition looking for the `Host` header:
```go
if _, ok := req.Header["Host"]; !ok {
    return true, errors.New("required 'Host' header not found")
}
```
Easy peasy.

### Implementing Keep-Alive
Implementing keep-alive is a bit more complicated and it requires some changes in how we handle the connection. If you recall [from before](/posts/http1.0-from-scratch) ({{< github-link file="../http1.0-from-scratch/go/server/main.go" >}}), we had a server with the following methods:
```go
ListenAndServe() error
handleConnection(conn net.Conn) error
```
ListenAndServe() just has an infinite for loop, accepting new connections and spawning a new goroutine where it calls `handleConnection(conn)`. `handleConnection` would read a single response and close. Now we need `handleConnection` to loop as well, since that one connection can potentially handle an unlimited number of requests, not just one. So I moved all the code and added a loop to `handleConnection`. The new `handleConenction` function looks pretty simple now:

```go
func (s *Server) handleConnection(conn net.Conn) error {
	defer conn.Close()
	for {
		shouldClose, err := s.handleRequest(conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if shouldClose {
			return nil
		}
	}
}
```

Most of the code was moved to `handleRequest` without any modification except that now the code returns two things: a boolean saying if the connection should be closed and an error. The new boolean is there because some requests may decided to not keep the connection alive by setting a header that looks like `Connection: close`. Also, we have to handle if the error is an `EOF` error, because those are also normal now, because the client will eventually disconnect and that's not really an "error" we need to bubble up and log. So here's the biggest changes to `handleRequest` to support keep-alive:

```go
switch strings.ToLower(req.Header.Get("Connection")) {
case "keep-alive", "":
    req.Close = false
case "close":
    req.Close = true
}
```

This code checks the "Connection" header to see if the connection should be closed after this request or not. The default behavior is to keep the connection alive. At the end of the function, it now returns `return req.Close, nil` so that requests get closed by `handleConnection`.

### Chunked Encoding
This is probably the most interesting part of implementing HTTP/1.1. I described before, Chunked encoding allows clients to send requests responses of unknown size. It is supported for both request and response bodies.

{{< image src="conveyer.jpg" width="700px" class="center" >}}

## Testing
Once our HTTP/1.1 server is implemented, we can test it using various methods:

### Web Browsers
Any modern web browser can be used to send requests and interact with our server. If the server doesn't work with browsers then you're doing it wrong.

### CLI Tools
I like testing this kind of thing with CLI tools because it gives the most flexibility.

#### Keep-Alive
With `curl`, I can test keep-alives! Here's an example showing the server keeping the connection open and handling two different requests:
```shell
$ curl --http1.1 -I http://127.0.0.1:9000 http://127.0.0.1:9000 -v
*   Trying 127.0.0.1:9000...
* Connected to 127.0.0.1 (127.0.0.1) port 9000
> HEAD / HTTP/1.1
> Host: 127.0.0.1:9000
> User-Agent: curl/8.7.1
> Accept: */*
>
* Request completely sent off
< HTTP/1.1 200 OK
HTTP/1.1 200 OK
< Last-Modified: Sat, 24 Aug 2024 16:06:15 GMT
Last-Modified: Sat, 24 Aug 2024 16:06:15 GMT
< Content-Type: text/html; charset=utf-8
Content-Type: text/html; charset=utf-8
< Accept-Ranges: bytes
Accept-Ranges: bytes
< Content-Length: 15663
Content-Length: 15663
<

* Connection #0 to host 127.0.0.1 left intact
* Found bundle for host: 0x600001098480 [serially]
* Re-using existing connection with host 127.0.0.1
> HEAD / HTTP/1.1
> Host: 127.0.0.1:9000
> User-Agent: curl/8.7.1
> Accept: */*
>
* Request completely sent off
< HTTP/1.1 200 OK
HTTP/1.1 200 OK
< Last-Modified: Sat, 24 Aug 2024 16:06:15 GMT
Last-Modified: Sat, 24 Aug 2024 16:06:15 GMT
< Content-Type: text/html; charset=utf-8
Content-Type: text/html; charset=utf-8
< Accept-Ranges: bytes
Accept-Ranges: bytes
< Content-Length: 15663
Content-Length: 15663
<

* Connection #0 to host 127.0.0.1 left intact
```

This output was a little much but I can see a few hints that the keep-alive logic is working. Here's a stripped down version with only the relevant parts for keep-alive:
```shell
* Connected to 127.0.0.1 (127.0.0.1) port 9000
> HEAD / HTTP/1.1
* Connection #0 to host 127.0.0.1 left intact
* Re-using existing connection with host 127.0.0.1
> HEAD / HTTP/1.1
* Connection #0 to host 127.0.0.1 left intact
```
Notice how there's no request header saying that keep-alives should happen. That's because HTTP/1.1 servers will keep connections open for a short time by default.

#### Host header enforcement
Last time we used `nc` to test the server, so let's see what that looks like now:
```shell
$ printf "GET /headers HTTP/1.0\r\nMy-Custom-Header: Hello from nc!\r\n" | nc localhost 9000
```

With that command, we no longer get output and there's a new error log on the server:
```plaintext
2024/08/24 19:34:39 ERROR http error: required 'Host' header not found
```
This is intentional! The `nc` call I made before doesn't work because our server now requires the host header to be set. Let's see it again with the header set:
```shell
$ printf "GET /headers HTTP/1.0\r\nHost: localhost\r\n" | nc localhost 9000
HTTP/1.1 200 OK
Content-Type: application/json

{"Host":["localhost"]}
```
That's better! We get a response now as expected!

## Conclusion
HTTP/1.1 has played a crucial role in shaping the modern web. It powers a wide range of applications.

As we've seen, HTTP/1.1's longevity and widespread adoption suggest it may outlive its successors, HTTP/2 and HTTP/3. Its key features, such as persistent connections, virtual hosting, and chunked transfer encoding, have significantly improved the performance, scalability, and flexibility of the web.

By understanding HTTP/1.1 from scratch, we gain a deeper appreciation for the inner workings of the internet and lay a solid foundation for exploring more advanced web technologies. So, let's continue our journey into the world of HTTP and uncover the secrets that power the modern web.
