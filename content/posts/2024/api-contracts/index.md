---
categories: ["tutorial"]
tags: ["api", "protobuf", "grpc", "openapi", "arvo", "thrift"]
date: "2024-04-09"
description: "Building for Scale: Why contract-based APIs are the future."
cover: "cover.jpg"
images: ["/posts/api-contracts/cover.jpg"]
title: "Building APIs with Contracts"
slug: "api-contracts"
type: "posts"
devtoId: 1815913
devtoPublished: true
devtoSkip: false
canonical_url: https://sudorandom.dev/posts/api-contracts
mastodonID: "112277335329654214"
---

In today's interconnected world, APIs (Application Programming Interfaces) are the glue that connects computers. They allow different applications to talk to each other, share data, and perform actions. However, traditional methods of creating APIs can lead to challenges, especially when dealing with versioning changes and integrating complex systems. This is where **contract-based APIs** come in, offering a more robust and reliable approach and taming some of the wildness that exists on the web.

### The Power of Pre-defined API Contracts

Imagine building a house without a blueprint. It would be chaotic and prone to errors. A contract-based API is like a detailed blueprint for communication between applications. It defines exactly what data can be exchanged, in what format, and what actions can be performed. This pre-defined agreement unlocks several advantages for developers and applications:

- **Improved Developer Experience:**
  - Developers on both sides (client and server) have a clear understanding of what's expected, making integration smoother.
  - **Automated Documentation:** Contracts serve as self-documenting artifacts, reducing the need for manual documentation creation and maintenance. This saves development time and ensures documentation stays in sync with the actual API implementation.
- **Reduced Errors:**
  - Mismatched data formats or API changes become less likely, leading to fewer bugs and headaches. Contracts act as a validation layer, catching potential issues early in the development process.
- **Easier Integration:**
  - Contracts act as a single source of truth, simplifying the process of connecting different services.  Developers can quickly understand how to interact with the API without extensive back-and-forth communication.
- **Streamlined Development:**
  - Contract-based APIs often enable tools to automatically generate code for both client and server implementations based on the defined contracts. This eliminates manual coding and boilerplate, allowing developers to focus on core functionalities.

By leveraging pre-defined contracts, you can build more robust, reliable, and maintainable APIs that streamline development efforts and improve overall communication within your application ecosystem. 

### Protobuf: The Language of APIs

In my world, the foundation of many contract-based APIs lies in [**Protocol Buffers (protobuf)**](https://protobuf.dev/). It's a language-neutral data format specifically designed for structured messages exchanged between applications. Protobuf offers several advantages:

* **Smaller Message Sizes:** Protobuf messages are compact and efficient, leading to faster transmission and reduced bandwidth usage.
* **Faster Parsing:** Parsing protobuf messages is significantly faster compared to traditional formats like JSON or XML.
* **Cross-language Compatibility:** Protobuf definitions are language-agnostic. Code for interacting with the API can be generated for many programming languages.

Defining service contracts in protobuf involves creating `.proto` files. These files specify the structure of messages (data fields and types) and define the service methods (requests and responses) that an API offers.

Here's a basic example of a `.proto` file defining messages for a user and an address:

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

In this example, the `User` message has fields for name, ID, email, and an `Address` message. The `Address` message itself has fields for street, city, state, and zip code. These defined message structures ensure consistent data exchange between applications.

### gRPC: Building APIs on a Solid Foundation

**gRPC (gRPC Remote Procedure Call)** is a high-performance framework that builds upon protobuf's strengths. It provides a powerful way to implement remote procedure calls, allowing applications to interact using clients generated for each language using (usually) types and semantics that make sense to the language. Here's how gRPC leverages protobuf:

* **Protobuf Messages for Communication:** gRPC uses protobuf messages to define the request and response data exchanged between client and server.
* **Generated Code for Seamless Interaction:** gRPC automatically generates server and client stubs (boilerplate code) from the `.proto` files. This eliminates manual coding and ensures consistency between client and server implementations.

#### Introducing Services and Request/Response Types with gRPC

Now let's expand on the concept of services within a `.proto` file. We can define a service called `UserService` with methods for user management:

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

This example defines a `UserService` with two methods: `CreateUser` and `GetUser`. Each method takes a specific request message and returns a response message. The `CreateUserRequest` message contains a `User` object to be created, while the `GetUserRequest` specifies the user ID to retrieve. This clear separation of request and response types further enhances the contract between client and server. You should also notice just how clear the intention is for the methods of UserServer. A reader of this spec doesn't have to do a mapping of extremely vague words like "POST" to more understandable actions like "create". Also notice how the `CreateUser` and `GetUser` methods are [greppable](https://en.wiktionary.org/wiki/greppable), making it trivial to locate uses of these RPCs, even across several repositories.

## Alternatives
While Protobuf and gRPC are a powerful duo, there are other contract-based API solutions to consider:

- [**OpenAPI (Swagger)**](https://www.openapis.org/): This is a popular specification for defining RESTful APIs, offering a standardized way to document and interact with web services.
- [**Thrift**](https://thrift.apache.org/): Similar to protobuf, Thrift is a language-neutral protocol for defining service contracts. It supports various RPC protocols beyond gRPC.
- [**Avro**](https://avro.apache.org/): This JSON-like data format uses schemas to ensure reliable data exchange. It's often used with Apache Kafka for streaming data pipelines.

The choice between these options depends on your specific needs. gRPC and protobuf excel for high-performance, RPC-based communication, while OpenAPI is better suited for existing RESTful APIs. Thrift is also there. Avro has some dynamic typing features and has self-describing messages but is also slower for the same reason.

## Conclusion

Contract-based APIs offer a significant advantage in building robust and scalable communication between applications. protobuf and gRPC provide a powerful combination for defining clear contracts and generating efficient code. By leveraging these technologies, you can streamline API development, improve developer experience, and ensure seamless integration within.
