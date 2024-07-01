---
categories: ["article"]
tags: ["grpc", "protobuf", "api", "rpc", "webdev", "humor", "http2", "http3"]
series: ["gRPC: the good and the bad"]
date: "2024-06-18"
description: "gRPC isn't perfect but who is?"
cover: "cover.jpg"
images: ["/posts/grpc-the-bad-parts/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC: The Bad Parts"
slug: "grpc-the-bad-parts"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-the-bad-parts
featured: true
---

gRPC, the high-performance RPC framework, has been super successful (if you work for Google) and has drastically changed the way we all deploy APIs (if you work for Google). gRPC and protobuf is an extremely performant contract-focused framework with extremely wide language support. But it's not without its downsides. Making a RPC framework that requires code generation and support in many programming languages is sure to get some things wrong. As gRPC approaches a decade of usage, it is important to reflect on what could have been better.

## Learning Curve

Let's start out extremely nit picky. So-called unary RPCs are calls where the client sends a single request to the server and gets a single response back. Why does gRPC have to use such a non-standard term for this that only mathematicians have an intuitive understanding of? I have to explain the term every time I use it. And I'm a little tired of it.

Speaking of unary RPCs, the implementation is more complicated than it needs to be. While gRPC's streaming capabilities are powerful, they have introduced complexity for simple RPC calls that don't require streaming. This hurts the ability to inspect gRPC calls because now there is framing on every unary RPC which only makes sense for streaming. Protobuf encoding is complicated enough so let's not add extra gRPC framing where it isn't needed. Also, it doesn't pass my "send a friend a cURL example" test for any web API. It's just super annoying to explain to someone how to use gRPC. I've said "okay, but is server reflection enabled?" so many times. I'm just tired of it.

This complexity also bleeds into the tooling with the mandatory code generation step. This can be a hurdle, especially for dynamic languages where runtime flexibility is valued. Additionally, some developers might be hesitant to adopt a technology that necessitates an extra build step. We already need 20 build steps for modern web development, it's sometimes hard to justify one more.

## Compatability with the Web

The reliance on HTTP/2 initially limited gRPC's reach, as not all platforms and browsers fully supported it. This has improved over time, but it still poses a challenge in some environments. But even with HTTP/2 support, browsers have avoided adding a way to process HTTP trailers so browsers today still cannot use "original" gRPC. gRPC-Web has acted as a plaster for this issue by avoiding the use of trailers, but it often requires "extra stuff" like running a proxy that supports gRPC-Web. Which is annoying.

Late Adoption of HTTP/3: The delay in embracing HTTP/3 might have hindered gRPC's ability to take full advantage of the protocol's performance and efficiency benefits. I have personally been affected by the [head-of-line blocking](https://http3-explained.haxx.se/en/why-quic/why-tcphol) issue that can happen when using gRPC with HTTP/2 and it would be so nice to be able to completely do away with this issue by being able to use HTTP/3 with gRPC. It's strange to see a framework that pushed many languages to support HTTP/2 struggling to do the same thing with HTTP/3.

## JSON Mapping and Prototext

Another area where the "timing" was wrong was the lack of a standardized JSON mapping early on. It has made gRPC less accessible for developers accustomed to JSON-based APIs and I don't think it ever recovered from that stigma. Having a mapping between protobuf types and JSON simplifies integration and interoperability with existing tools and systems. You would not believe how happy web developers can get when you say "yeah,
this is a super-efficient binary format... but you can set this flag and get JSON back if you want to debug." They get unreasonably excited. *Unreasonably. excited.* Anyway, now that protobuf has standard rules for mapping protobuf types to JSON (and the other way) I feel like the [protobuf text format](https://protobuf.dev/reference/protobuf/textformat-spec/) is an unnecessary complexity. I don't see a use-case for the text format now that we have JSON. So let's throw the text format away. We don't need it and I'm down to pretend like it never existed if everyone else is. Cool?

## Finite Message Sizes

Most Protobuf encoders/decoders expect to fully parse an entire message and give the full response to the consumer but memory is finite and sometimes you might want larger messages. Sometimes you want to stream parts of these larger messages somewhere else and not keep the entire message in memory. Therefore, if you want to, for example, upload large files you're going to need to implement some kind of chunking. While chunking is a reasonable solution for handling large files, the absence of a standardized approach within gRPC might lead to inconsistent implementations and increased development effort.

As a demonstration, here's what it may look like to upload a file with gRPC:

```protobuf
syntax = "proto3";

package file_service;

service FileService {
   rpc Upload(stream UploadRequest) returns(UploadResponse);
}

message UploadRequest {
    string file_name = 1;
    bytes chunk = 2;
}

message UploadResponse {
  string etag = 1;
}
```

This is both a strength and weakness of protobufs. This concept is super easy to define in protobuf but in practice, the code to properly implement this can be cumbersome and error-prone. And while Google, the creator of gRPC, has figured out solutions for their APIs, the lack of a standardized approach leaves others to reinvent the wheel.

You might be thinking "Google uses gRPC in most of their APIs so obviously they've done this" and you'd be right. They actually have a gRPC and HTTP version for downloading (potentially large) files. We can compare the [gRPC](https://github.com/googleapis/google-cloud-go/blob/v0.114.0/storage/grpc_client.go#L996-L1152) and [HTTP](https://github.com/googleapis/google-cloud-go/blob/v0.114.0/storage/http_client.go#L888-L911) version directly and gRPC is BY FAR more complex. Go ahead compare the linked code. I'll wait.


## ded internet theory

I see a lot of gRPC/protobuf communities that are devoid of activity. The lack of visible activity on some websites might create the impression that gRPC is stagnant or less actively maintained. This could deter potential adopters and contribute to slower community growth. This might be a case of too many options, making it difficult to find someone to nerd out about gRPC outside of GitHub issues where such enthusiasm might be perceived as annoying.

## Bad tooling

For the longest time, when I saw that a codebase uses protobuf I find a weird script which downloads random protobuf files in super custom ways and places them in random paths and then makes a series of super complex calls to `protoc`. Only google would think not solving dependency management is the solution to dependency management. Google has their own extremely Google-y way of managing dependencies that us peasants can only dream of using.

## It can be (and is) better

While I've been critical of gRPC, I hope my comments come across as constructive. Those who have read this far down in the article get to know that many of these issues are already fixed or at least on the way to being fixed!

- Several gRPC implementations already support HTTP/3. ConnectRPC makes it pretty easy to use HTTP/3 with gRPC (I'll followup on this in a future post).
- Since the [protobuf spec has a canonical mapping to/from JSON](https://protobuf.dev/programming-guides/proto3/#json) I no longer have to worry about the text format. I really do hope that everyone forgets that it exists. There's only room for so many text-based formats. I wasn't joking about that. This is the last time I'm acknowledging its existence.
- The gRPC community is actually alive and well if you know where to look. For example, the [buf slack](https://buf.build/links/slack) has been a great resource for me. You may find me hanging out and answering questions fairly often.
- The [Buf CLI](https://buf.build/docs/ecosystem/cli-overview) is an amazing tool for gRPC. It completely replaces `protoc` but also adds linting, breaking change detection, curl for gRPC, integration with the Buf Schema Registry (wow, real dependency management!) and more! In addition, more tools that you know and love from HTTP are supporting gRPC like [Postman](https://blog.postman.com/postman-now-supports-grpc/), [Insomnia](https://docs.insomnia.rest/insomnia/grpc) and [k6](https://k6.io/docs/using-k6/protocols/grpc/).

Despite gRPC's undeniable successes, it's important to acknowledge the framework's shortcomings to ensure its continued evolution and improvement. By addressing its learning curve, compatibility issues, lack of standardization, and community engagement, we can unlock gRPC's full potential and make it a more accessible and user-friendly tool for all developers.
