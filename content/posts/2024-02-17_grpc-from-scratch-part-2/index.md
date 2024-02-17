+++
categories = ["article"]
tags = ["networking", "grpc", "http", "http/2", "go", "golang", "turotial", "protobuf"]
date = "2024-02-17"
description = "Last part we created a simple gRPC client. Let's take it a bit further. Let's implement a simple gRPC server in go."
cover = "cover.jpg"
images = ["/posts/grpc-from-scratch-part-2/social.jpg"]
featured = ""
featuredalt = ""
featuredpath = "date"
linktitle = ""
title = "gRPC From Scratch: Part 2"
slug = "grpc-from-scratch-part-2"
type = "posts"
+++

> This is part two of a series. [Click here to see gRPC From Scratch: Part 1 where I build a simple gRPC client. >>](/posts/grpc-from-scratch)

Last time we made a super simple gRPC client. This time we're going to make a gRPC server. We are going to completely reuse the [writeMessage](/posts/grpc-from-scratch#encoding-the-request) and [readMessage](/posts/grpc-from-scratch#decoding-the-response) from [last time](https://sudorandom.dev/posts/grpc-from-scratch/) because they work the same on the server. After all, the envelope for servers is the same as the envelope for clients. Sweet!

## The Setup
Like last time, we're going to use ConnectRPC to help us test our implementation. Last time we used the ConnectRPC server to test our custom gRPC client so this time we're going to use the ConnectRPC client to test our custom gRPC server. Here's what the full client looks like:

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
Trailers are the same idea as headers but they happen after the response instead of before. Since gRPC is a streaming protocol it uses trailers to report on the overall status of the request over the HTTP status code. Go supports sending trailers but I am having a hard time coming up with a good explanation of how it works, so here's an example. Why doesn't Go support a normal interface for trailers? Who knows.

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

## Implementing the Greet Handler
As I did last time, I'm omitting error handling for clarity since `readMessage` and `writeMessage` can both return an error. The code that I end up with [does handle errors that might happen when reading or writing the HTTP body](https://github.com/sudorandom/sudorandom.dev/tree/main/content/posts/2024-02-17_grpc-from-scratch-part-2/go/server/main.go). However, take a look at the cleaner version first:

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
Surprisingly, this is pretty much all of the parts that are needed to handle a unary gRPC RPC. You can see [the entire working prototype here](https://github.com/sudorandom/sudorandom.dev/tree/main/content/posts/2024-02-17_grpc-from-scratch-part-2/go).

**Logs from the client**

```text
send-> name:"World"
recv<- greeting:"Hello, World!"
```

**Logs from the server**

```text
recv<- name:"World"
send-> greeting:"Hello, World!"
```

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

[See the full prototype from this post here.](https://github.com/sudorandom/sudorandom.dev/tree/main/content/posts/2024-02-17_grpc-from-scratch-part-2/go)
