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
---

I would like to introduce **[FauxRPC](http://fauxrpc.com/)**, a powerful tool that empowers you to accelerate development and testing by effortlessly generating fake implementations of gRPC, gRPC-Web, Connect, and REST services. If you have a [protobuf-based workflow](/posts/api-contracts/), this tool could help.

## Why FauxRPC?
* **Faster Development & Testing:** Work independently without relying on fully functional backend services.
* **Isolation & Control:** Test frontend components in isolation with controlled fake data.
* **Multi-Protocol Support:** Supports multiple protocols (gRPC, gRPC-Web, Connect, and REST).
* **Prototyping & Demos:** Create prototypes and demos quickly without building the full backend. Fake it till you make it.
* **Improved Collaboration:** Bridge the gap between frontend and backend teams.
* **Plays well with others:** Test data from FauxRPC will try to automatically follow any [protovalidate](https://github.com/bufbuild/protovalidate) constraints that are defined.

## How it Works
FauxRPC leverages your Protobuf definitions to generate fake services that mimic the behavior of real ones. You can easily configure the fake data returned, allowing you to simulate various scenarios and edge cases. It takes in `*.proto` files or protobuf descriptors (in binpb, json, txtpb, yaml formats), then it automatically starts up a server that can speak gRPC/gRPC-Web/Connect and REST (as long as there are `google.api.http` annotations defined). Descriptors contain all of the information found in a set of `.proto` files. You can generate them with `protoc` or the `buf build` command.

{{< diagram >}}
{{< image src="diagram.svg" width="800px" class="center" >}}
{{< /diagram >}}

## Get Started
FauxRPC is available as an open-source project. Check out [the documentation](https://fauxrpc.com/docs/intro/) and examples to get started. Here's a quick overview, but be sure to check the official documentation for the most up-to-date instructions:

### Install via source
```
go install github.com/sudorandom/fauxrpc/cmd/fauxrpc@latest
```

### Pre-built binaries
Binaries are built for several platforms for each release. See the latest ones on [the releases page](https://github.com/sudorandom/fauxrpc/releases/latest).

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

## Multi-protocol Support
The multi-protocol support [is based on ConnectRPC](https://connectrpc.com/docs/multi-protocol/). So with FauxRPC, you get **gRPC, gRPC-Web and Connect** out of the box. However, FauxRPC does one thing more. It allows you to use [`google.api.http` annotations](https://grpc-ecosystem.github.io/grpc-gateway/docs/tutorials/adding_annotations/) to present a JSON/HTTP API, so you can gRPC and REST together! This is normally done with [an additional service](https://github.com/grpc-ecosystem/grpc-gateway) that runs in-between the outside world and your actual gRPC service but with FauxRPC you get the so-called transcoding from HTTP/JSON to gRPC all in the same package. Here's a concrete example:

```protobuf
syntax = "proto3";

package http.service;

import "google/api/annotations.proto";

service HTTPService {
  rpc GetMessage(GetMessageRequest) returns (Message) {
    option (google.api.http) = {get: "/v1/{name=messages/*}"};
  }
}
message GetMessageRequest {
  string name = 1; // Mapped to URL path.
}
message Message {
  string text = 1; // The resource content.
}
```

Again, we start the service by building the descriptors and using
```
$ buf build ./httpservice.proto -o ./httpservice.binpb
$ fauxrpc run --schema=httpservice.binpb
```

Now that we have the server running we can test this with the "normal" curl:
```shell
$ curl http://127.0.0.1:6660/v1/messages/123456
{"text":"Retro."}⏎
```
Sweet. You can now easily support REST alongside gRPC. If you are wondering how to do this with "real" services, look into [vanguard-go](https://github.com/connectrpc/vanguard-go). This library is doing the real heavy lifting.

## What does the fake data look like?
You might be wondering what actual responses look like. FauxRPC's fake data generation is continually improving so these details might change as time goes on. It uses a library called [fakeit](https://github.com/brianvoe/gofakeit) to generate fake data. Because protobufs have pretty well-defined types, we can easily generate data that technically matches the types. This works well for most use cases, but FauxRPC tries to be a little bit better. If you annotate your protobuf files with [protovalidate](https://github.com/bufbuild/protovalidate) constraints, FauxRPC will try its best to generate data that matches these constraints. Let's look at some examples!

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

With FauxRPC, you will get any kind of word, so it might look like this:
```json
{
  "greeting": "sufficient"
}
```
This is fine, but for the RPC, we know a bit more about the type being returned. We know that it sends a greeting back that looks like "Hello, [name]". So here's what the same protobuf file might look like with protovalidate constraints:


Now let's see what this looks like with protovalidate constraints:
```protobuf
syntax = "proto3";

import "buf/validate/validate.proto";

package greet.v1;

message GreetRequest {
  string name = 1 [(buf.validate.field).string = {min_len: 3, max_len: 100}];
}

message GreetResponse {
  string greeting = 1 [(buf.validate.field).string.pattern = "^Hello, [a-zA-Z]+$"];
}

service GreetService {
  rpc Greet(GreetRequest) returns (GreetResponse) {}
}
```

With this new protobuf file, this is what FauxRPC might output now:

```json
{
  "greeting": "Hello, TWXxF"
}
```
This shows how protovalidate constraints enable FauxRPC to generate more realistic and contextually relevant fake data, aligning it closer to the expected behavior of your actual services. As another example, I will show one of Buf's services used to manage users, [buf.registry.owner.v1.UserService](https://buf.build/bufbuild/registry/docs/main:buf.registry.owner.v1#buf.registry.owner.v1.UserService). Here's what the `UserRef` message looks like:

```protobuf
message UserRef {
  option (buf.registry.priv.extension.v1beta1.message).request_only = true;
  oneof value {
    option (buf.validate.oneof).required = true;
    // The id of the User.
    string id = 1 [(buf.validate.field).string.tuuid = true];
    // The name of the User.
    string name = 2 [(buf.validate.field).string = {
      min_len: 2
      max_len: 32
      pattern: "^[a-z][a-z0-9-]*[a-z0-9]$"
    }];
  }
}
```

So let's make our descriptors for this service, start the FauxRPC server and make our example request:

```shell
$ buf build buf.build/bufbuild/registry -o bufbuild.registry.binpb
$ fauxrpc run --schema=./bufbuild.registry.binpb
$ buf curl --http2-prior-knowledge http://127.0.0.1:6660/buf.registry.owner.v1.UserService/ListUsers
{
  "nextPageToken": "Food truck.",
  "users": [
    {
      "id": "c4468393f926400d8880a264df9c284a",
      "createTime": "2012-03-06T12:15:03.239463070Z",
      "updateTime": "1990-10-29T13:12:31.224347086Z",
      "name": "jexox",
      "type": "USER_TYPE_STANDARD",
      "description": "Tattooed taxidermy.",
      "url": "http://www.productexploit.name/synergies/target"
    },
    {
      "id": "0e4ca24f4ff54761b109daab0da1bea2",
      "createTime": "1955-05-16T02:37:30.643378679Z",
      "updateTime": "1923-08-28T04:28:43.330711919Z",
      "name": "ya0",
      "type": "USER_TYPE_STANDARD",
      "state": "USER_STATE_INACTIVE",
      "description": "Helvetica.",
      "url": "https://www.centralengage.info/markets/scale/e-commerce/exploit",
      "verificationStatus": "USER_VERIFICATION_STATUS_UNVERIFIED"
    }
  ]
}
```
Hopefully, this gives you a good idea of what the output might look like. The better your validation rules, the better the FauxRPC data will be.

## What's left?
FauxRPC is already great for some use cases but it's not "done" as there's more to do to make it better. I have plans to add the ability to configure stubs for each RPC method. This will allow you to define specific responses or behaviors for each RPC, giving you more control over the simulated service. I hope this will make it easier to iterate on protobuf designs without needing to actually implement services until later.

## Stay Tuned
I made a [documentation website](http://fauxrpc.com/) to organize documentation. I think it looks pretty good for how quickly I threw it together. The code for FauxRPC lives on GitHub at [sudorandom/fauxrpc](github.com/sudorandom/fauxrpc). It's a little thin now but there's a lot that I can write about in there. I'm actively developing FauxRPC and have many exciting features planned for the future. This is early on for this project but it has come together as a coherent and useful program for me extremely quickly. So please try it out and let me know your feedback and suggestions. Stay tuned for updates!

*... and don't forget to [star the repo on GitHub](https://github.com/sudorandom/fauxrpc). It helps more than you know!*