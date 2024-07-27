---
categories: ["article"]
tags: ["networking", "http", "go", "golang", "tutorial"]
series: ["HTTP from Scratch"]
date: "2024-07-30"
description: "Building the foundation with HTTP/0.9"
cover: "cover.jpg"
images: ["/posts/http-from-scratch/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "HTTP From Scratch: Part 1 - HTTP/0.9"
slug: "http-from-scratch"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/http-from-scratch/
featured: false
featuredTitle: "Series: HTTP From Scratch"
draft: true
---

## Introduction
Welcome to the first installment of our "HTTP from Scratch" blog series! In this series, we'll embark on a journey through the evolution of the Hypertext Transfer Protocol (HTTP), the backbone of the World Wide Web. By building simple implementations of each major HTTP version, we'll gain a deep understanding of how this essential protocol has shaped the internet we use every day and how it has evolved to what we have now.

In this post, we'll travel back to the early days of the web and explore HTTP/0.9, HTTP's initial incarnation. HTTP/0.9 was a groundbreaking technology that enabled the first web browsers and servers to communicate, laying the foundation for the World Wide Web that we know today. But HTTP/0.9 had its limitations. We'll discuss these shortcomings, which ultimately paved the way for subsequent versions of HTTP that introduced features like headers, status codes, and support for additional HTTP methods, connection reuse, binary framing and, eventually, abandoning TCP for UDP for better reliability and performance.

To get a hands-on understanding of HTTP/0.9, we'll take a practical approach. Since no modern web servers support this early version, we'll create our own HTTP/0.9 server from scratch using Go. This will allow us to experiment with the protocol and gain valuable insights into its inner workings.

By the end of this post, you'll have a solid grasp of HTTP/0.9 and the foundation it laid for the modern web. You will be well-equipped to continue our journey through the evolution of HTTP in the upcoming articles.

## Understanding HTTP/0.9

HTTP/0.9, the inaugural version of the Hypertext Transfer Protocol, laid the groundwork for the web as we know it today. It introduced a simple request-response model that facilitated communication between web clients and servers.

### The Request

In HTTP/0.9, a client sends a single-line request to the server. This request line consists of only two components:

1. **GET Method:**  The only method supported in HTTP/0.9. It instructs the server to retrieve the specified resource.
2. **Resource Path:**  The path to the resource on the server that the client wants to access.

For example, to request the index.html file from the server's root directory, the client would send:

```
GET /index.html
```

That's... it. It can't get much simpler than that.

### The Response

Upon receiving a request, the server responds with the contents of the requested resource. This response is a simple stream of bytes, usually representing an HTML document. Notably, there are no headers in an HTTP/0.9 response or a status code that tells us if we're receiving an error page or the resource that we asked for. Since there were no status codes, there was no way to indicate if a requested resource was not found, a concept that would later be introduced with the 404 status code.

### Limitations

While revolutionary for its time, HTTP/0.9 had some significant limitations that paved the way for future improvements:

* **No Headers:**  The absence of headers meant that the server could not convey any additional information about the response, such as content type or length.
* **Only GET:** The only supported method was GET, which meant that clients could only request resources and not submit data to the server.
* **No Status Codes:**  There was no mechanism to indicate errors or other status information.

## Implementing an HTTP/0.9 Server

```go
package main

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type Server struct {
	Addr    string
	Handler http.Handler
}

func (s *Server) ServeAndListen() error {
	if s.Handler == nil {
		panic("http server started without a handler")
	}
	l, err := net.Listen("tcp", s.Addr)

	if err != nil {
		return err
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func(c net.Conn) {
			reader := bufio.NewReader(c)
			line, _, err := reader.ReadLine()
			if err != nil {
				return
			}

			fields := strings.Fields(string(line))
			if len(fields) < 2 {
				return
			}
			r := &http.Request{
				Method:     fields[0],
				URL:        &url.URL{Scheme: "http", Path: fields[1]},
				Proto:      "HTTP/0.9",
				ProtoMajor: 0,
				ProtoMinor: 9,
				RemoteAddr: c.RemoteAddr().String(),
			}

			s.Handler.ServeHTTP(newWriter(c), r)
			c.Close()
		}(conn)
	}
}

type responseBodyWriter struct {
	conn net.Conn
}

// Header implements http.ResponseWriter.
func (r *responseBodyWriter) Header() http.Header {
	panic("unsupported with HTTP/0.9")
}

// Write implements http.ResponseWriter.
// Subtle: this method shadows the method (Conn).Write of responseBodyWriter.Conn.
func (r *responseBodyWriter) Write(b []byte) (int, error) {
	return r.conn.Write(b)
}

// WriteHeader implements http.ResponseWriter.
func (r *responseBodyWriter) WriteHeader(statusCode int) {
	panic("unsupported with HTTP/0.9")
}

func newWriter(c net.Conn) http.ResponseWriter {
	return &responseBodyWriter{
		conn: c,
	}
}

func main() {
    addr := "127.0.0.1:9000"
	s := Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World!"))
		}),
	}
    log.Printf("Listing on %s", addr)
	if err := s.ServeAndListen(); err != nil {
		log.Fatal(err)
	}
}
```

Now we just need to run the server:
```shell
$ go run server/main.go
2024/07/27 21:55:28 Listing on 127.0.0.1:9000
```

Now that we have an HTTP/0.9 server, how do we test it?? Since HTTP/0.9 pretty much isn't used anywhere, how do find a client to test this server? Luckily, `curl` supports HTTP/0.9, so let's try that!
```shell
$ curl --http0.9 http://127.0.0.1:9000/this/is/a/test
Hello World!
* Closing connection 0
```

You should note that curl is *sending* a request as if it were HTTP/1.1 but appears to treat the body correctly. That's confusing because I told curl to use HTTP/0.9! The `--http0.9` flag impacts how the response is treated. Here's what the curl manpage says about the flag:

```shell
       --http0.9
              (HTTP) Tells curl to be fine with HTTP version 0.9 response.

              HTTP/0.9 is a completely headerless response and therefore you can also connect with this to non-HTTP servers and still get a response since curl will simply transparently downgrade - if allowed.
```

I verified this behavior a bit more when I mixed `--http1.0` and `--http0.9`. Instead of using `HTTP/1.1` in the first line of the request, curl declares that it is using `HTTP/1.0` while still treating the body correctly (not expecting a status code or headers):

```shell
$ curl --http1.0 --http0.9 http://127.0.0.1:9000/this/is/a/test
Hello World!
* Closing connection 0
```

By the way, if you don't use the `--http0.9` (or if your server returns the status code/headers in a format that doesn't make sense) you will receive an error message, "Received HTTP/0.9 when not allowed":
```shell
$ curl http://127.0.0.1:9000/this/is/a/test
curl: (1) Received HTTP/0.9 when not allowed
```

You can test this server with simpler tools. For example, netcat (`ncat`), can be used to make this same request. This will be a theme for the text-based protocols where often it's simpler just to write text out directly to netcat than it is to use other kinds of tooling:

```shell
$ echo GET this/is/a/test | ncat 127.0.0.1 9000
Hello World!
```

## Implementing an HTTP/0.9 Client

Now that we've made a server and used existing clients, we might as well make a client in Go as well. Don't worry, this one is super simple:

```go
package main

import (
	"fmt"
	"io"
	"log"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		log.Fatalf("err: %s", err)
	}

	if _, err := conn.Write([]byte("GET /this/is/a/test\r\n")); err != nil {
		log.Fatalf("err: %s", err)
	}

	body, err := io.ReadAll(conn)
	if err != nil {
		log.Fatalf("err: %s", err)
	}

	fmt.Println(string(body))
}
```

This code does the following:

- Establishes a TCP connection to the server.
- Sends the HTTP/0.9 request.
- Receives the response and displays it to the user.

Simple, right? Almost ***too simple***.

```go
$ go run client/main.go
Hello World!
```

## Conclusion

In this post, we delved into the origins of the web by exploring HTTP/0.9. While simple in its design, HTTP/0.9 was a groundbreaking protocol that enabled the first web browsers and servers to communicate.

We explored the basic request-response model of HTTP/0.9, understanding how clients could request resources and servers could deliver them. We also acknowledged the limitations of this early version, including the lack of headers, status codes, and support for other HTTP methods besides GET.

By building a rudimentary HTTP/0.9 server and client in Go, we gained hands-on experience with the protocol's core concepts. We learned how to handle TCP connections, parse requests, and send responses, laying a solid foundation for understanding more advanced HTTP versions.

As we move forward in this series, we'll see how HTTP evolved to address the shortcomings of HTTP/0.9. We'll explore the introduction of headers, status codes, and additional methods, which enabled more robust and feature-rich communication between web clients and servers. Stay tuned for the next part of our series, where we'll dive into HTTP/1.0 and its significant enhancements over HTTP/0.9. As a sneak peek, these are the major features added to each version:

- HTTP/0.9: First attempt to transfer generic resources by paths
- HTTP/1.0: Adds status codes, headers, verbs
- HTTP/1.1: Adds connection re-use so connections can be reused
- HTTP/2: Switches from a text-based protocol to binary
- HTTP/3: Built on QUIC instead of TCP

Resources:
- https://www.w3.org/Protocols/HTTP/AsImplemented.html
- https://ec.haxx.se/http/versions/http09.html
