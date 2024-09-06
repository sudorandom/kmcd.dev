---
categories: ["article"]
tags: ["protobuf", "grpc", "testing", "json"]
date: "2024-09-10T10:00:00Z"
description: "Deep dive into a small Protobuf tool"
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

In the world of data serialization and communication protocols, Protocol Buffers (protobuf) have gained immense popularity due to their efficiency and performance advantages, further fueled by the spread of gRPC and contract-based APIs. However, the learning curve associated with protobuf's syntax and concepts can be a deterrent for developers, especially those already comfortable with the ubiquitous JSON format.

Enter the [JSON-to-Proto](https://json-to-proto.github.io/), a tool that aims to bridge this gap by automatically translating JSON data structures into their Protobuf equivalents. While this can offer a seemingly straightforward entry point to protobuf adoption, it's important to understand both the potential benefits and the limitations that come with such an approach.

The tool is located here: [json-to-proto.github.io](https://json-to-proto.github.io/).

## **Potential use cases**
While it's essential to recognize the converter's limitations, there are specific scenarios where it can provide significant value:

- **Rapid prototyping:** When experimenting with a new project or feature, the converter enables developers to quickly generate protobuf schemas from existing JSON data structures, accelerating the initial development cycle.

- **Legacy system migration:** For organizations transitioning from JSON-based systems to protobuf, the converter can automate the initial conversion of large datasets, easing the migration process.

- **API design exploration:** During the API design phase, the converter can be used to explore different protobuf schema representations based on sample JSON requests and responses, facilitating discussions and decision-making.

- **Educational purposes:** The converter can serve as a valuable learning tool, allowing developers to visualize how JSON structures translate into protobuf schemas, aiding their understanding of protobuf concepts.

- **Limited protobuf expertise:** In situations where in-depth protobuf knowledge is lacking within a team, the converter can provide a temporary solution while developers gradually upskill.

It's crucial to remember that the converter is most effective when used strategically in conjunction with a growing understanding of protobuf. While it can provide a helpful starting point, developers should always strive to refine the generated schemas and leverage protobuf's full capabilities for optimal performance and maintainability. 


## Getting it right
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

This is pretty decent. If I'm going to reuse `Category` and `Tags` in other types, I would move it from outside of `SomeMessage`, but generally this did extremely well. It made some assumptions that `uint32` is what all IDs look like, which may be fair but also may be wrong. Let's dig into the ways this tool could lead us astray.

## The Price of Simplicity

### Loss of Expressiveness
JSON, while flexible, lacks the type richness and structural constraints of protobuf. This means that converting JSON to protobuf can result in a loss of information and reduced clarity.
- **Enums**: Protobuf's enums provide a clear way to define a restricted set of values. JSON, lacking this concept, forces converters to rely on string representations, which are less type-safe and can lead to runtime errors.
- **Numeric Types**: JSON's "number" type is ambiguous, encompassing a wide range of numeric representations. Protobuf, on the other hand, offers a selection of specific integer and floating-point types, enabling more efficient storage and processing. Converters may struggle to accurately infer the intended protobuf type from a JSON number.
- **Maps**: Converters probably can't pick up on a field that has arbitrary key/value pairs. Since everything is a "JSON Object" it's hard to tell the difference between a `map<string, string>` and `message { ... }`.
- **oneOf**; This tool doesn't detect situations where `oneOf` would be useful.

### Imperfect Conversions
The complexity of data structures and the ambiguity inherent in JSON can lead to conversion errors or less-than-ideal protobuf schemas.
- **Ambiguous input, ambiguous output**: If your examples leave a field as "null" then you may end up with `google.protobuf.Any` types, which is usually not what you want. The example data should always be "complete" to avoid this behavior.
- **One field, two types**: When a JSON field can hold values of different types, converters might default to using `google.protobuf.Any`, a generic container type that can encapsulate any protobuf message. While convenient, this approach sacrifices type safety and can make working with the resulting protobuf data more cumbersome.

## Getting it wrong
Let's look at some concrete examples from JSON-to-proto to see where it can get things wrong and what you should do to correct it.

### Incomplete examples
If any fields are excluded or set to `null`, JSON-to-proto will use `google.protobuf.Any` as the type. This is almost always the wrong answer:
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

### Multi-typed Fields
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
The tool assumes that `user.id` is a string because it was the first type seen for that field. In this case, this is probably the wrong behavior. If you switch the order, you will get a uint32 as the type:

```json
{
    "users": [
        {"type":"user", "id": 1234},
        {"type":"admin", "id": "admin-1"}
    ]
}
```

Yields:

```protobuf
syntax = "proto3";

message SomeMessage {

    message Users {
        string type = 1;
        uint32 id = 2;
    }

    repeated Users users = 1;
}
```
This is absolutely wrong, because there's no reasonable way to encode "admin-1" as a single `uint32` value. This behavior can also happen in a list that contains multiple types:
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

This tool could definitely be smarter about this. In this particular case, I might have used the [google.protobuf.Value](https://protobuf.dev/reference/protobuf/google.protobuf/#value) well-known type to indicate that the array can have nulls, numbers, strings, booleans, or a list/struct that contains the same. This type is very useful for working with dynamically typed fields and maps very cleanly to JSON. However, this kind of field that can have multiple types are usually frowned upon as bad API design, so you may want to use this conversion process to move to a "one field: one type" design and not carry forward bad API design into protobufs as well.

## Best Practices
### Provide comprehensive and well-structured JSON examples
The converter relies on the provided JSON data to infer the protobuf schema. Make sure your examples are complete, representing all possible field types and variations within your data structure. Well-structured JSON with clear nesting and consistent naming conventions will further improve the accuracy of the conversion.

- **Manually review and refine the generated protobuf schema**: The converter is not infallible. It's crucial to carefully examine the generated protobuf schema, ensuring that it accurately reflects your data requirements and adheres to best practices. Consider aspects like field naming, data types, and the use of enums or nested messages to optimize the schema for performance and maintainability.
- **Use the converter as a starting point, not a definitive solution**:  The converter can quickly generate a protobuf schema, but it's unlikely to be perfect right away. Treat the output as a draft and iterate on it based on your specific needs and protobuf best practices.
- **Avoid ambiguous or inconsistent JSON data**: JSON's flexibility can lead to ambiguity, which can confuse the converter. Try to maintain consistent data types within fields and avoid using null values whenever possible.
- **Consider edge cases**: Ensure your JSON examples cover a wide range of possible scenarios, including edge cases and potential variations in your data. This will help the converter generate a more robust and adaptable protobuf schema.
- **Use protobuf's other features**: Once you've generated an initial schema, explore how protobuf's advanced features, such as enums, oneof fields, and maps, can be used to further refine and optimize your data representation.

By adhering to these best practices, you can leverage the JSON-to-Proto converter effectively while minimizing potential issues and ensuring the resulting protobuf schema aligns with your requirements and best practices.

## A useful tool, but not a replacement for understanding protobuf
JSON to protobuf converters *can* be helpful in specific scenarios, especially for quick experimentation and legacy system integration. However, they should not be seen as a substitute for gaining a solid understanding of protobuf's concepts and syntax, especially as gRPC and contract-based APIs continue to gain traction.

By investing time in [learning protobuf](https://protobuf.dev/) directly, developers can leverage its full potential, designing schemas that are not only more expressive but also optimized for performance and maintainability.

Remember, a converter like this is a tool, not a magic wand. Use it wisely, but never let it replace the power of knowledge and try to recognize when this tool might not actually help you.
