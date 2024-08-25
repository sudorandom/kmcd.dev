---
categories: ["article"]
tags: ["protobuf", "grpc", "api", "rpc", "workflow"]
date: "2024-08-27T10:00:00Z"
description: "Tools and tricks for developing with protobuf."
cover: "cover.jpg"
images: ["/posts/protobuf-workflow/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Improving the Protobuf Workflow"
slug: "protobuf-workflow"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/protobuf-workflow/
featured: true
draft: true
---

Protocol Buffers (protobuf) are a language-agnostic binary serialization format developed by Google. They offer a compact and efficient way to structure data for storage or transmission, making them ideal for applications like gRPC services, data storage, and inter-service communication. However, the traditional protobuf workflow can sometimes feel a bit cumbersome with tooling that isn't quite made to be easy to use and, worse, are easy to use incorrectly. In this article, we'll explore some tools that streamline this process, making protobuf development more enjoyable and productive. But first, let's recap how the "traditional" protobuf workflow typically works.

## Traditional Workflow
Here's what a typical workflow can look like when working with protobufs:

1. **Define the Protobuf File:**  You create a `.proto` file, defining your message structures, enums, and services (if using gRPC). 
2. **Compile with `protoc`:** You use the `protoc` compiler to generate code in your desired languages (e.g., Go, Java, Python, Rust, etc.) from your `.proto` files.
3. **Implement and Use:** You implement the server-side logic (if applicable) using the generated server stubs and utilize the generated client code to interact with your protobuf-based services.

### What's Missing?
While this workflow gets the job done, it leaves a lot of room for improvement:

- Error-Prone Manual Steps: The process often involves manual compilation and code generation, increasing the risk of human errors. Most people introduce a `Makefile` or a bash script to handle calling `protoc`. Surprisingly, this can also be error-prone.
- This workflow doesn't include a standardized way to handle protobuf dependencies. Traditionally, this involves the Makefile or bash script that I just mentioned to download protobufs from *places*.
- Lack of Consistency: Without proper tooling, it's challenging to maintain consistent formatting and styling across multiple .proto files.
- Breaking Changes: Manually tracking changes to your protobufs and ensuring backward compatibility can be tedious and error-prone.
- No Mocking or Prototyping: It takes a good amount of effort to make a gRPC service, even if you just want to use mock data.

Let's visualize this traditional workflow to highlight these pain points:

{{< diagram >}}
{{< image src="workflow-legacy.svg" height="600px" class="center" >}}
{{< /diagram >}}

Fortunately, the Protobuf ecosystem has evolved dramatically in the last few years, and most of the advancement is from third-parties (not Google), which is exciting. So let's cover parts of the modern workflow that you won't see in a gRPC or protobuf tutorial.

## The next generation of protobuf tooling
{{< image src="suprise.png" width="600px" class="center" >}}

Now, let's explore some tools that enhance this workflow.

### JSON to Proto
[JSON to Proto](https://json-to-proto.github.io/) is a fantastic online tool that simplifies the creation of protobuf definitions. You can paste in sample JSON data, and it will generate a corresponding `.proto` file for you. This is particularly useful when you're starting with existing JSON data or want to quickly prototype a protobuf schema. Learn more from [the Github repo](https://github.com/json-to-proto/json-to-proto.github.io).

### Protobuf Pal
[Protobuf Pal](https://www.protobufpal.com/) is a browser-based protobuf editor designed to streamline the creation and editing of `.proto` files. It offers features like syntax highlighting, error checking, and auto-completion, making it easier to write valid protobuf definitions.

### Buf CLI
[Buf](https://buf.build/) is a comprehensive toolkit designed to make working with protobufs a breeze. It offers several powerful features:

- **`buf generate`:**  A replacement for `protoc` that provides consistent code generation across different languages and environments. [Read more here.](https://buf.build/docs/generate/tutorial)
- **`buf lint`:**  A linter that helps you maintain clean and consistent protobuf definitions, catching potential issues early. [Read more here.](https://buf.build/docs/lint/tutorial)
- **`buf format`:** An opinionated formatter to ensure consistent styling across your `.proto` files. [Read more here.](https://buf.build/blog/introducing-buf-format)
- **`buf curl`:** proves useful not only for development but also for testing. You can use it to send gRPC requests from your terminal, making it easy to verify the behavior of your services. It can craft requests using server reflection, local protobuf files, descriptor files or from a reference to the buf.build registry. [Read more here.](https://buf.build/docs/curl/usage)

```shell
$ buf curl --list-methods https://demo.connectrpc.com
connectrpc.eliza.v1.ElizaService/Converse
connectrpc.eliza.v1.ElizaService/Introduce
connectrpc.eliza.v1.ElizaService/Say
$ buf curl -d '{"sentence":"Hello! I need some help, doc"}' https://demo.connectrpc.com/connectrpc.eliza.v1.ElizaService/Say
{
  "sentence": "Hello...I'm glad you could drop by today."
}
```

### Buf Schema Registry (BSR)
[The Buf Schema Registry (BSR)](https://buf.build/docs/bsr/introduction) is the missing package manager for Protobufs, but it does a bit more than that. Not only does it allow you to push versioned schemas to one place, but it also has cool features like [automatic code generation](https://buf.build/docs/bsr/generated-sdks/overview). The idea is that you can just import the package in your favorite language and magically you have your server stubs and clients in the language of your choice (as long as it's Go, Typescript/Javascript, Java/Kotlin, Swift, Python or Rust).

With BSR, you no longer need bespoke Makefile or bash scripts to pull down protobuf dependencies.

### Buf Studio
[Buf Studio](https://buf.build/studio) is an interactive web UI for all your gRPC and Protobuf services stored on the Buf Schema Registry. With Buf Studio you can craft gRPC/gRPC-Web/Connect requests using images on the buf registry. Buf Studio uses those protobuf schemas to support autocompletion of these requests, which is super cool. It can also use an agent that is built into the `Buf CLI` to proxy requests from internal networks, making this web-based tool a bit more flexible.

### Postman
[Postman](https://blog.postman.com/postman-now-supports-grpc/), a popular API testing tool, now supports gRPC. You can leverage its familiar interface to construct and send gRPC requests, making it a convenient option for testing your protobuf services.

### Insomnia
[Insomnia](https://docs.insomnia.rest/insomnia/grpc) is another API testing platform that has added gRPC support. Similar to Postman, it allows you to design and execute gRPC requests within its user-friendly environment.

### k6
[k6](https://k6.io/docs/using-k6/protocols/grpc/) is a powerful load testing tool that can be used to simulate heavy traffic on your gRPC services. It helps you identify performance bottlenecks and ensure your services can handle real-world loads.

### FauxRPC
I couldn't write this article without promoting my own tool, [FauxRPC](https://fauxrpc.com). FauxRPC is a tool that enables developers to quickly generate mock gRPC servers from protobuf definitions, facilitating early API development and testing. By incorporating FauxRPC into your workflow, you can easily create realistic mock services that simulate real gRPC server behavior, allowing you to test your client implementations and identify potential issues without the need for a fully implemented backend. This streamlined prototyping and testing process ultimately fosters faster iteration and more robust API development.

With a single command, you can have a server running with fake data!
```shell
$ buf build buf.build/connectrpc/eliza -o eliza.binpb
$ fauxrpc run --schema=eliza.binpb
FauxRPC (0.0.16 (97e4c8caf9d3c22387a393180e00bce40b2834c6) @ 2024-08-22T18:42:56Z; go1.22.4)
Listening on http://127.0.0.1:6660
OpenAPI documentation: http://127.0.0.1:6660/fauxrpc.openapi.html

Example Commands:
$ buf curl --http2-prior-knowledge http://127.0.0.1:6660 --list-methods
$ buf curl --http2-prior-knowledge http://127.0.0.1:6660/[METHOD_NAME]
```

Learn more about it in [my previous post announcing it](/posts/fauxrpc/) or [the documentation website](https://fauxrpc.com/docs/intro/).

## New Workflow
The modern protobuf workflow leverages specialized tooling to overcome the limitations of the traditional approach. Let's explore how these tools fit into the enhanced workflow:

{{< diagram >}}
{{< image src="workflow-new.svg" width="900px" class="center" >}}
{{< /diagram >}}

### More ways to start making protobuf files
Developers can still define services, messages, and enums directly in the .proto file. However, tools like JSON-to-Proto and Protobuf Pal provide visual aids and assistance in creating and editing .proto files, reducing errors and improving productivity.

### Automated Code Generation and Management
Buf's `buf generate` command streamlines the code generation process with a set of declarative configuration files, ensuring consistency across different languages and platforms. Buf's dependency management capabilities eliminate the need for manual scripts to handle protobuf dependencies.

### Enhanced Quality Control
- Buf's `buf lint` enforces coding standards and best practices, catching potential issues early in development.
- Buf's `buf format` automatically formats your .proto files, ensuring consistency and readability.
- Buf's `buf breaking` change detection helps you avoid introducing changes that could disrupt existing clients.

All of these put together mean that issues in your API schema are discovered automatically, as soon as possible.

### Streamlined Testing and Prototyping
Buf's `buf curl` and Buf Studio's built-in gRPC client enable quick testing and interaction with your services. FauxRPC facilitates rapid prototyping by allowing you to mock gRPC services without implementing the actual server-side logic.

## Conclusion
The traditional protobuf workflow, while functional, can involve manual steps and potential pitfalls. By incorporating tools like the Buf CLI, JSON-to-Proto, Protobuf Pal, and FauxRPC into your development process, you can significantly enhance your productivity and ensure the quality of your protobuf definitions. Note that I haven't come close to outlining all of the different tools that you can use with Protobuf. So don't hesitate to explore these tools and discover how they can revolutionize your Protobuf development experience!
