---
categories: ["article"]
tags: ["networking", "grpc", "http", "go", "golang", "tutorial", "protobuf", "connectrpc"]
series: ["gRPC from Scratch"]
date: "2024-02-17"
description: "Last part we created a simple gRPC client. Let's take it a bit further. Let's implement a simple gRPC server in go."
cover: "cover.jpg"
images: ["/posts/grpc-from-scratch-part-2/social.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC From Scratch: Part 2 - Server"
slug: "grpc-from-scratch-part-2"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-from-scratch-part-2/
mastodonID: "112277285817413643"
---

Last time we made a super simple gRPC client. **This time we're going to make a gRPC server**. We are going to completely reuse the [writeMessage](/posts/grpc-from-scratch/#encoding-the-request) and [readMessage](/posts/grpc-from-scratch/#decoding-the-response) from [last time](/posts/grpc-from-scratch/) because they work the same on the server. After all, the envelope for servers is the same as the envelope for clients. Sweet!

## The Setup
Like last time, we're going to use [ConnectRPC](https://connectrpc.com/docs/go/getting-started) to help us test our implementation. Last time we used the ConnectRPC's server to test our custom gRPC client so this time we're going to use the ConnectRPC's client to test our custom gRPC server. Did I say that right? Yeah, I think so... Let's move on. Here's what the full client looks like:

{{% render-code file="go/client/main.go" language="go" %}}

And as for more setup, here's some of the more boring parts of the server:

```go
func main() {
	mux := http.NewServeMux()
	mux.Handle("/greet.v1.GreetService/Greet", http.HandlerFunc(greetHandler))
	log.Fatal(http.ListenAndServe(
		"localhost:9000",
		h2c.NewHandler(mux, &http2.Server{}),
	))
}
```
Here we create an HTTP server [with h2c so we aren't required to use TLS for these examples](https://connectrpc.com/docs/go/deployment/#h2c), mount an HTTP path for our one endpoint and start the server. The real fun happens in `greetHandler`. But before I show that, I need to talk about HTTP trailers.

## HTTP Trailers
Trailers are the same idea as headers but they happen after the response instead of before. Since gRPC is a streaming protocol it uses trailers to report on the overall status of the request instead of the HTTP status code. Go supports sending trailers. I am having a hard time coming up with a good explanation of how it works, so here's an example:

```go
import (
	"io"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/sendstrailers", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Trailer", "MessagesSent")
		w.WriteHeader(http.StatusOK)
        io.WriteString(w, "Hello world!")
		w.Header().Set("MessagesSent", "100")
	})
}
```

In this example, a trailer named `MessagesSent` will be sent AFTER the `Hello world!` body has been sent to the client. Notice how we set a header called `Trailer` at the beginning with `MessagesSent` as the value? Yeah, that's just how you do it in Go. It's [super strange](https://pkg.go.dev/net/http#example-ResponseWriter-Trailers). A more complete explanation exists in the [Go documentation](https://pkg.go.dev/net/http#ResponseWriter) where there exists a separate equally magical way to add trailers that is less preferred for an unspecified reason:

```go
	// There are two ways to set Trailers. The preferred way is to
	// predeclare in the headers which trailers you will later
	// send by setting the "Trailer" header to the names of the
	// trailer keys which will come later. In this case, those
	// keys of the Header map are treated as if they were
	// trailers. See the example. The second way, for trailer
	// keys not known to the [Handler] until after the first [ResponseWriter.Write],
	// is to prefix the [Header] map keys with the [TrailerPrefix]
	// constant value.
```

Why doesn't Go support a more... normal interface for trailers? [Who knows.](https://go-review.googlesource.com/c/go/+/2157). Trailers are often a bit of an afterthought. That is also proven by the fact that the "Trailer" values [available in Go's http.Request](https://pkg.go.dev/net/http#Request) uses the type `http.Headers`. Oof.

## Implementing the Greet Handler
As I did last time, I'm omitting error handling for clarity since `readMessage` and `writeMessage` can both return an error. The code that I end up with [does handle errors that might happen when reading or writing the HTTP body](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2024/grpc-from-scratch-part-2/go/server/main.go). However, take a look at the cleaner version first:

```go
func greetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Trailer", gRPCStatusHeader+", "+gRPCMessageHeader)
	w.Header().Set("Content-Type", "application/grpc+proto")
	w.WriteHeader(http.StatusOK)
	defer r.Body.Close()

	req := &greetv1.GreetRequest{}
	readMessage(r.Body, req)
	writeMessage(w, &greetv1.GreetResponse{
		Greeting: fmt.Sprintf("Hello, %s!", req.Name),
	})
	w.Header().Set(gRPCStatusHeader, "0")
	w.Header().Set(gRPCMessageHeader, "")
}
```

## Putting it all together
Surprisingly, this is pretty much all of the parts that are needed to handle a unary gRPC RPC. You can see [the entire working prototype here](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2024/grpc-from-scratch-part-2/go).

### Logs from the client

```text
send-> name:"World"
recv<- greeting:"Hello, World!"
```

### Logs from the server

```text
recv<- name:"World"
send-> greeting:"Hello, World!"
```

### Using buf curl

In addition to using the generating Go code we can also use tools like `buf curl` to test our server.
```shell
$ buf curl -v \
           --protocol=grpc \
           --schema=greet.proto \
           -d '{"name": "World"}' \
           --http2-prior-knowledge \
           http://127.0.0.1:9000/greet.v1.GreetService/Greet
buf: * Invoking RPC greet.v1.GreetService.Greet
buf: * Dialing (tcp) 127.0.0.1:9000...
buf: * Connected to 127.0.0.1:9000
buf: > (#1) POST /greet.v1.GreetService/Greet
buf: > (#1) Accept-Encoding: identity
buf: > (#1) Content-Type: application/grpc+proto
buf: > (#1) Grpc-Accept-Encoding: gzip
buf: > (#1) Grpc-Timeout: 119989m
buf: > (#1) Te: trailers
buf: > (#1) User-Agent: grpc-go-connect/1.14.0 (go1.21.6) buf/1.29.0
buf: > (#1)
buf: } (#1) [5 bytes data]
buf: } (#1) [7 bytes data]
buf: * (#1) Finished upload
buf: < (#1) HTTP/2.0 200 OK
buf: < (#1) Content-Length: 20
buf: < (#1) Content-Type: application/grpc+proto
buf: < (#1) Date: Sat, 17 Feb 2024 07:35:20 GMT
buf: < (#1)
buf: { (#1) [5 bytes data]
buf: { (#1) [15 bytes data]
buf: < (#1)
buf: < (#1) Grpc-Message:
buf: < (#1) Grpc-Status: 0
buf: * (#1) Call complete
{
  "greeting": "Hello, World!"
}
```

It works! The `[5 bytes data]` log lines that you see are the gRPC framing that happens before each message. The `[7 bytes data]` and `[15 bytes data]` are the request body and response body respectively. Also, notice the trailers at the end; `Grpc-Status: 0` indicates that the RPC finished successfully.

## How to improve
How can we improve the client and server that we've written? **MANY** details were glossed over when making this client/server. Here are just a small handful:

- Different encodings ([JSON](https://protobuf.dev/programming-guides/proto3/#json))
- Compression Support
- Support for multiple gRPC status codes
- Respecting the status codes at all in the client
- Interceptor support
- Compile code from protobufs
- TLS support
- Usage of the `Grpc-Timeout` header
- Retries on error if the method is marked as idempotent
- Fix a lot of implementation details
  - For example, our server does not percent-encode the error messages, which should be done according to the gRPC spec
- [gRPC-Web](https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md) support
- [ConnectRPC](https://connectrpc.com/docs/protocol/) support

There is quite a lot that goes into making a real gRPC client/server and a lot of small details that can have a large impact. Many of those details can be found in [the gRPC over HTTP2 spec](https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md). Also, if I were making a real client/server I would use a test suite designed for testing a gRPC implementation's conformity to the gRPC spec. [ConnectRPC has one](https://github.com/connectrpc/conformance). Using that paired with testing with various gRPC tools and languages while trying to use as many features is probably the best way to validate your client and gRPC server implementations.

[See the full prototype from this post here.](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2024/grpc-from-scratch-part-2/go)
