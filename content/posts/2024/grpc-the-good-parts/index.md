---
categories: ["article"]
tags: ["grpc", "protobuf", "api", "rpc", "webdev", "humor", "http2", "http3"]
date: "2024-07-09"
description: "gRPC isn't perfect but who is?"
cover: "cover.jpg"
images: ["/posts/grpc-the-good-parts/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC: The Good Parts"
slug: "grpc-the-good-parts"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-the-good-parts
draft: true
---

gRPC is rapidly gaining traction as a compelling alternative to traditional REST APIs. Let's dive into the key advantages that make gRPC a powerful tool for modern web development.

## Performance

Is the protobuf encoding the most performant serialization ever? No, of course not. Is it *way* more efficient than JSON or XML? Absolutely. The binary format of Protocol Buffers (protobufs) means smaller payloads and faster transmission, making gRPC a speed demon for many applications.

## Strongly Typed Contracts

Say goodbye to the guesswork of loosely typed APIs. gRPC's protobuf definitions create rock-solid contracts between client and server. This translates to:

* **Fewer errors:** Clear expectations for data types reduce the chance of mismatched data.
* **Better code generation:** Automatic generation of client and server code in various languages saves time and effort.
* **Smoother development cycles:** Consistent contracts make it easier to evolve your API without breaking existing clients.
* **Generated Documentation:** Automatic generation of documentation means that your documentation will never be out of sync with your API.

## Streaming Support

Need real-time data? gRPC's got you covered. Bidirectional streaming allows for continuous data flow, making it ideal for:

* **Chat applications:** Seamlessly handle messages flowing back and forth.
* **Live updates:** Push updates to clients as soon as they happen.
* **Any scenario where constant communication is key:** From gaming to financial data, gRPC's streaming capabilities open up a world of possibilities.

If you come from the networking world, you might know that gNMI (which is based on gRPC) is the replacement for SNMP. Instead of polling network devices for the same data every minute, you can now use gNMI to subscribe. I've written more about this in a post called [Why you should use gNMI over SNMP in 2024](https://kmcd.dev/posts/gnmi/).

**Cross-Language Support**

gRPC doesn't care what programming language you prefer. Thanks to code generation tools, you can seamlessly work with gRPC in a wide range of languages, including:

* Go
* Rust
* Java
* Python
* C#
* Node.js
* Ruby
* ...and many more!

This promotes flexibility, collaboration, and the ability to choose the right tool for the job.

## Pioneered HTTP/2

gRPC was a driving force behind the adoption of HTTP/2, a major upgrade to the web's underlying protocol. This means you get all the benefits of HTTP/2's:

* **Multiplexing:** Multiple requests and responses can share a single connection, improving efficiency.
* **Header compression:** Smaller headers mean faster transmission.
* **Overall performance improvements:** HTTP/2 is simply a faster, more efficient way to communicate over the web.

### HTTP/3
There's some movement on HTTP/3 support for gRPC. There is an [open proposal](https://github.com/grpc/proposal/blob/master/G2-http3-protocol.md) created by the dotnet gRPC library maintainers and there is [an open issue to discuss actually adding HTTP/3 to the gRPC spec](https://github.com/grpc/grpc/issues/19126). Frustratingly, there hasn't been a lot of movement on the official gRPC repo to add support directly to any of their implementations, but as you can see from the thread, there's a lot of interest and a lot of people making prototypes that prove the concept.

This is likely an incomplete list but here are the packages that you can likely use HTTP/3 with today:
- The standard grpc library for C#, dotnet-grpc [(ref)](https://devblogs.microsoft.com/dotnet/http-3-support-in-dotnet-6/#grpc-with-http-3)
- It may already be possible in rust with Tonic with the Hyper http transport [(ref)](https://github.com/hyperium/tonic/issues/339)
- It's possible in Go if you use [ConnectRPC](https://connectrpc.com/) with [quic-go](https://github.com/quic-go/quic-go) - I don't have a link for this, but I've tested this out myself. This is a topic for a future post!
- This is untested but I believe many gRPC-Web implementations in the browser might "just work" with HTTP/3 as well as long as the browsers are informed of the support via the [ALT-SVC header](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Alt-Svc) and the servers support it.

As more servers and clients support HTTP/3 they should see faster connection establishment times, complete removal of the [head-of-line blocking problem](https://blog.cloudflare.com/the-road-to-quic#headoflineblocking) and much better recovery from packet loss. There's a long way to go here, but there is progress.

## The Future is Now

If you're looking to gradually adopt gRPC or need to support existing REST clients, there are several options available now!

### JSON/HTTP Transcoding
Tools like [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway), [Google Cloud Endpoints](https://cloud.google.com/endpoints) and [Envoy](https://www.envoyproxy.io/) can expose REST-like interfaces while still reaping the benefits of gRPC on the backend. You can define a service that looks like this:
```protobuf
syntax = "proto3";
package your.service.v1;
option go_package = "github.com/yourorg/yourprotos/gen/go/your/service/v1";

import "google/api/annotations.proto";

message StringMessage {
  string value = 1;
}

service YourService {
  rpc Echo(StringMessage) returns (StringMessage) {
    option (google.api.http) = {
      post: "/v1/example/echo"
      body: "*"
    };
  }
}
```

And get a REST-line endpoint where you can make this request:
```
curl -XPOST '{"value": "my value!"}' http://localhost/v1/example/echo
```

This is pretty amazing because it's doing a lot of the hard work for you and you can now support many different REST APIs without writing any additional code. This is a simple example here but there are many options, like being able to populate message fields from components of the path.

### gRPC-Web
One of the big limitations of gRPC is that it doesn't work on the web with web browsers due to limited support of HTTP trailers. Maybe browsers will support receiving trailers but there isn't yet a way to retrieve those trailers from javascript. Yes, this is incredibly frustrating, especially since there are many small use cases where trailer support would be amazing to have.

For example, I imagine using trailers to return MD5, SHA1, or other kinds of hashes after a file upload. Right now we force clients to compute this hash before sending the file but if the server returns the hash that it is computing as the upload progresses then the client can compare this against the hash that it's also computing while uploading to ensure that the file uploaded properly. Is this the best way of doing this? I don't know, but trailer support would enable some unique optimization opportunities that we don't have today.

Either way, the gRPC-Web protocol for browsers to finally use gRPC, which drastically improves the story of contract-based services in gRPC. It also allows for HTTP/1.1 clients to work with gRPC. Some platforms (I'm looking at you, Unity) still don't support HTTP/2, even though it's 2024 and the `HTTP/2` spec was created nearly a decade ago.

### ConnectRPC
[ConnectRPC](https://connectrpc.com/) automatically generates JSON/HTTP APIs from your gRPC definitions while also maintaining compatibility with gRPC and gRPC-Web. This HTTP protocol, [called Connect](https://connectrpc.com/docs/protocol/), follows HTTP standards more closely. For example, the `Content-Coding` header, `Content-Length` header, HTTP status codes, etc. all work as expected for unary RPC calls. That means you can run this normal-looking curl command and talk to a gRPC service:

```protobuf
curl --header "Content-Type: application/json" \
    --data '{"sentence": "I feel happy."}' \
    https://demo.connectrpc.com/connectrpc.eliza.v1.ElizaService/Say
```

### TWiRP
[Twirp](https://twitchtv.github.io/twirp/) is very similar to ConnectRPC. It was developed by Twitch, and is another framework that can help bridge the gap between gRPC and REST. [Twirp's approach](https://twitchtv.github.io/twirp/docs/spec_v7.html) is to use protobufs to generate an alternative protocol that also aligns more with HTTP conventions. It doesn't also support gRPC and gRPC-Web. Implementing those alongside twirp is left as an exercise for the user.

## Tooling
I have mentioned that gRPC tooling isn't that great. I still agree with that if we're talking about the "out of the box" tooling from the gRPC project. However, the community is much bigger than the gRPC Authors and someone finally made the protobuf code generation a lot better.

### Buf CLI
[Buf](https://buf.build/) (the company) has made a client called "Buf CLI", which I'm going to refer to as just "buf" from here on out. Buf completely replaces [`protoc`](https://grpc.io/docs/protoc-installation/), so it can take protobuf files and generate code, documentation, and other kinds of output using the same protoc plugins that protoc uses. How is it different? It adds a set of config files for defining the structure of your protobuf, including external dependencies. Instead of random makefile directives or bash scripts, we now have a well-defined config file for defining how the protobuf is built, which is amazing. In a similar vein, `buf curl` replaces [grpcurl](https://github.com/fullstorydev/grpcurl).

In addition to replacing existing tooling with easier-to-use versions, buf also implements some extremely useful functions. I first started using buf by using `buf lint`, which helps enforce some [common rules and practices](https://buf.build/docs/lint/rules) that developers should follow when making their protobuf files. Soon after, I started using `buf breaking` which will report on breaking changes being made to protobuf files that may break clients. Both have easy-to-use Github actions and were pretty painless to set up.

Adding buf into the mix can greatly improve your workflow with protobufs, especially when working in a larger team or working with other teams.

### Third-party protoc plugins
There's so many plugins now. I even [made one](https://github.com/sudorandom/protoc-gen-connect-openapi)! I think this really is the way that API development should work where you base API services off of a contract and generate everything from that same contract. No typos. No confusion over what methods exist. No arguing over REST semantics that has never been clear to anyone.

## Conclusion
gRPC offers a compelling set of advantages for modern web development. Its performance, strong typing, streaming capabilities, cross-language support, and HTTP/2 foundation make it a powerful tool for building efficient and scalable APIs. With various adoption options available, you can gradually incorporate gRPC into your projects and experience its benefits firsthand.
