---
categories: ["article"]
tags: ["protobuf", "grpc", "testing", "json"]
date: "2024-09-03"
description: "A Quick Start with Caveats"
cover: "cover.jpg"
images: ["/posts/json-to-proto/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "JSON to Protobuf Conversion"
slug: "json-to-proto"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/json-to-proto/
draft: true
---

In the world of data serialization and communication protocols, Protocol Buffers (Protobuf) have gained immense popularity due to their efficiency and performance advantages, further fueled by the spread of gRPC and contract-based APIs. However, the learning curve associated with Protobuf's syntax and concepts can be a deterrent for developers, especially those already comfortable with the ubiquitous JSON format.

Enter the [JSON to Protobuf converter](https://json-to-proto.github.io/), a tool that aims to bridge this gap by automatically translating JSON data structures into their Protobuf equivalents. While this can offer a seemingly straightforward entry point to Protobuf adoption, it's important to understand both the potential benefits and the limitations that come with such an approach.

The tool is located here: [json-to-proto.github.io](https://json-to-proto.github.io/).

## Accelerating Protobuf Adoption

- **Rapid Prototyping**: The most apparent advantage of a JSON to Protobuf converter is its ability to help developers quickly experiment with Protobuf without having to delve deep into the intricacies of its schema language and data types.
- **Legacy System Integration**: In scenarios where existing systems rely on JSON, a converter can facilitate a gradual transition to Protobuf by automating the initial conversion process.

### Getting it right
One place I like to go to for a typical REST API is the swagger petstore example, [available at petstore.swagger.io](https://petstore.swagger.io). This API acts an example for OpenAPI, so it has a mix of features that you'll see when using a typical REST API. So let's what what JSON-to-proto will do with this API.

Starting with this example:
```json
{
  "id": 0,
  "category": {
    "id": 0,
    "name": "string"
  },
  "name": "doggie",
  "photoUrls": [
    "string"
  ],
  "tags": [
    {
      "id": 0,
      "name": "string"
    }
  ],
  "status": "available"
}
```

I get the following protobuf back:
```protobuf
syntax = "proto3";

message SomeMessage {

    message Category {
        uint32 id = 1;
        string name = 2;
    }

    message Tags {
        uint32 id = 1;
        string name = 2;
    }

    uint32 id = 1;
    Category category = 2;
    string name = 3;
    repeated string photo_urls = 4;
    repeated Tags tags = 5;
    string status = 6;
}
```

This is pretty decent. If I'm going to reuse `Category` and `Tags` in other types, I would move it from outside of `SomeMessage`, but generally this did extremely well. It made some assumptions that `uint32` is what all IDs look like, which may be fair but also may be wrong. Let's did into the ways this tool could lead us astray.

## The Price of Simplicity

- **Loss of Expressiveness**: JSON, while flexible, lacks the type richness and structural constraints of Protobuf. This means that converting JSON to Protobuf can result in a loss of information and reduced clarity.
    - **Enums**: Protobuf's enums provide a clear way to define a restricted set of values. JSON, lacking this concept, forces converters to rely on string representations, which are less type-safe and can lead to runtime errors.
    - **Numeric Types**: JSON's "number" type is ambiguous, encompassing a wide range of numeric representations. Protobuf, on the other hand, offers a selection of specific integer and floating-point types, enabling more efficient storage and processing. Converters may struggle to accurately infer the intended Protobuf type from a JSON number.
- **Imperfect Conversions**: The complexity of data structures and the ambiguity inherent in JSON can lead to conversion errors or less-than-ideal Protobuf schemas.
    - **Ambiguous input, ambiguous output**: If your examples leave a field as "null" then you may end up with `google.protobuf.Any` types, which is usually not what you want. The example data should always
    - **One field, two types**: When a JSON field can hold values of different types, converters might default to using `google.protobuf.Any`, a generic container type that can encapsulate any Protobuf message. While convenient, this approach sacrifices type safety and can make working with the resulting Protobuf data more cumbersome.

### Getting it wrong
Let's look at some concrete examples.

#### Incomplete examples
```json
{
    "id": 1234,
    "name": null
}
```
yields
```protobuf
syntax = "proto3";

import "google/protobuf/any.proto";

message SomeMessage {
    uint32 id = 1;
    google.protobuf.Any name = 2;
}
```

#### Multi-typed Fields
Here's an example where the ID for users could be a string or an integer:
```json
{
    "users": [
        {"type":"admin", "id": "admin-1"},
        {"type":"user", "id": 1234}
    ]
}
```
and here is the resulting protobuf:
```protobuf
syntax = "proto3";

message SomeMessage {

    message Users {
        string type = 1;
        string id = 2;
    }

    repeated Users users = 1;
}
```
The tool assumes that user.id is a string. This may be "fine" but also maybe it isn't. That all depends.

This behavior is even worse when you consider a list that contains multiple types:
```json
{
    "items": [1, "2", 3.3, null]
}
```

results in:
```protobuf
syntax = "proto3";

message SomeMessage {
    repeated uint32 items = 1;
}
```

Oh no! It only picked uint32 for the type, so this tool seems to only pick the first type it sees as a candidate type for a field. It could definitely be smarter about this. In this particular case, I might have used the [google.protobuf.Value](https://protobuf.dev/reference/protobuf/google.protobuf/#value) well-known type to indicate that the array can have nulls, numbers, strings, booleans, or a list/struct that contains the same. This type is very useful for working with dynamically typed fields. These kinds of fields with multiple types are usually frowned upon, so you may want to use this conversion process to move to a "one field: one type" design.

## A Useful Tool, But Not a Replacement for Understanding Protobuf
JSON to Protobuf converters can undoubtedly be helpful in specific scenarios, especially for quick experimentation and legacy system integration. However, they should not be seen as a substitute for gaining a solid understanding of Protobuf's concepts and syntax, especially as gRPC and contract-based APIs continue to gain traction.

By investing time in learning Protobuf directly, developers can leverage its full potential, designing schemas that are not only more expressive but also optimized for performance and maintainability.

Remember, a converter like this is a tool, not a magic wand. Use it wisely, but never let it replace the power of knowledge. 
