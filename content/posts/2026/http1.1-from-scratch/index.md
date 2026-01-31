---
categories: ["article"]
tags: ["networking", "http", "go", "golang", "tutorial", "web", "webdev"]
series: ["HTTP from Scratch"]
date: "2026-02-04"
description: "When the web became whole."
cover: "cover.svg"
images: ["/posts/http1.1-from-scratch/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "HTTP/1.1 From Scratch"
slug: "http1.1-from-scratch"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/http1.1-from-scratch/
draft: false
---

HTTP/1.1 has become synonymous with "HTTP/1" because it made several steps towards enabling the scale that the web was starting to experience in the late 1990s. It took the foundational concepts we explored in [HTTP/0.9](/posts/http0.9-from-scratch) and [HTTP/1.0](/posts/http1.0-from-scratch) and made several advancements and adjustments that enable the web to scale for almost two decades.

While newer protocols like `HTTP/2` and `HTTP/3` have since arrived with their own improvements, HTTP/1.1 remains a non-negotiable requirement... well, almost. There are a few who believe that it is time to [kill HTTP/1.1](https://http1mustdie.com/) and some believe it would immediately [reduce bot traffic](https://markmcb.com/web/selectively_disabling_http_1/). It has historically been the default transport for the web and the protocol that servers and clients fall back on. Its simplicity and power are why, even now, a massive portion of internet traffic flows over HTTP/1.1.

Let's start looking at the new features in HTTP/1.1 that made it a such a sturdy pillar of the web for so long and then build a server in Go that implements them from scratch.

## Improvements over HTTP/1.0

HTTP/1.1 wasn't just a bump in the version number; it was a targeted response to the scaling bottlenecks of the 90s. It introduced specific features to cut down on connection overhead and handle dynamic data more efficiently than 1.0 ever could.

### Persistent Connections (Keep-Alive)

This is probably the most important performance improvement in HTTP/1.1. In HTTP/1.0, every single request required a new, separate TCP connection. Setting up a TCP connection is a multi-step handshake process that introduces significant latency. For a typical website that requires dozens of resources (CSS, JavaScript, images), this overhead added up quickly. And this was before the web was encrypted, so wrapping requests in TLS for "secure pages" would add two more round trips too this.

HTTP/1.1 introduced persistent connections by default. This allows the browser to send multiple requests over a single TCP connection, eliminating the repeated connection setup cost.

{{< diagram >}}
{{< image src="http-versions.svg" width="800px" class="center" title="HTTP/1.1 reuses a single TCP connection for multiple requests." >}}
{{< /diagram >}}

A client or server can signal that they wish to close the connection after a request by sending the `Connection: close` header. Otherwise, the connection is assumed to be "kept alive".

### The `Host` Header: Enabling Virtual Hosting

In the early web, a server at a specific IP address hosted a single website. As the web grew, this became incredibly inefficient. The `Host` header, which became mandatory in HTTP/1.1, solved this. It specifies the domain name of the server the client is trying to reach. This allowed a single server (with a single IP address) to host hundreds or thousands of different websites, a practice known as virtual hosting. This was a critical innovation for the economic scalability of web hosting.

### Chunked Transfer Encoding

Before HTTP/1.1, for a server to send a response, it had to know its exact size beforehand to set the `Content-Length` header. This was fine for static files but a major problem for dynamically generated content. What if you were streaming a large video or generating a big HTML page on the fly? You'd have to buffer the entire response in memory just to calculate its size.

Chunked Transfer Encoding elegantly solves this. The server can send the response body in a series of "chunks." Each chunk is prefixed with its size in hexadecimal, followed by the chunk data itself. The stream is terminated by a final chunk of size 0.

Here's what a chunked response looks like on the wire:
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
This allowed for much more efficient handling of dynamic content and laid the groundwork for streaming media.

### Modern Caching with `Cache-Control`

HTTP/1.0 had basic caching headers, but HTTP/1.1 introduced the powerful `Cache-Control` header. This gave developers fine-grained control over how browsers and intermediate proxies cache resources. Directives like `max-age`, `public`, `private`, `no-cache`, and `no-store` allowed for sophisticated caching strategies, dramatically reducing bandwidth usage and improving load times.

A quick look at my own site's headers shows this in action:
```shell
$ curl --http1.1 -I https://kmcd.dev
HTTP/1.1 200 OK
Date: Sat, 24 Jan 2026 10:12:55 GMT
Content-Type: text/html; charset=utf-8
Connection: keep-alive
Cache-Control: max-age=31536000, public
Server: cloudflare
```
This tells any browser or CDN that it's safe to cache this response for a full year, which is great for performance.

### More Methods for RESTful APIs

HTTP/1.0 was primarily about `GET`, `POST`, and `HEAD`. HTTP/1.1 expanded the vocabulary of the web by adding new methods that were crucial for the development of RESTful APIs:
- **PUT:** Create or replace a resource at a given URI.
- **PATCH:** Partially modify a resource.
- **DELETE:** Delete a resource at a given URI.
- **CONNECT:** Used for creating tunnels, most notably for HTTPS through proxies.
- **OPTIONS:** Describe the communication options for the target resource.
- **TRACE:** Performs a message loop-back test along the path to the target resource.

### `100 Continue` Status Code

When a client needs to send a large request body (like uploading a file), it can be inefficient to send the entire payload only for the server to reject it (e.g., due to authentication failure or size limits). The `100 Continue` status code provides a solution.

The client can send the request headers with the `Expect: 100-continue` header and then wait.
```http
PUT /images HTTP/1.1
Host: images.example.com
Content-Type: image/png
Content-Length: 500000
Expect: 100-continue

```
If the server is willing to accept the request, it responds with `HTTP/1.1 100 Continue`. The client then proceeds to send the request body. If the server is not going to accept it, it can immediately send a final error code like `413 Payload Too Large`, and the client knows not to waste bandwidth sending the body.

## Building a Simple HTTP/1.1 Server in Go

Now for the fun part. Let's build a server that understands these new features. We'll be pulling from the complete server code found here: {{< github-link file="go/server/main.go" >}}.

### Handling Persistent Connections

To support keep-alive, our connection handler can't just handle one request and then close the connection. It needs to loop, processing multiple requests on the same connection until the client or server decides to close it.

The structure of our server looks like this: `ListenAndServe` accepts new TCP connections and spins up a `handleConnection` goroutine for each one.
```go
// ListenAndServe starts the server.
func (s *Server) ListenAndServe() error {
	// ... listener setup ...
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go func() {
			if err := s.handleConnection(conn); err != nil {
				slog.Error(fmt.Sprintf("http error: %s", err))
			}
		}()
	}
}
```

Now let's look at the new `handleConnection` method. It contains an infinite loop that repeatedly calls `handleRequest`. Previously, this method didn't exist because in HTTP/1.0 we only ever handled a single request per connection. The loop only exits if there's a fatal error (timeout, protocol error, etc.) or if `handleRequest` signals that the connection should be closed like when the user sends a request with the header `Connection: close`.
```go
func (s *Server) handleConnection(conn net.Conn) error {
	defer conn.Close()
	for {
		// handleRequest does the work of reading and responding
		shouldClose, err := s.handleRequest(conn)
		if err != nil {
			// io.EOF is a normal way for a persistent connection to end.
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if shouldClose {
			return nil // Client requested a close, so we exit the loop.
		}
	}
}
```

Inside `handleRequest`, we determine if the connection should be closed by inspecting the `Connection` header.
```go
// Default to keeping the connection alive for HTTP/1.1
req.Close = false 

// Check if the client or a previous handler wants to close the connection.
switch strings.ToLower(req.Header.Get("Connection")) {
case "close":
    req.Close = true
case "keep-alive":
	// This is the default, but we handle it explicitly.
    req.Close = false
}
```

### Requiring the `Host` Header

This is a simple but crucial part of our server. After parsing the headers, we just check for the presence of the `Host` header. If it's missing, we return an error and close the connection.
```go
if _, ok := req.Header["Host"]; !ok {
    // We send an error response here in a real implementation.
    return true, errors.New("required 'Host' header not found")
}
```

### Handling Chunked Bodies

This is the most complex part. We need to be able to both read a chunked request body from a client and send a chunked response.

#### Reading a Chunked Request

When we parse the request headers, if we see `Transfer-Encoding: chunked`, we know we can't just use `io.LimitReader` with `Content-Length`. Instead, we need a special reader. Our `chunkedBodyReader` does just this.
```go
type chunkedBodyReader struct {
	reader *bufio.Reader
	n      int64 // bytes left in current chunk
	err    error
}

func (r *chunkedBodyReader) Read(p []byte) (n int, err error) {
	if r.err != nil {
		return 0, r.err
	}
	// Do we need to read the size of the next chunk?
	if r.n == 0 {
		r.n, r.err = r.readChunkSize()
		if r.err != nil {
			return 0, r.err
		}
	}
	// If the next chunk size is 0, we're at the end.
	if r.n == 0 {
		return 0, io.EOF
	}
    // ... logic to read from the current chunk ...
}
```
The `readChunkSize` method is responsible for reading a line, parsing the hexadecimal chunk size, and preparing the reader to consume that many bytes.

#### Sending a Chunked Response

On the response side, things are even cooler. If our `http.ResponseWriter` implementation doesn't have a `Content-Length` set when `WriteHeader` is called, we can automatically switch to using chunked encoding.

Our `responseBodyWriter` checks for this condition.
```go
func (r *responseBodyWriter) writeHeader(conn io.Writer, proto string, headers http.Header, statusCode int) error {
	_, clSet := r.headers["Content-Length"]
	_, teSet := r.headers["Transfer-Encoding"]
	// If no length is set, we decide to use chunking.
	if !clSet && !teSet {
		r.chunkedEncoding = true
		r.headers.Set("Transfer-Encoding", "chunked")
	}
    // ... write headers ...
}
```
Then, the `Write` method, if `chunkedEncoding` is true, will write each chunk with the required formatting.
```go
func (r *responseBodyWriter) Write(b []byte) (int, error) {
	// ... ensure headers are written ...

	if r.chunkedEncoding {
		// Write the chunk size in hex, followed by \r\n
		chunkSize := fmt.Sprintf("%x\r\n", len(b))
		if _, err := r.conn.Write([]byte(chunkSize)); err != nil {
			return 0, err
		}
	}

	// Write the actual chunk data
	n, err := r.conn.Write(b)
	if err != nil {
		return n, err
	}

	if r.chunkedEncoding {
		// Write the trailing \r\n for the chunk
		if _, err := r.conn.Write(nlcf); err != nil {
			return n, err
		}
	}

	return n, nil
}
```
Finally, after the last `Write` call, we send the terminal `0\r\n\r\n` chunk to signal the end of the response.

## Testing Our Server

Using command-line tools is the best way to see these features in action.

### Testing Keep-Alive

We can use `curl`'s verbose mode (`-v`) to see connection reuse. We'll make two requests to our server on the same command line.
```shell
$ curl --http1.1 -v http://127.0.0.1:9000/headers http://127.0.0.1:9000/status/204

*   Trying 127.0.0.1:9000...
* Connected to 127.0.0.1 (127.0.0.1) port 9000 (#0)
> GET /headers HTTP/1.1
> Host: 127.0.0.1:9000
> ...

< HTTP/1.1 200 OK
< Connection: keep-alive
< ...
* Connection #0 to host 127.0.0.1 left intact
* Found bundle for host 127.0.0.1: 0x1400084c0 [can pipeline]
* Re-using existing connection! (#0) with host 127.0.0.1
> GET /status/204 HTTP/1.1
> Host: 127.0.0.1:9000
> ...

< HTTP/1.1 204 No Content
< Connection: keep-alive
< ...
* Connection #0 to host 127.0.0.1 left intact
```
The key lines are `Re-using existing connection!` and `Connection #0 to host 127.0.0.1 left intact`, which confirm that the second request was sent over the same connection as the first.

### Testing the Host Header

Using `netcat` (`nc`), we can manually craft an HTTP request. First, let's try one without a `Host` header.
```shell
$ printf "GET / HTTP/1.1\r\n\r\n" | nc localhost 9000
```
The command returns nothing, and on the server side, we see our error log:
```plaintext
2026/01/24 19:34:39 ERROR http error: required 'Host' header not found
```
Success! Now let's add the `Host` header.
```shell
$ printf "GET /headers HTTP/1.1\r\nHost: localhost\r\n\r\n" | nc localhost 9000
HTTP/1.1 200 OK
Connection: keep-alive
Content-Type: application/json
Transfer-Encoding: chunked

61
{"Accept-Encoding":["gzip"],"Host":["localhost"],"User-Agent":["Go-http-client/1.1"]}
0

```
It works perfectly, and we even get a chunked response back!

### Testing Chunked Encoding

Our server has an `/echo/chunked` endpoint that streams the request body right back to the response. We can use `curl` to send a chunked request to it.
```shell
# We send two chunks: "hello" and " world"
$ (
  printf "5\r\nhello\r\n";
  sleep 1;
  printf "6\r\n world\r\n";
  sleep 1;
  printf "0\r\n\r\n";
) | curl --http1.1 -X POST --header "Transfer-Encoding: chunked" -T - http://127.0.0.1:9000/echo/chunked

hello world
```
The command pipes a manually created chunked body into `curl`. `curl` sends it to our server, which reads it using `chunkedBodyReader` and writes it back using our chunked `responseBodyWriter`. The final output `hello world` confirms the whole process worked.

## Conclusion

See all of the code mentioned in this article here: {{< github-link file="go" name="full source" >}}.

HTTP/1.1 was and still is an amazing protocol. It introduced connections re-use, virtual hosting, and streaming request and response bodies. The design choices made in HTTP/1.1 were so robust that they remain deeply embedded in the internet's infrastructure today.

If HTTP/1.1 was so great, why was HTTP/2 created? And what's the deal with HTTP/3? Stay tuned for the next post in this series where we start looking at `HTTP/2`.
