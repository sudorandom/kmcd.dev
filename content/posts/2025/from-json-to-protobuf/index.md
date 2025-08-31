---
categories: ["article"]
tags: ["protobuf", "grpc", "testing", "json"]
date: "2025-09-02T10:00:00Z"
description: "A Tool to Ease Your Schema Journey"
cover: "cover.png"
images: ["/posts/from-json-to-protobuf/cover.png"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "From JSON to Protobuf"
slug: "from-json-to-protobuf"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/from-json-to-protobuf/
---

When I first approached the possibility of using protobuf or gRPC, I was intimidated by a few things. First, it was the terrible tooling: Protoc was weird, the plugin system it used was odd, the source of binaries was from wildly different places depending on the platform, and the code it generated was insane. However, [the buf CLI](https://buf.build/product/cli) fixes most of these issues for me. The next challenge I distinctly remember was getting started with my own Protobuf definitions. The tutorials were perfectly fine, but once you are set off on your own it's kind of hard to know what to do next. I ended up reading the spec, since it's not all that long. But I do feel like others don't learn well from that method. Others will learn from examples. And the best examples are those that build on a foundation of knowledge they already have.

This is the problem I built a tool to solve. JSON has become an inescapable reality for developers of all types. It is very hard to become a junior developer without seeing and using a decent amount of JSON. Therefore, I figured that JSON is a good starting point for explaining protobufs. They are similar in many ways; in fact, there is now a [standard way to convert protobuf types into JSON](https://protobuf.dev/programming-guides/json/). That said, you may know that protobuf has some differences. First, it has an encoding that is more efficient than JSON, in terms of space and often in terms of processing time. Second, it is a schema-driven format. Protobuf files can be used to generate types for code in most programming languages and can be used to automatically generate documentation and other kinds of artifacts. This is a powerful concept that enables the sharing of types to many different languages. This approach reduces errors, reduces the amount of duplicate code that you have to write and maintain for common types and it enables some more efficient storage and processing of data.

The tool I built is meant to be a bridge for those stuck in the world of schema-less JSON, guiding them into a world of type safety and all the other benefits that Protobufs provide. I introduce, [json-to-proto.kmcd.dev](https://json-to-proto.kmcd.dev/). This website/tool will take example JSON documents and produces a protobuf file that can consume and emit the same JSON. The goal here is mostly for education; to translate something junior developers know (how to read JSON) to something that they might not know (how to make the equivalent protobuf file). I'm hoping that it will provide enough material to get developers started.

{{< figure src="screenshot.png" height="800px" caption="[json-to-proto.kmcd.dev](https://json-to-proto.kmcd.dev/)" >}}

{{< bigtext >}}Give it a try at [json-to-proto.kmcd.dev](https://json-to-proto.kmcd.dev/)!{{< /bigtext >}}

{{< warning-box >}}
Use this as an educational tool. _Understand your protobufs before you ship_.
{{< /warning-box >}}

---

## How it works
The magic of this tool lies in its straightforward approach to translating the flexible world of JSON into the structured world of Protobuf. It does this by applying a series of simple rules and heuristics to the input data. The core of this process is type mapping, where the tool identifies each JSON data type (a string, number, boolean, or object) and converts it to a suitable Protobuf counterpart.

### JSON String → Protobuf string
All JSON strings are converted to the Protobuf `string` type.

```json
{
  "username": "joe"
}
```

```protobuf
syntax = "proto3";

package my_package;

message UserProfile {
  string username = 1;
}
```

This is a straightforward mapping. The tool recognizes the string value `"joe"` and translates the field to the corresponding `string` type in Protobuf, a simple and direct conversion that works for text of all kinds.

### JSON Number → Protobuf int64 or double

The converter maps integer numbers to `int64` and floating-point numbers to `double`. The use of `int64` and `double` is a safe choice to ensure both large numbers and decimal values are handled correctly.

```json
{
  "userId": 12345,
  "gpa": 3.4
}
```

```protobuf
syntax = "proto3";

package my_package;

message UserProfile {
  int64 userId = 1;
  double gpa = 2;
}

```

### JSON Boolean → Protobuf bool
JSON boolean values (`true` and `false`) are converted to the Protobuf `bool` type.

```json
{
  "graduated": false,
  "enrolled": true
}
```

```protobuf
syntax = "proto3";

package my_package;

message UserProfile {
  bool graduated = 1;
  bool enrolled = 2;
}
```

Like strings, booleans have a direct Protobuf equivalent. The tool sees the `true` and `false` values and maps them directly to a `bool` type, making this a simple and lossless conversion.

### JSON Object → Protobuf message
Any JSON object is mapped to a new, nested Protobuf `message` type. The name of the new message is derived from the object's field name. I've been using objects for all of my examples since they are essentially required at the top level. However, there's something interesting to note about the attribute names. If the JSON attributes don't match in terms of capitalization and underscore usage that protojson uses, json-to-proto.kmcd.dev will add a `json_name` annotation to make it match up correctly.

```json
{
  "user_id": 12345,
  "UserName": "joe",
  "ENROLLED": true
}
```

```protobuf
syntax = "proto3";

package my_package;

message UserProfile {
  int64 user_id = 1 [json_name = "user_id"];
  string user_name = 2 [json_name = "UserName"];
  bool enrolled = 3 [json_name = "ENROLLED"];
}

```

This is where the tool gets clever with its handling of naming conventions. Notice how `UserName` is converted to `user_name` in the Protobuf file but keeps its original name in the JSON with a `json_name` option. This ensures the generated Protobuf still works seamlessly with your existing JSON data, a powerful feature for maintaining compatibility.

### JSON Array → Protobuf repeated

A JSON array is converted into a `repeated` field in Protobuf. The type of the elements within the `repeated` field is inferred from the data contained in the array.

```json
{
  "courses": [
    {
      "courseId": "CS101",
      "courseName": "Intro to CS"
    },
    {
      "courseId": "MA203",
      "courseName": "Linear Algebra",
      "credits": null
    }
  ],
  "login_timestamps": [
    1679400000,
    1679486400
  ],
  "mixed_data": [
    1,
    "test",
    true,
    {
      "key": "value"
    },
    null
  ]
}
```

```protobuf
syntax = "proto3";

import "google/protobuf/struct.proto";

package my_package;

message Course {
  repeated string course_id = 1;
  repeated string course_name = 2;
  repeated google.protobuf.Value credits = 3;
}

message UserProfile {
  repeated Course courses = 1;
  repeated int64 login_timestamps = 2 [json_name = "login_timestamps"];
  repeated google.protobuf.Value mixed_data = 3 [json_name = "mixed_data"];
}
```

This example shows three of these cases:
- `courses` has a repeated field of `Course` messages.
- `login_timestamps` as a repeated field with `int64` values.
- `mixed_data` as a repeated field of `google.protobuf.Value` messages, because the JSON array has several different types.

Another edge-case is hit here as well. The `Course` message has a `credits` field of type `google.protobuf.Value`. Why? Because the input JSON only has that field as being `null`. So the tool doesn't have enough information to know what type `"credits"` should be.

Here, the tool's logic for handling arrays shines. A `repeated` field is Protobuf's way of representing a list. The tool infers the types within each array: a custom `Course` message, a list of `int64` values, and a catch-all `google.protobuf.Value` for the `mixed_data` array, which contains different data types. To get the most accurate protobuf schema, it's best to provide a representative JSON example that includes all possible data types and structures.

---

## When you shouldn't use this tool

It's crucial to acknowledge that a tool like this is not a perfect replacement for human judgment. It is a powerful starting point, but a developer must always review and refine the output to ensure it perfectly matches the application's needs.

### Type Inference

While the tool makes an intelligent guess about types, it can't read your mind. For example, a JSON number might be inferred as an `int64`, but your application may only ever need an `int32`. A developer should always make the final decision based on their knowledge of the data.

### map vs. message
The tool defaults to a `message` for JSON objects. A developer might need to manually change this to a `map` if the object's keys are truly arbitrary (e.g., a dictionary of feature flags).

---

## A Starting Point, Not a Destination

In a world where JSON's flexibility often leads to schema-less chaos, Protobuf offers a path to efficiency, speed, and type safety. My tool is designed to serve as a powerful bridge, providing a smooth on-ramp for developers who are intimidated by the initial hurdle of defining a `.proto` schema from scratch.

I encourage you to give it a try. The tool is a powerful assistant that eliminates the tedious first steps, freeing you up to focus on what matters most: **building robust, scalable applications with a solid schema as their foundation.**
