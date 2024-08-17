---
categories: ["article"]
tags: ["grpc", "protobuf", "api", "rpc", "webdev", "http2", "http3", "connectrpc", "web", "testing"]
date: "2024-08-20T10:00:00Z"
description: "I made a server that outputs nonsense."
cover: "cover.jpg"
images: ["/posts/fauxrpc/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "FauxRPC"
slug: "fauxrpc"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/fauxrpc/
draft: true
---

I'm excited to announce [**FauxRPC**](https://github.com/sudorandom/fauxrpc), a powerful tool designed to accelerate your development and testing workflows by providing fake implementations of gRPC, gRPC-Web, Connect, and REST services (as long as it's based on your Protobuf definitions).

## Why FauxRPC?
* **Faster Development & Testing:** Work independently without relying on fully functional backend services.
* **Isolation & Control:** Test frontend components in isolation with controlled fake data.
* **Multi-Protocol Support:** Supports multiple protocols (gRPC, gRPC-Web, Connect, and REST).
* **Prototyping & Demos:** Create prototypes and demos quickly without building the full backend. Fake it till you make it.
* **Improved Collaboration:** Bridge the gap between frontend and backend teams.
* **Ecosystem Integration:** Test data from FauxRPC will try to automatically follow any [protovalidate](https://github.com/bufbuild/protovalidate) constraints that are defined.

## How it Works
FauxRPC leverages your Protobuf definitions to generate fake services that mimic the behavior of real ones. You can easily configure the fake data returned, allowing you to simulate various scenarios and edge cases. It takes in protobuf files or descriptors (in binpb, json, txtpb, yaml formats), then it automatically starts up a server that can speak gRPC/gRPC-Web/Connect and REST (as long as there are `google.api.http` annotations defined).

{{< diagram >}}
{{< image src="diagram.svg" width="800px" class="center" >}}
{{< /diagram >}}

## Get Started
FauxRPC is available as an open-source project. Check out [the documentation](https://github.com/sudorandom/fauxrpc) and examples to get started. Here's an abbreviated version here:

### Use Descriptors
Make an `example.proto` file (or use a file that already exists):
```protobuf
syntax = "proto3";

package greet.v1;

message GreetRequest {
  string name = 1;
}

message GreetResponse {
  string greeting = 1;
}

service GreetService {
  rpc Greet(GreetRequest) returns (GreetResponse) {}
}
```

Create a descriptors file and use it to start the FauxRPC server:
```shell
$ buf build ./example.proto -o ./example.binpb
$ fauxrpc run --schema=./example.binpb
2024/08/17 08:01:19 INFO Listening on http://127.0.0.1:6660
2024/08/17 08:01:19 INFO See available methods: buf curl --http2-prior-knowledge http://127.0.0.1:6660 --list-methods
```
Done! It's that easy. Now you can call the service with any tooling that supports gRPC, gRPC-Web, or connect. So [buf curl](https://buf.build/docs/reference/cli/buf/curl), [grpcurl](https://github.com/fullstorydev/grpcurl), [Postman](https://www.postman.com/), [Insomnia](https://insomnia.rest/) all work fine!

```shell
$ buf curl --http2-prior-knowledge http://127.0.0.1:6660/greet.v1.GreetService/Greet
{
  "greeting": "dream"
}
```

### Server Reflection
If there's an existing gRPC service running that you want to emulate, you can use server reflection to start the FauxRPC service:
```shell
$ fauxrpc run --schema=https://demo.connectrpc.com
```

### From BSR (Buf Schema Registry)
Buf has a [schema registry](https://buf.build/product/bsr) where many schemas are hosted. Here's how to use FauxRPC using images from the registry.

```shell
$ buf build buf.build/bufbuild/registry -o bufbuild.registry.json
$ fauxrpc run --schema=./bufbuild.registry.json
```

### Multiple Sources
You can define this `--schema` option as many times as you want. That means you can add services from multiple descriptors and even mix and match from descriptors and from server reflection:
```shell
$ fauxrpc run --schema=https://demo.connectrpc.com --schema=./example.binpb
```

## Stay Tuned
I'm actively developing FauxRPC and have many exciting features planned for the future. This is early on for this project but it has come together as a coherent and useful program for me extremely quickly. So please try it out and let me know your feedback and suggestions. Stay tuned for updates!
