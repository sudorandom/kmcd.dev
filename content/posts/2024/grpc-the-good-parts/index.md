---
categories: ["article"]
tags: ["grpc", "protobuf", "api", "rpc", "webdev", "http2"]
series: ["gRPC: the good and the bad"]
date: "2024-07-02"
description: "Not perfect, but still pretty awesome."
cover: "cover.png"
images: ["/posts/grpc-the-good-parts/cover.png"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC: The Good Parts"
slug: "grpc-the-good-parts"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-the-good-parts/
mastodonID: "112715146738324384"
---

While REST APIs remain a popular choice for building web services, gRPC is increasingly being adopted for its unique advantages in performance, efficiency, and developer experience. You may have seen my post, [gRPC: The Bad Parts](/posts/grpc-the-bad-parts/), where I talk about some of my issues with gRPC. Based on the many comments about that article, I could easily write a sequel with even more complaints. However, today I'm going to focus on the *good* parts of gRPC. It has become obvious to me that many people didn't read the ending of the last post which attempted to outline how many of the points that I made are no longer true. So I figured that I need to give the positive aspects of gRPC a dedicated post.

Let's dive into the key advantages that make gRPC a powerful tool for modern web development.

## Performance
This one might be a little controversial, but [Protocol Buffers](https://protobuf.dev/) is, indeed, faster than JSON and XML. This continues [to be demonstrated](https://streamdal.com/blog/ptotobuf-vs-json-for-your-event-driven-architecture/) over and over again. Protobuf is able to be faster for these reasons:
- Field names are not included in the message. Instead, protobuf uses numbers to distinguish fields. In most cases you'll see field numbers take one or two bytes on the wire when it can be much more than that depending on your JSON field names.
- Protobuf's `VARINT` type allows for small scale integers to take up a single byte, even if it's an int64. Realistically we really don't use that many large numbers so these savings can add up. Again, it's much better than ASCII-encoded numbers being used for each digit.
- There's no real winning with strings or byte arrays but compression is still supported with gRPC so at worst this aspect is even with HTTP/JSON.

I've personally seen 50% data transfer savings by switching to protobuf encoding with realistic payloads.

There are [some haters](https://reasonablypolymorphic.com/blog/protos-are-wrong/) of protobuf encoding, and that's perfectly fine. The only "fatal flaw" that I've actually been annoyed with is "map values cannot be other maps." It does seem like that should be possible, even when you consider the internal representation of a map:

```protobuf
map<key_type, value_type> map_field = N;
```

transforms into:

```protobuf
message MapFieldEntry {
  key_type key = 1;
  value_type value = 2;
}

repeated MapFieldEntry map_field = N;
```

It's super frustrating because I don't understand why `value_type` can't be a map. The solution to this problem is just to make your own wrapper type to use as the value that contains a map. It is kind of annoying and this does come up semi-often. Crap, this was supposed to be a positive article. Let's get back on track.

I think the protobuf encoding is better than JSON in many ways. However, I understand that sometimes you just want JSON, and with [gRPC you absolutely can just use JSON](https://protobuf.dev/programming-guides/proto3/#json). gRPC still has a few binary framing bytes before each message that won't be human readable but if you're really concerned with those check out the [ConnectRPC](/posts/grpc-the-good-parts/#connectrpc) section below.

Most gRPC implementations also let you define your own encoding, so it may possible to insert your own favorite encoding if you want to push the limits.

## Strongly Typed Contracts
Say goodbye to the guesswork of loosely typed APIs. gRPC's protobuf definitions create rock-solid contracts between client and server. This translates to:

* **Fewer errors:** Clear expectations for data types reduce the chance of mismatched data.
* **Better code generation:** Automatic generation of client and server code in various languages saves time and effort.
* **Smoother development cycles:** Consistent contracts make it easier to evolve your API without breaking existing clients.
* **Generated Documentation:** Automatic generation of documentation means that your documentation will never be out of sync with your API.

API contracts are very powerful. For more on this topic, I've written an article discussing API contracts called [Building APIs with Contracts](/posts/api-contracts/).

## Streaming Support
Streaming support is arguably the best and most unique feature for gRPC. It does away with needing to frequently poll for updates in many scenarios which make it a good candidate for:

* **Chat applications:** Seamlessly handle messages flowing back and forth.
* **Live updates:** Push updates to clients as soon as they happen.
* **Any scenario where constant communication is key:** From gaming to financial data, gRPC's streaming capabilities open up a world of possibilities.

If you come from the networking world, you might know that gNMI (which is based on gRPC) is the replacement for SNMP. Instead of polling network devices for the same data every minute, you can now use gNMI to subscribe to counters. I've written more about this in a post called [Why you should use gNMI over SNMP in 2024](/posts/gnmi/).

## Cross-Language Support
gRPC doesn't care what programming language you prefer. Thanks to code generation tools, you can seamlessly work with gRPC in a wide range of languages, including:

* Go
* Rust
* Java
* Python
* C#
* Node.js
* Ruby
* ...and many more!

This promotes flexibility, collaboration, and the ability to choose the right tool for the job. Currently, I believe gRPC has such a large momentum with language support that it's hard for me to consider an alternative that doesn't also speak gRPC. Many alternatives that have similar benefits to gRPC have mediocre support for a handful of languages at best.

## Pioneered HTTP/2
gRPC was a driving force behind the adoption of HTTP/2, a major upgrade to the web's underlying protocol. This means you get all the benefits of HTTP/2's:

* **Multiplexing:** Multiple requests and responses can share a single connection, improving efficiency.
* **Header compression:** Smaller headers mean faster transmission.
* **Overall performance improvements:** HTTP/2 is simply a faster, more efficient way to communicate over the web.

### HTTP/3
There's some movement on HTTP/3 support for gRPC. There is an [open proposal](https://github.com/grpc/proposal/blob/master/G2-http3-protocol.md) created by the dotnet gRPC library maintainers and there is [an open issue to discuss actually adding HTTP/3 to the gRPC spec](https://github.com/grpc/grpc/issues/19126). Frustratingly, there hasn't been a lot of movement on the official gRPC repo to add support directly to any of their implementations, but as you can see from the thread, there's a lot of interest and a lot of people making prototypes that prove the concept.

This is likely an incomplete list but here are the packages that you can likely use HTTP/3 with today:
- The standard grpc library for C#, dotnet-grpc [(ref)](https://devblogs.microsoft.com/dotnet/http-3-support-in-dotnet-6/#grpc-with-http-3)
- It may already be possible in rust with Tonic with the Hyper HTTP transport [(ref)](https://github.com/hyperium/tonic/issues/339)
- It's possible in Go if you use [ConnectRPC](https://connectrpc.com/) with [quic-go](https://github.com/quic-go/quic-go) - I don't have a link for this, but I've tested this out myself. This is a topic for a future post!
- This is untested but I believe many gRPC-Web implementations in the browser might "just work" with HTTP/3 as well as long as the browsers are informed of the support via the [ALT-SVC header](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Alt-Svc) and the servers support it.

As more servers and clients support HTTP/3 they should see faster connection establishment times, complete removal of the [head-of-line blocking problem](https://blog.cloudflare.com/the-road-to-quic#headoflineblocking) and much better recovery from packet loss. There's a long way to go here, but there is progress.

## Bridging the Gap
If you're looking to gradually adopt gRPC or need to support existing REST clients, there are several options available *today*!

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

And get a REST-like endpoint where you can make this request:
```shell
curl -XPOST '{"value": "my value!"}' http://localhost:8000/v1/example/echo
```

This is pretty amazing because it's doing a lot of the hard work for you and you can now support many different REST APIs without writing any additional code. This is a simple example here but there are many options, like being able to populate message fields from components of the path.

### gRPC-Web
One of the big limitations of gRPC is that it doesn't work on the web with web browsers due to limited support of HTTP trailers. Browsers support receiving trailers but there isn't yet a way to retrieve those trailers from javascript. Yes, this is incredibly frustrating, especially since there are many small use cases where trailer support would be amazing to have.

The gRPC-Web protocol gives browsers the ability to use gRPC, which drastically improves the story of contract-based services in gRPC. It also allows for HTTP/1.1 clients to work with gRPC. Some platforms (I'm looking at you, [Unity](https://forum.unity.com/threads/support-for-http-2-with-unitywebrequest.1030510/)) still don't support HTTP/2, even though it's 2024 and the `HTTP/2` spec was created nearly a decade ago.

### ConnectRPC
[ConnectRPC](https://connectrpc.com/) automatically generates JSON/HTTP APIs from your gRPC definitions while also maintaining compatibility with gRPC and gRPC-Web. This HTTP protocol, [called Connect](https://connectrpc.com/docs/protocol/), follows HTTP standards more closely. For example, the `Content-Coding` header, `Content-Length` header, HTTP status codes, etc. all work as expected for unary RPC calls. That means you can run this normal-looking curl command and talk to a gRPC service:

```shell
curl --header "Content-Type: application/json" \
    --data '{"sentence": "I feel happy."}' \
    https://demo.connectrpc.com/connectrpc.eliza.v1.ElizaService/Say
```

### Twirp
[Twirp](https://twitchtv.github.io/twirp/) is very similar to ConnectRPC. It was developed by Twitch, and is another framework that can help bridge the gap between gRPC and REST. [Twirp's approach](https://twitchtv.github.io/twirp/docs/spec_v7.html) is to use protobufs to generate an alternative protocol that also aligns more with HTTP conventions. It doesn't also support gRPC and gRPC-Web. Implementing those alongside twirp is left as an exercise for the user if you want to interoperate with other gRPC tooling.

## Tooling
I have mentioned that gRPC tooling isn't that great. I still agree with that if we're talking about the "out of the box" tooling from the gRPC project. However, the community is much bigger than the gRPC Authors and someone finally made the protobuf code generation a lot better.

### Buf CLI
[Buf](https://buf.build/) (the company) has made a client called [Buf CLI](https://buf.build/product/cli), which I'm going to refer to as just "buf" from here on out.

`protoc` is the official compiler for protobufs, which has plugins for many languages, frameworks, documentation and other kinds of outputs. Buf *completely* replaces [`protoc`](https://grpc.io/docs/protoc-installation/) by using the same protoc plugins that `protoc` uses. How is it better? It adds a set of config files for defining the structure of your protobuf files, including external protobuf dependencies and external plugins using the [Buf Schema Registry](https://buf.build/product/bsr). Instead of random makefile directives or bash scripts, we now have a well-defined config file for defining how the protobuf is built, which is amazing.

Similarly, `buf curl` provides a convenient way to interact with gRPC services, much like the popular tool [grpcurl](https://github.com/fullstorydev/grpcurl).

In addition to replacing existing tooling with easier-to-use versions, buf also implements some extremely useful functions. I first started using buf by using `buf lint`, which helps enforce some [common rules and practices](https://buf.build/docs/lint/rules) that developers should follow when making their protobuf files. Soon after, I started using `buf breaking` which will report on breaking changes being made to protobuf files that may break clients. Both have easy-to-use Github actions and were pretty painless to set up.

Adding buf into the mix can greatly improve your workflow with protobufs, especially when working in a larger team or working with other teams.

### Third-party protoc plugins, libraries and tools
There's so many plugins now. I even [made one](https://github.com/sudorandom/protoc-gen-connect-openapi) and to be honest, plugins aren't that hard to make. I think this is the way that API development should work where you base API services off of a contract and generate everything from that same contract. No typos. No confusion over what methods exist. No arguing over REST semantics that has never been clear to anyone.

- **[protoc-gen-doc](https://github.com/pseudomuto/protoc-gen-doc)**: Builds gRPC documentation in several formats. The default styling honestly doesn't look the prettiest but it does allow you to specify a custom template which has been amazing for me to generate something custom without requiring an entire plugin.
- **[protoc-gen-connect-openapi](https://github.com/sudorandom/protoc-gen-connect-openapi)**: This is my plugin. It generates OpenAPIv3 specs for your ConnectRPC services and has support for [protovalidate](https://github.com/sudorandom/protoc-gen-connect-openapi/blob/main/protovalidate.md), [gnostic OpenAPIv3 annotations](https://github.com/sudorandom/protoc-gen-connect-openapi/blob/main/gnostic.md), and [gRPC-Gateway annotations](https://github.com/sudorandom/protoc-gen-connect-openapi/blob/main/grpcgateway.md).
- **[protovalidate](https://github.com/bufbuild/protovalidate)**: Protovalidate allows you to embed validation rules in your protobuf files which can be used by an associated library to enforce those rules. A large complaint that people have with gRPC is that it's hard to use whenever every single field is optional. Now you can replace a lot of validation code, including required fields, with protobuf options. I'm anxiously awaiting [typescript support](https://github.com/bufbuild/protovalidate/issues/67) so validation logic can be shared on web frontends and backends, which, to me, is the "holy grail" feature of a contract-driven service.

In addition to these libraries and plugins, more tools that you know and love from HTTP are supporting gRPC like [Postman](https://blog.postman.com/postman-now-supports-grpc/), [Insomnia](https://docs.insomnia.rest/insomnia/grpc) and [k6](https://k6.io/docs/using-k6/protocols/grpc/).

The availability of numerous third-party plugins underscores the fact that gRPC is more than just a framework – it's a dynamic ecosystem that fosters innovation and empowers developers to customize their workflows to meet their specific requirements.

## Conclusion
gRPC offers a compelling set of advantages for modern web development.

Its performance, strong typing, streaming capabilities, cross-language support, and HTTP/2 foundation make it a powerful tool for building efficient and scalable APIs. With various adoption options available, you can gradually incorporate gRPC into your projects and experience its benefits firsthand.

The growing community and active development around gRPC suggest a bright future for this technology. If you're looking to build fast, reliable, and future-proof APIs, gRPC is a tool that deserves a serious look. Dive in, explore the ecosystem, and discover how gRPC can revolutionize the way you approach API development.
