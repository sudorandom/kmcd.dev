---
categories: ["article"]
tags: ["fauxrpc", "grpc", "protobuf", "protovalidate", "api", "connectrpc", "testing"]
date: "2024-11-12T10:00:00Z"
description: ""
cover: "cover.jpg"
images: ["/posts/fauxrpc-protovalidate/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "FauxRPC and Protovalidate"
slug: "fauxrpc-protovalidate"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/fauxrpc-protovalidate/
draft: true
---

[FauxRPC](https://fauxrpc.com/), a tool for generating fake gRPC servers, now integrates with [protovalidate](https://github.com/bufbuild/protovalidate), a library for defining validation rules in your Protobuf definitions. This means faster debugging, increased reliability, and a smoother development experience.

Now every request processed by FauxRPC will be automatically validated against your protovalidate rules. Not only will you get high quality data validation in your application, but now you can have this validation before you even write your application logic! This leads to:

* **Faster debugging:** Identify and resolve data issues early on, saving you precious development time.
* **Increased reliability:** Ensure your RPCs only work with valid data, preventing unexpected behavior and crashes.
* **Improved developer experience:** Get clear and concise error messages that pinpoint the source of any validation problems.
* **Greater consistency:** Enforce data integrity across your entire application.

## How it works

Simply define your validation rules using protovalidate's constraint annotations within your Protobuf definitions. Fauxrpc will take care of the rest, automatically validating each request against these rules. If a request fails validation, a detailed error message will be returned, guiding you towards a quick resolution.

### Example

**greet.proto**
```protobuf
syntax = "proto3";

package greet.v1;

import "buf/validate/validate.proto";

message GreetRequest {
  string name = 1 [(buf.validate.field).string = {min_len: 3, max_len: 20}];
}

message GreetResponse {
  string greeting = 1 [(buf.validate.field).string.example = "Hello, user!"];
}

service GreetService {
  rpc Greet(GreetRequest) returns (GreetResponse) {}
}
```

To use protovalidate with FauxRPC, we need to leverage another tool that's extremely helpful for working with protobuf definitions. This is because we used protovalidate, a dependency. Usually dependencies are hard to deal with in protobuf because the default protobuf tooling just doesn't manage dependencies at all. However, the [Buf CLI](https://buf.build/product/cli) and the [BSR](https://buf.build/product/bsr) can help us here.

First, create a new `buf.yaml` file in the same directory as `greet.proto` with the contents:

```yaml
version: v2
deps:
  - buf.build/bufbuild/protovalidate
```

Now we're just a few commands away from having a mock service running:

```shell
# Get Buf CLI to pull our new dependency
$ buf dep update
# Build our protobuf file (and dependencies) into a protobuf "image": https://buf.build/docs/build/overview/
$ buf build . -o greet.binpb
# Start FauxRPC with this image
$ fauxrpc run --schema=greet.binpb
```

Now let's try some requests against this new service:
```shell
$ buf curl --http2-prior-knowledge -d '{}' http://127.0.0.1:6660/greet.v1.GreetService/Greet
{
   "code": "invalid_argument",
   "message": "validation error:\n - name: value length must be at least 3 characters [string.min_len]",
   "details": [
      {
         "type": "buf.validate.Violations",
         "value": "CkIKBG5hbWUSDnN0cmluZy5taW5fbGVuGip2YWx1ZSBsZW5ndGggbXVzdCBiZSBhdCBsZWFzdCAzIGNoYXJhY3RlcnM",
         "debug": {
            "violations": [
               {
                  "fieldPath": "name",
                  "constraintId": "string.min_len",
                  "message": "value length must be at least 3 characters"
               }
            ]
         }
      }
   ]
}
```

Oh, duh! We hit the constraint, so our empty object `{}` isn't good enough anymore. We need a name and it needs to have between 3 and 20 characters. Let's try once more:

```shell
$ buf curl --http2-prior-knowledge -d '{"name": "Bob"}' http://127.0.0.1:6660/greet.v1.GreetService/Greet
{
  "greeting": "Hello, user!"
}
```
Sweet we got a response! And the response is populated using our example annotation. This will allow us to stand up a fake service with a few simple commands that has request validation built in and will use the constraints to make realistic fake data.

I hope this example shows the power of mixing protovalidate with FauxRPC.

As an aside, I am personally excited and anxious to eventually have [Typescript Support for protovalidate](https://github.com/bufbuild/protovalidate/issues/67). That would allow us to use protovalidate for both clean frontend UX and robust server side validation.

## Benefits for you

FauxRPC and protobuf synergizes well with the model-driven API design with protobuf. From a single API definition you have strongly typed definitions that has support for many programming languages, powerful validation constraints, examples of what each field looks like. And FauxRPC lets you experiment with all of that before writing a single line of application code. This means:

* **Reduced development time:** Spend less time debugging data issues and more time building amazing features.
* **Increased confidence:** Trust that your RPCs are handling data correctly, leading to more stable and reliable applications.
* **Improved collaboration:** Make it easier for teams to work together by ensuring everyone adheres to the same data standards.

## Ready to give it a try?

Ready to experience the power of FauxRPC and protovalidate?

- Update to the latest version of FauxRPC: [github.com/sudorandom/fauxrpc/releases/latest](https://github.com/sudorandom/fauxrpc/releases/latest)
- Learn more about protovalidate: [github.com/bufbuild/protovalidate](https://github.com/bufbuild/protovalidate)
- Explore the FauxRPC documentation: [fauxrpc.com](https://fauxrpc.com/)

For reference, all of the code in the article [is available here](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2024/fauxrpc-protovalidate/proto).