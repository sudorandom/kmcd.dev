---
categories: ["article"]
tags: ["grpc", "connectrpc", "rpc", "http3"]
date: "2024-09-17T10:00:00Z"
description: "Followup on gRPC over HTTP/3."
cover: "cover.jpg"
images: ["/posts/grpc-over-http3-followup/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC Over HTTP/3: Followup"
slug: "grpc-over-http3-followup"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-over-http3-followup/
draft: true
---

Remember that time we talked about gRPC and how it could be interesting to use HTTP/3 as the transport? Well, guess what? The future is now!

In my previous post, "[gRPC Over HTTP/3](/posts/grpc-over-http3/)," we dove into the exciting possibilities of combining these technologies. At that time, some of the pieces were missing. Specifically, the quic-go HTTP/3 implementation didn't have support for HTTP trailers. But now things have recently changed there!

### The updates you've been waiting for
* **quic-go now supports HTTP Trailers:** If you recall, this was a major roadblock for getting gRPC to work over HTTP/3. Trailers are crucial for gRPC's error handling and status codes, so this was a big deal. This works as of [v0.47.0](https://github.com/quic-go/quic-go/releases/tag/v0.47.0).
* **Buf's curl command has a new `--http3` flag:** That's right, you can now easily test your gRPC services over HTTP/3 from the command line. This is a fantastic development for quick prototyping, debugging and having a simple tool to call gRPC services. You can use this as of [v1.41.0](https://github.com/bufbuild/buf/releases/tag/v1.41.0).

I'm very happy to have contributed both of these features. Like I said in my [previous post about this topic](/posts/grpc-over-http3/), the Go version of ConnectRPC seemed so close to having full HTTP/3 support in all three protocols: Connect/gRPC-Web and the original gRPC; it just needed trailer support to push gRPC over the finish line. And with other gRPC implementations, like [grpc-dotnet](https://devblogs.microsoft.com/dotnet/http-3-support-in-dotnet-6/), I hope the addition of HTTP/3 to `buf curl` command can be useful as well.

### What does this mean for you?
In short, it means that if you're working on a gRPC project, it's now slightly more viable to use HTTP/3 *today*... in specific contexts. Here's a recap of the benefits:

* **Faster connections, especially on unreliable mobile connections:** HTTP/3's connection setup is lightning-fast compared to HTTP/2, and it handles flaky networks like a champ. This is a major win for mobile apps and any situation where network conditions aren't ideal. This can be useful when you're using gRPC-Web or Connect on the frontend.
* **No more head-of-line blocking:** HTTP/3 eliminates this pesky problem that can slow down HTTP/2 in certain scenarios. If your gRPC service handles lots of concurrent streams, you might see an improvement.

But the QUIC and HTTP/3 world isn't all roses. Let's cover why you may not want to jump on HTTP/3 quite yet.

Here's an example of starting a HTTP/3 server with ConnectRPC:
{{< highlight go >}}
func main() {
	mux := http.NewServeMux()
	// Implementation is only in the full source
	mux.Handle(elizav1connect.NewElizaServiceHandler(&server{}))

	addr := "127.0.0.1:6660"
	log.Printf("Starting connectrpc on %s", addr)
	h3srv := http3.Server{
		Addr:    addr,
		Handler: mux,
	}
	if err := h3srv.ListenAndServeTLS("cert.crt", "cert.key"); err != nil {
		log.Fatalf("error: %s", err)
	}
}
{{< / highlight >}}

{{< aside >}}
<a href="https://github.com/sudorandom/example-connect-http3/blob/v0.0.2/server-single/main.go" target="_blank">See the full source at GitHub.</a>
{{</ aside >}}

This example uses the HTTP/3 server from quic-go to provide HTTP/3. Now you can test it using `buf curl`. Here are examples with gRPC, gRPC-Web and Connect:

```shell
$ buf curl --http3 -k --schema=buf.build/connectrpc/eliza -d '{"sentence":"Hello, with gRPC+h3"}' https://127.0.0.1:6660/connectrpc.eliza.v1.ElizaService/Say --protocol=grpc
{
  "sentence": "Hello, with gRPC+h3"
}
$ buf curl --http3 -k --schema=buf.build/connectrpc/eliza -d '{"sentence":"Hello, with gRPC-Web+h3"}' https://127.0.0.1:6660/connectrpc.eliza.v1.ElizaService/Say --protocol=grpcweb
{
  "sentence": "Hello, with gRPC-Web+h3"
}
$ buf curl --http3 -k --schema=buf.build/connectrpc/eliza -d '{"sentence":"Hello, with Connect+h3"}' https://127.0.0.1:6660/connectrpc.eliza.v1.ElizaService/Say --protocol=connect
{
  "sentence": "Hello, with Connect+h3"
}
```
Note that if you don't the `--http3` flag this doesn't work. That's because we've only started an HTTP/3 server. We can run HTTP/3 alongside HTTP/1.1 and HTTP/2:

```go
func main() {
	mux := http.NewServeMux()
	mux.Handle(elizav1connect.NewElizaServiceHandler(&server{}))

	addr := "127.0.0.1:6660"
	log.Printf("Starting connectrpc on %s", addr)
	h3srv := http3.Server{
		Addr:    addr,
		Handler: mux,
	}

	srv := http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	eg, _ := errgroup.WithContext(context.Background())
	eg.Go(func() error {
		return h3srv.ListenAndServeTLS("cert.crt", "cert.key")
	})
	eg.Go(func() error {
		return srv.ListenAndServeTLS("cert.crt", "cert.key")
	})
	if err := eg.Wait(); err != nil {
		log.Fatalf("error: %s", err)
	}
}
```
{{< aside >}}
<a href="https://github.com/sudorandom/example-connect-http3/blob/v0.0.2/server-multi/main.go" target="_blank">See the full source at GitHub.</a>
{{</ aside >}}

With this code, you can now connect using any version of HTTP and with gRPC, gRPC-Web or Connect. The compatibility matrix is now all green:

```shell
$ buf curl -k --schema=buf.build/connectrpc/eliza -d '{"sentence":"Hello, with gRPC+h2"}' https://127.0.0.1:6660/connectrpc.eliza.v1.ElizaService/Say --protocol=grpc
{
  "sentence": "Hello, with gRPC+h2"
}
```

See the repo at [sudorandom/example-connect-http3](https://github.com/sudorandom/example-connect-http3/) to see the full example.

### So everything is fast with this, right?
Well, no, HTTP/3 isn't always a performance win... and actually, today, it may generally be slower or, at best, the same speed as HTTP/2. Part of the cause is that it is uses a lot of CPU cycles compared to HTTP/1.1 and HTTP/2. So this awesome protocol that is supposed to make things fast is *actually slower*? What's going on?

QUIC is still mostly implemented in user-space and is lacking the half-century of optimizations that TCP has had. I recently saw [this paper](https://dl.acm.org/doi/10.1145/3589334.3645323) which looks to be some decent data regarding actual HTTP/3 performance. Generally, it's not a good story for QUIC or HTTP/3.

Just for completeness, here are some other testimonies of the performance of HTTP/3 and QUIC:
- https://dl.acm.org/doi/10.1145/3589334.3645323 (2024)
- https://daniel.haxx.se/blog/2024/06/10/http-3-in-curl-mid-2024/comment-page-1/ (2024)
- https://www.cloudpanel.io/blog/http3-vs-http2/ (2024)
- https://tailscale.com/blog/quic-udp-throughput (2023)
- https://pulse.internetsociety.org/blog/measuring-http-3-real-world-performance (2023)
- https://pulse.internetsociety.org/blog/the-challenges-ahead-for-http-3 (2023)
- https://dropbox.tech/frontend/investigating-the-impact-of-http3-on-network-latency-for-search (2023)
- https://x.com/alonkochba/status/1424403252284694528 (2021)
- https://blog.cloudflare.com/http-3-vs-http-2/ (2020)
- https://dl.acm.org/doi/pdf/10.1145/3098822.3098842 (2017)

There is a mixed bag, but it generally indicates that the receiver end needs more optimizations. Specifically, the proposed solutions involve a technique called UDP generic receive offload (UDP GRO). Some experiments show that optimization can yield promising results.

We're still in the early stages of widespread HTTP/3 adoption for gRPC, especially on the backend. Native support across all languages and frameworks is still developing.

### However, there's reason for optimism
- The tooling is improving, making experimentation and early adoption easier.
- Performance optimizations are actively being researched and implemented.

HTTP/3 and QUIC have their niches that are pretty compelling. Specifically, HTTP/3 consistently does pretty well with reducing the number of pauses with video conferencing and video streaming while improving general web usage with slow/unstable networks, typically with mobile devices.

## Keep exploring!
I'm eager to see how the community leverages these advancements. Don't hesitate to experiment, measure performance in your own scenarios, and share your findings. The more we collaborate and learn, the faster we'll unlock the full potential of gRPC over HTTP/3.

### It's still early days, but...
While the core building blocks are in place, there's still work to be done. Full, native support for gRPC over HTTP/3 in all languages and frameworks is still a long ways off. But I feel like I've done a little part into pushing the tooling so now we at least have something to benchmark.
