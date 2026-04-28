---
categories: ["tutorial"]
tags: ["api", "protobuf", "grpc", "openapi", "avro", "thrift", "trpc", "connectrpc", "graphql", "twirp"]
date: "2026-04-28"
description: "Building for Scale: Why contract-based APIs are the future."
cover: "cover.svg"
images: ["/posts/api-contracts/cover.svg"]
title: "Building APIs with Contracts"
slug: "api-contracts"
type: "posts"
devtoId: 1815913
devtoPublished: true
devtoSkip: false
canonical_url: https://kmcd.dev/posts/api-contracts/
mastodonID: "112277335329654214"
---

{{< disclaimer >}}
This article was originally published in April 2024. It was republished in April 2026 after some significant editing and modernization.
{{< /disclaimer >}}

In today's interconnected world, APIs (Application Programming Interfaces) are the glue that connects computers. They allow different applications to talk to each other, share data, and perform actions. However, traditional methods of creating APIs often lead to frustrating challenges: breaking changes in JSON APIs, silent failures due to missing fields, frontend and backend drift, or schema mismatches that result in the classic "works on my machine" excuse.

Imagine a real-world scenario where the backend team renames a `userId` field to `user_id` and deploys their changes. Instantly, the frontend checkout process breaks in production because the API had no strict enforcement to catch the mismatch.

This is where **contract-based APIs** come in. A contract-based API is one where the schema is defined first in a formal specification, and both client and server are generated or validated against that contract. They reduce ambiguity and enforce consistency across services.

### The Power of Pre-defined API Contracts

A contract-based API defines exactly what data can be exchanged, in what format, and what actions can be performed. This strict, pre-defined agreement unlocks several immediate advantages:

* **Improved Developer Experience:** Developers on both sides (client and server) have a clear understanding of what is expected, making integration smoother.
* **Automated Documentation:** Contracts serve as self-documenting artifacts. This reduces the need for manual documentation maintenance and ensures the docs stay in sync with the actual API implementation.
* **Reduced Errors:** Mismatched data formats or API changes become less likely, leading to fewer bugs. Contracts act as a validation layer that catches potential issues early.
* **Easier Integration:** Contracts act as a single source of truth. Developers can quickly understand how to interact with the API without extensive back and forth communication.
* **Streamlined Development:** These APIs often enable tools to automatically generate code for both client and server implementations. This eliminates manual boilerplate so you can focus on core logic.

### Protobuf: The Language of APIs

In modern distributed systems, the foundation of many contract-based APIs lies in [**Protocol Buffers (protobuf)**](https://protobuf.dev/). It is a language-neutral data format specifically designed for structured messages. 

Unlike JSON, which is a text-based format designed to be human-readable, Protobuf is a **binary format**. This means you trade the ability to natively read the raw data in transit for significant performance gains:

* **Smaller Message Sizes:** Protobuf messages are compact and efficient, which leads to faster transmission and reduced bandwidth usage.
* **Faster Parsing:** Parsing binary protobuf messages is significantly faster compared to traditional formats like JSON or XML.
* **Built-in Versioning:** Protobuf uses field numbers (the `= 1`, `= 2` in the code below) to identify data. This allows for excellent backward and forward compatibility. You can add new fields without breaking older clients that do not know about them yet.
* **Cross-language Compatibility:** Protobuf definitions are language-agnostic. Code for interacting with the API can be generated for almost any modern programming language.

Because the data is binary, you cannot simply open your browser's network tab and read the payloads by default. You will usually need to rely on modern browser extensions (like the gRPC-Web or Connect dev tools) to decode the traffic. It also requires setting up specialized tooling and build steps to compile the generated code.

Here is a basic example of a `.proto` file defining messages for a user and an address:

```protobuf
syntax = "proto3";

message User {
  string name = 1;
  int32 id = 2;
  string email = 3;
  Address address = 4;
}

message Address {
  string street = 1;
  string city = 2;
  string state = 3;
  string zip = 4;
}
```

In this example, the `User` message has fields for name, ID, email, and an `Address` message. These defined structures ensure consistent data exchange between applications.

> **Key idea:** Protobuf relies on immutable field numbers instead of field names. This golden rule guarantees backward and forward compatibility.

### gRPC: Building APIs on a Solid Foundation

**gRPC (gRPC Remote Procedure Call)** is a high-performance framework that builds upon protobuf's strengths. It provides a powerful way to implement remote procedure calls, allowing applications to interact using clients generated for each language. 

#### Introducing Services and Request/Response Types with gRPC

We can expand the `.proto` file to define a service called `UserService` with methods for user management:

```protobuf
syntax = "proto3";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (User) {}
  rpc GetUser(GetUserRequest) returns (User) {}
}

message CreateUserRequest {
  User user = 1;
}

message GetUserRequest {
  int32 id = 1;
}
```

This example defines a `UserService` with two methods: `CreateUser` and `GetUser`. Each method takes a specific request message and returns a response. 

Notice how clear the intention is. A helpful mental model to contrast modern APIs is:
* **REST** is resource-oriented (relying on URLs and HTTP verbs).
* **gRPC** is action-oriented (relying on explicit methods).

A reader of this spec does not have to map vague HTTP verbs like "POST" to actions like "create." Also, these method names are [greppable](https://en.wiktionary.org/wiki/greppable). It is trivial to locate every use of `CreateUser` across several repositories, making refactoring and impact analysis much easier.

#### Server Reflection
Another powerful feature of the gRPC ecosystem is **Server Reflection**. This allows clients or debugging tools (like Postman or grpcurl) to query the server at runtime to discover the available services and methods. This eliminates the need to distribute `.proto` files to developers just so they can explore the API structure.

### Distributing API Contracts

Defining a contract is only half the battle. How do the frontend and backend teams actually share that `.proto` file? If the schema is not easily accessible, the contract is useless.

In practice, teams usually solve this distribution problem in one of three ways:
1.  **Monorepos:** Storing the backend, frontend, and API definitions in a single repository so all code shares the same source of truth.
2.  **Package Managers:** Generating the client SDKs in a CI/CD pipeline and publishing them as internal NPM, Maven, or Go packages.
3.  **Schema Registries:** Using dedicated tools like the [Buf Schema Registry](https://buf.build/) to manage, version, and distribute Protobuf files securely across an organization.

### What about public APIs?

Historically, strict RPC contracts were tough for external, public-facing APIs. If your primary consumers were third-party developers, handing them a raw Protobuf file or expecting them to set up gRPC clients caused massive friction. They just wanted to use standard REST with JSON. 

This is where tools like [**ConnectRPC**](https://connectrpc.com/) shine. ConnectRPC allows you to define your API using Protobuf, but it automatically exposes endpoints that support standard HTTP/1.1 and JSON serialization as a fallback format. 

This hybrid approach also solves the local debugging problem. You can configure ConnectRPC to use JSON during local development specifically so you can read the network tab in plain text, and then flip it to highly efficient binary for production. In practice, you write Protobuf once, and get both gRPC and REST/JSON APIs for free.

Even better, because the source of truth is still Protobuf, you can use ecosystem plugins to automatically generate an OpenAPI specification directly from your `.proto` files. You get a highly maintainable, contract-driven architecture on the backend, while your external users can still `curl` standard REST endpoints, read plain JSON, and explore your API via a generated Swagger UI. It offers the best of both worlds without compromising the developer experience on either side.

> **Key idea:** Tools like ConnectRPC allow you to maintain strict internal Protobuf contracts while exposing standard REST/JSON APIs to external consumers.

## Alternatives

While Protobuf and gRPC are a powerful duo, there are other contract-based API solutions to consider depending on your architecture:

* [**OpenAPI (Swagger)**](https://www.openapis.org/): Contracts are not exclusive to RPC. You can use OpenAPI to define strict contracts for RESTful services. However, a harsh reality of the industry is that OpenAPI specs often drift from the actual code because they are bolted on after the fact. To make OpenAPI truly safe, teams must rely on strict framework integration (like FastAPI in Python or tsoa in Node) where the code generates the spec, or vice versa.
* [**GraphQL**](https://graphql.org/): Arguably the most mainstream contract-driven API paradigm for frontend developers. Its strictly typed schema defines the exact shape of the available data. Unlike gRPC, which has fixed responses, GraphQL allows the client to dictate the exact payload it wants to receive.
* [**Twirp**](https://twitchtv.github.io/twirp/): Developed by Twitch, Twirp is a lightweight RPC framework built on top of Protobuf and HTTP/1.1. It shares similarities with ConnectRPC but focuses on absolute simplicity. It avoids the complexity of HTTP/2 and gRPC streams while still providing generated clients, making it an excellent alternative if full gRPC is overkill for your needs.
* [**Thrift**](https://thrift.apache.org/): Originally developed at Facebook, Thrift is a language-neutral protocol for defining service contracts similar to Protobuf. It is often found in large-scale data environments and supports various RPC protocols.
* [**tRPC**](https://trpc.io/): This tool defines the API schema directly in TypeScript code to be reused on both the client and the server. While it often pairs with libraries like Zod for runtime validation, it lacks true language-agnostic safety across the network boundary since it relies entirely on a TypeScript ecosystem.
* [**Avro**](https://avro.apache.org/): This format uses JSON-like schemas but stores data in a compact binary format. It is a staple in the Apache Kafka ecosystem for streaming data pipelines. It handles schema evolution differently than Protobuf (often sending the schema alongside the data), making it highly flexible for dynamic systems.

## When NOT to Use API Contracts

While these tools are powerful, they are not a silver bullet. You should reconsider using strict API contracts if:
* **You are building small projects or MVPs:** The initial setup, code generation, and boilerplate overhead might slow down your speed of delivery when rapid iteration is the top priority.
* **Simplicity for external consumers outweighs strict contracts:** If you are building a straightforward public API and are not using a hybrid tool like ConnectRPC, raw JSON over REST remains the path of least resistance for third-party developers.
* **Your team lacks tooling maturity:** Implementing gRPC or Protobuf requires solid CI/CD pipelines and a team that is comfortable managing build steps, code generation, and backward-compatible schema evolutions.

> **Key idea:** Strict API contracts add overhead and may not be suitable for small MVPs, simple public APIs, or teams lacking tooling maturity.

## Conclusion

Contract-based APIs offer a significant advantage in building robust and scalable communication between applications. Protobuf and gRPC provide a powerful combination for defining clear contracts and generating highly efficient code. 

As a general rule of thumb: if you are building an early-stage prototype, stick to what is fast and familiar. But if you are scaling a complex system across multiple teams and services, contract-based APIs transition from a nice-to-have to an absolute necessity. Once multiple teams depend on your API, contracts stop being optional. They are how you avoid chaos.
