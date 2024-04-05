---
categories: ["article"]
tags: ["connectrpc", "grpc", "tutorial", "golang"]
date: "2024-03-05"
description: "Unleash the power of gRPC: ConnectRPC breaks down barriers, enabling frictionless communication between gRPC, gRPC-Web, and any HTTP client."
cover: "cover.jpg"
images: ["/posts/connectrpc/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Making gRPC more approachable with ConnectRPC"
url: "/posts/connectrpc"
slug: "connectrpc"
type: "posts"
devtoId: 1797651
devtoPublished: true
devtoSkip: false
canonical_url: https://sudorandom.dev/posts/connectrpc
---

## Introduction

gRPC is an open-source framework for building high-performance applications that communicate with each other. gRPC is language-neutral, meaning clients and servers can be written in different programming languages, and it offers features like authentication, streaming, and load balancing. Also, because it is built using protocolbuffers it gives an amazing way to define contracts in a much more clear way than bolting on swagger to your API after the fact. However, gRPC has some limitations that restrict its usage. It requires special gRPC clients and support for HTTP/2 (which still lacking in some areas), and it doesn't work from Javascript.

Support for Javascript has been solved in a couple of ways. Let's discuss the two approaches!

## gRPC-Web
gRPC-Web is a [variant of gRPC](https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md) and [gRPC client for Javascript](https://github.com/grpc/grpc-web) that avoids HTTP/2-specific features like [HTTP trailers](https://carlmastrangelo.com/blog/why-does-grpc-insist-on-trailers). This is an incredibly practical solution to the problem. And it does work really well, but I still have my reservations on some of the implementation details.

gRPC-Web doesn't fix the "this doesn't look like any HTTP API I've ever seen" issue that I have with gRPC in general. In other words, I want to be able to send a normal cURL example to someone. gRPC-Web doesn't work for that without special gRPC-specific clients or tooling.

Additionally, I don't like how gRPC-Web is typically deployed. You usually are forced into a proxy that can convert gRPC into gRPC-Web. Instead, I prefer the gRPC-Web implementation to sit alongside the actual gRPC server. The protocol isn't so different from the normal gRPC version so it shouldn't be too much work to add support to existing gRPC server implementations. I know that the popular gRPC library [tonic](https://docs.rs/tonic/latest/tonic/) supports [gRPC-Web](https://docs.rs/tonic-web/latest/tonic_web/) out of the box.

## gRPC Transcoding
The idea with transcoding is to annotate your protobuf service methods with HTTP verbs and path patterns that can map a more REST-like API to gRPC. Many solutions allow you to provide a separate config file so you aren't required to have the HTTP annotations making a mess of your protobuf files. [Google has a service](https://cloud.google.com/endpoints/docs/grpc/transcoding) that can use this mapping and provide a REST-like API on top of your gRPC service. and several gRPC proxies can do this kind of transcoding as well like [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway) and [envoy](https://www.envoyproxy.io).

Here's a simple version of what the annotations can look like:
```protobuf
syntax = "proto3";

package status.v1;

import "google/protobuf/empty.proto";
import "google/api/annotations.proto";
import "google/rpc/status.proto";

service Status {
  rpc GetStatus(google.protobuf.Empty) returns (google.rpc.Status) {
    option(google.api.http) = {
        get: "/v1/status"
    };
  }
}
```
You can see how `status.v1.Status.GetStatus` maps to `GET /v1/status`. Thanks, protobuf options!

However, most ways of deploying this in Go weirdly use proxies, creating a new network hop and a decoding/encoding step. Additionally, transcoding ruins a lot of the benefits you have from a contract-based interface that gRPC provides. Generated clients generally can't interpret the `google.api.http` options so if you want to keep to a contract-based model you have to rely on converting your protobuf file to OpenAPI and generating clients based on that. I don't generally prefer this method because it adds extra complexity. However, it can be a really good way to support existing APIs by "swapping out" traditional HTTP handles with gRPC.

## ConnectRPC
Let me introduce [ConnectRPC](https://connectrpc.com/). I believe it elegantly solves all of the issues I have with the gRPC ecosystem. ConnectRPC is a series of libraries for building browser and gRPC-compatible APIs. With ConnectRPC as the server, you get support for three protocols: gRPC, gRPC-Web and [the so-called "Connect" protocol](https://connectrpc.com/docs/protocol/). These three protocols are [all served from a single ConnectRPC server](https://connectrpc.com/docs/multi-protocol) by simply using the HTTP content-type header, which gRPC and gRPC-Web clients already send. Let me break down where you might use each protocol:

- Microservice communication: Connect or gRPC
- Publically exposed API: Connect, gRPC or gRPC-Web depending on client language support
- Clients running in environments without HTTP/2 support: Connect or gRPC-Web
- Sending simple API examples to colleagues: Connect

Hopefully, I convinced you to at least give ConnectRPC a try. Now, get started with it! Here are the [getting started docs for Go](https://connectrpc.com/docs/go/getting-started). It does a great job at detailing how you make a Go server and client implemented with ConnectRPC.

## Multiple Protocol Support
So, ConnectRPC is running *three* different protocols? Isn't that overkill? You can make everything work with the Connect protocol, but there are benefits to supporting all three.

We can use the gRPC protocol to call into this ConnectRPC service using normal gRPC tooling like [grpcurl](https://github.com/fullstorydev/grpcurl):
```shell
$ grpcurl -plaintext \
          -proto greet/v1/greet.proto \
          -d '{"name": "Jane"}' \
          127.0.0.1:8080 \
          greet.v1.GreetService.Greet
{
  "greeting": "Hello, Jane!"
}
```

If you have the [reflection API enabled](https://github.com/connectrpc/grpcreflect-go), you can omit the -proto option.

```shell
$ grpcurl -plaintext \
          -d '{"name": "Jane"}' \
          127.0.0.1:8080 \
          greet.v1.GreetService.Greet
{
  "greeting": "Hello, Jane!"
}
```

Now here's where the magic is. In addition to gRPC-specific tooling, I can also use generic HTTP tools with ConnectRPC servers, like [curl](https://curl.se/):
```shell
$ curl -XPOST \
       -H"Content-Type: application/json" \
       -d '{"name": "Jane"}' \
       "http://127.0.0.1:8080/greet.v1.GreetService/Greet"
```
This shows how the "I want to be able to send a normal cURL example to someone" desire from above is completely fulfilled.

I also want to point out that there is also the `buf curl` command which is a CLI tool that allows you to call ConnectRPC services using all three protocols (gRPC, gRPC-Web, Connect).

```shell
$ buf curl --http2-prior-knowledge \
           -d '{"name": "Jane"}' \
           http://127.0.0.1:8080/greet.v1.GreetService/Greet
{
  "greeting": "Hello, Jane!"
}
```
This last command actually uses the Connect protocol. We can pass in `--protocol` to use the gRPC-Web or the gRPC protocol instead:
```bash
$ buf curl --http2-prior-knowledge \
           -d '{"name": "Jane"}' \
           http://127.0.0.1:8080/greet.v1.GreetService/Greet \
           --protocol=grpcweb
{
  "greeting": "Hello, Jane!"
}
$ buf curl --http2-prior-knowledge \
           -d '{"name": "Jane"}' \
           http://127.0.0.1:8080/greet.v1.GreetService/Greet \
           --protocol=grpc
{
  "greeting": "Hello, Jane!"
}
```

### Digging Deeper (optional)
This is an optional section where we will dig deeper into what is happening here with this last command to show what is happening under the hood. We will see how server reflection works with gRPC and a couple uses of it. We can see more details of the previous `buf curl` command by adding `-v` at the end:
```shell
$ buf curl --http2-prior-knowledge \
           -d '{"name": "Jane"}' \
           http://127.0.0.1:8080/greet.v1.GreetService/Greet -v
buf: * Using server reflection to resolve "greet.v1.GreetService"
buf: * Dialing (tcp) 127.0.0.1:8080...
buf: * Connected to 127.0.0.1:8080
buf: > (#1) POST /grpc.reflection.v1.ServerReflection/ServerReflectionInfo
buf: > (#1) Accept-Encoding: identity
buf: > (#1) Content-Type: application/grpc+proto
buf: > (#1) Grpc-Accept-Encoding: gzip
buf: > (#1) Grpc-Timeout: 119999m
buf: > (#1) Te: trailers
buf: > (#1) User-Agent: grpc-go-connect/1.14.0 (go1.21.6) buf/1.29.0
buf: > (#1)
buf: } (#1) [5 bytes data]
buf: } (#1) [23 bytes data]
buf: < (#1) HTTP/2.0 200 OK
buf: < (#1) Content-Type: application/grpc+proto
buf: < (#1) Date: Sat, 02 Mar 2024 06:41:48 GMT
buf: < (#1) Grpc-Accept-Encoding: gzip
buf: < (#1) Grpc-Encoding: gzip
buf: < (#1)
buf: { (#1) [5 bytes data]
buf: { (#1) [244 bytes data]
buf: * Server reflection has resolved file "greet/v1/greet.proto"
buf: * Invoking RPC greet.v1.GreetService.Greet
buf: > (#2) POST /greet.v1.GreetService/Greet
buf: > (#2) Accept-Encoding: identity
buf: > (#2) Content-Type: application/grpc+proto
buf: > (#2) Grpc-Accept-Encoding: gzip
buf: > (#2) Grpc-Timeout: 119994m
buf: > (#2) Te: trailers
buf: > (#2) User-Agent: grpc-go-connect/1.14.0 (go1.21.6) buf/1.29.0
buf: > (#2)
buf: } (#2) [5 bytes data]
buf: } (#2) [6 bytes data]
buf: * (#2) Finished upload
buf: < (#2) HTTP/2.0 200 OK
buf: < (#2) Content-Type: application/grpc+proto
buf: < (#2) Date: Sat, 02 Mar 2024 06:41:48 GMT
buf: < (#2) Greet-Version: v1
buf: < (#2) Grpc-Accept-Encoding: gzip
buf: < (#2) Grpc-Encoding: gzip
buf: < (#2)
buf: { (#2) [5 bytes data]
buf: { (#2) [38 bytes data]
buf: < (#2)
buf: < (#2) Grpc-Message:
buf: < (#2) Grpc-Status: 0
buf: * (#2) Call complete
{
  "greeting": "Hello, Jane!"
}
buf: < (#1)
buf: < (#1) Grpc-Message:
buf: < (#1) Grpc-Status: 0
buf: * (#1) Call complete
```

This is a little overwhelming at first so I will break down the two different calls that are being made here. Because we aren't passing the protobuf file (or [descriptors](https://protobuf.com/docs/descriptors)) as the `--schema` option we are missing some information needed to generate the protobuf messages from the given JSON string. So that's the first call that `buf curl` will make, using the [Server Reflection](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md) API:
```shell
buf: > (#1) POST /grpc.reflection.v1.ServerReflection/ServerReflectionInfo
buf: > (#1) Accept-Encoding: identity
buf: > (#1) Connect-Accept-Encoding: gzip
buf: > (#1) Connect-Protocol-Version: 1
buf: > (#1) Connect-Timeout-Ms: 119999
buf: > (#1) Content-Type: application/connect+proto
buf: > (#1) User-Agent: connect-go/1.14.0 (go1.21.6) buf/1.29.0
buf: > (#1)
buf: } (#1) [5 bytes data]
buf: } (#1) [23 bytes data]
buf: < (#1) HTTP/2.0 200 OK
buf: < (#1) Connect-Accept-Encoding: gzip
buf: < (#1) Connect-Content-Encoding: gzip
buf: < (#1) Content-Type: application/connect+proto
buf: < (#1) Date: Sat, 02 Mar 2024 06:33:57 GMT
buf: < (#1)
buf: { (#1) [5 bytes data]
buf: { (#1) [244 bytes data]
buf: * Server reflection has resolved file "greet/v1/greet.proto"
```
It acquired a version of the `greet.proto` file from the server. Now our client has everything it needs to make the actual request where it calls the actual service.

With HTTP, that service lives at `POST /greet.v1.GreetService/Greet`. Here's what the call looks like:
```shell
buf: * Invoking RPC greet.v1.GreetService.Greet
buf: > (#2) POST /greet.v1.GreetService/Greet
buf: > (#2) Accept-Encoding: identity
buf: > (#2) Content-Type: application/grpc+proto
buf: > (#2) Grpc-Accept-Encoding: gzip
buf: > (#2) Grpc-Timeout: 119994m
buf: > (#2) Te: trailers
buf: > (#2) User-Agent: grpc-go-connect/1.14.0 (go1.21.6) buf/1.29.0
buf: > (#2)
buf: } (#2) [5 bytes data]
buf: } (#2) [6 bytes data]
buf: * (#2) Finished upload
buf: < (#2) HTTP/2.0 200 OK
buf: < (#2) Content-Type: application/grpc+proto
buf: < (#2) Date: Sat, 02 Mar 2024 06:41:48 GMT
buf: < (#2) Greet-Version: v1
buf: < (#2) Grpc-Accept-Encoding: gzip
buf: < (#2) Grpc-Encoding: gzip
buf: < (#2)
buf: { (#2) [5 bytes data]
buf: { (#2) [38 bytes data]
buf: < (#2)
buf: < (#2) Grpc-Message:
buf: < (#2) Grpc-Status: 0
buf: * (#2) Call complete
{
  "greeting": "Hello, Jane!"
}
buf: < (#1)
buf: < (#1) Grpc-Message:
buf: < (#1) Grpc-Status: 0
buf: * (#1) Call complete
```

I wanted to show you how the server reflection works because it is a really good strength of gRPC. You can expose an API and have it be completely discoverable. Tools can automatically use this discovery mechanism... but so can humans. Look at these commands with `grpcurl`:

```shell
$ grpcurl -plaintext 127.0.0.1:8080 list
greet.v1.GreetService

$ grpcurl -plaintext 127.0.0.1:8080 describe
greet.v1.GreetService is a service:
service GreetService {
  rpc Greet ( .greet.v1.GreetRequest ) returns ( .greet.v1.GreetResponse );
}

$ grpcurl -plaintext 127.0.0.1:8080 describe .greet.v1.GreetRequest
greet.v1.GreetRequest is a message:
message GreetRequest {
  string name = 1;
}
```
We can see all the available services using the `list` and `describe` commands. And if you pass an object to the `describe` command you can dig down into message definitions. Protobuf works well here as the API contract for our services.

## Conclusion

ConnectRPC offers a compelling solution for building gRPC servers that support multiple protocols. It seamlessly integrates gRPC, gRPC-Web, and the Connect protocol, allowing clients written in various languages and environments to interact with your service.  

Here are the key takeaways:

* ConnectRPC eliminates the limitations of traditional gRPC by supporting HTTP/1.1 and Javascript environments.
* It provides a familiar REST-like API through the Connect protocol while still leveraging the benefits of Protobuf for contracts.
* Multiple protocol support with a single server simplifies deployment and reduces complexity.
* Existing gRPC tooling can still be used for server reflection and making gRPC requests.
* The Connect protocol itself can be used with generic tools like curl, making API exploration a breeze.

If you're looking for a flexible and future-proof way to build gRPC APIs, ConnectRPC is definitely worth considering.  I highly recommend checking out the [getting started guide for Go](https://connectrpc.com/docs/go/getting-started) for a hands-on approach to using ConnectRPC.
