---
categories: ["article"]
tags: ["protobuf", "grpc", "testing", "testcontainers", "fauxrpc", "mocking", "stubs"]
date: "2024-10-15"
description: "Effortless gRPC Mocking in Go"
cover: "cover.jpg"
images: ["/posts/fauxrpc-testcontainers/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "FauxRPC + Test Containers"
slug: "fauxrpc-testcontainers"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/fauxrpc-testcontainers/
---

Testing gRPC services can be tricky. You often need a real server running, which can introduce complexity and slow down your tests. Enter **[FauxRPC](https://fauxrpc.com)** + **[Testcontainers](https://testcontainers.com/)**, and small [Go package](https://github.com/sudorandom/fauxrpc/blob/main/testcontainers/testcontainers.go) that simplifies gRPC mocking.

To address challenges with testing while using gRPC services, we can leverage the power of [Testcontainers](https://testcontainers.com/), a library that lets you run throwaway, lightweight instances of common databases, web browsers, or any other application that can run in a Docker container. This allows you to easily integrate these dependencies into your automated tests, providing a consistent and reliable testing environment. By using Testcontainers, you can ensure that your tests are always running against a known and controlled version of your dependencies, avoiding inconsistencies and unexpected behavior while also simplifying test setup/teardown.

While Testcontainers provides the infrastructure, [FauxRPC](https://fauxrpc.com) takes care of the mocking itself. FauxRPC is a tool that generates fake gRPC, gRPC-Web, Connect, and REST servers from your Protobuf definitions. By combining it with Testcontainers, you gain a lightweight, isolated environment for testing your gRPC clients without relying on a real server implementation. I've made a package to make this simpler using Go but the same could be done for other languages that Testcontainers supports.

## Show by Example

### 1. Setting up the Container
```go
container, err := fauxrpctestcontainers.Run(ctx, "docker.io/sudorandom/fauxrpc:latest")
// ... error handling ...
t.Cleanup(func() { container.Terminate(context.Background()) })
```

This snippet starts a FauxRPC container using the `fauxrpctestcontainers.Run` function. The `t.Cleanup` function ensures the container is terminated after the test, keeping your testing environment clean.

### 2. Registering the Protobuf Definition
```go
container.MustAddFileDescriptor(ctx, elizav1.File_connectrpc_eliza_v1_eliza_proto)
```

You register your Protobuf file descriptor with the container. This lets FauxRPC understand the structure of your gRPC service. Now you have a fully functional FauxRPC service that mimics the services in the file descriptor that you gave it. The data is all randomly generated. Now let's test it a bit.

### 3. Making gRPC Calls
```go
baseURL := container.MustBaseURL(ctx)
elizaClient := elizav1connect.NewElizaServiceClient(http.DefaultClient, baseURL)
resp, err := elizaClient.Say(ctx, connect.NewRequest(&elizav1.SayRequest{
    Sentence: "testing!",
}))
// ... error handling and assertions ...
```

This code is getting base URL of the FauxRPC server running in the container and creating a gRPC client (using ConnectRPC). ConnectRPC isn't a requirement. You can use grpc-go instead. With this client, you can make calls to your gRPC service as you would in a real environment. In this setup, FauxRPC automatically generates responses based on your Protobuf definitions. Here you would normally have some application logic that you want to test so this code might live elsewhere. The randomly generated data might work in a few scenarios but in order to test

### 4. Defining Stub Responses
For more control over the responses you can define stubs. This allows you to simulate specific scenarios and test how your client handles different responses.

```go
container.MustAddStub(ctx, "connectrpc.eliza.v1.ElizaService/Say", &elizav1.SayResponse{
    Sentence: "I am setting this text!",
})
```

### The power of schemas and mocking

This is basically it. In these examples, I showed how you can:
1. Stand up an empty FauxRPC service
2. Populate this server with some Protobuf schema
3. Connect and use this service
4. Set stub responses

See full examples [in the FauxRPC repo](https://github.com/sudorandom/fauxrpc/blob/main/testcontainers/testcontainers_test.go).

## Benefits of using FauxRPC+Testcontainers
As demonstrated in the examples above, this approach simplifies gRPC testing by:

* **Simplified Testing:** No need to set up a real gRPC server for testing.
* **Isolated Environment:** Each test runs in its own container, preventing conflicts and ensuring consistency.
* **Increased Speed:** Tests run faster due to the lightweight nature of containers.
* **Improved Control:**  Stubbing allows you to simulate various scenarios and edge cases.

This package makes gRPC testing in Go much easier and more efficient. Give it a try and let me know what you think!

## Alternatives
In Go (and many targets for gRPC), you will get an interface that you can use to generate mock clients. Using the mock client you can usually set responses and assert on data from the request. This is a fair critique, and I feel like this strategy of testing could get you pretty far. However, there are a few reasons that FauxRPC+testcontainers is better.

First, using traditional mocking techniques will prevent you from testing your middleware. Maybe you have middleware that modifies the actual message in certain cases, returns an error, or performs some extra validation or accounting. With the FauxRPC+Testcontainers, you get to exercise the middleware code because you're talking to a real gRPC service.

In addition to that, maintaining and updating mock clients can be tedious as the gRPC API evolves. FauxRPC avoids this step by being dynamically configurable with protobuf descriptors.

Also, I like that this approach of using FauxRPC is language agnostic. Sure, the little library that makes it easier to use is written specifically in Go, but this code is very trivial to write for other languages.

Ultimately, the choice between mocking strategies depends on your specific needs and priorities.

## What's Next
Excited about the possibilities of FauxRPC Testcontainers? There's more to come! FauxRPC is still under active development and there's a lot more on the horizon! Here are a few features I'm exploring:

- Rules using CEL: Fine-grained control over stub behavior using Common Expression Language (CEL) to define complex matching conditions and response generation logic. This will enable more dynamic and flexible stubbing scenarios. Imaging having a rule saying: `req.Name == "Bob"` then return a specific stub user.
- Request Logging: Detailed logging of requests and responses to facilitate debugging and troubleshooting during test execution.
- Improved documentation: I'm currently working on a refresh of [fauxrpc.com](https://fauxrpc.com/) that includes these new features.

Have an idea for a new feature or a suggestion for improvement? We'd love to hear from you! [Open an issue](https://github.com/sudorandom/fauxrpc/issues) on the GitHub repository to share your thoughts or contribute to the project.

### References
* **FauxRPC Testcontainers:** [github.com/sudorandom/fauxrpc/testcontainers](https://github.com/sudorandom/fauxrpc/tree/main/testcontainers)
* **FauxRPC:** [fauxrpc.com](https://fauxrpc.com)
* **Testcontainers:** [testcontainers.com](https://testcontainers.com)
