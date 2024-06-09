---
categories: ["article", "tutorial"]
tags: ["connectrpc", "golang", "unittest", "grpc", "testing", "backend", "tutorial"]
date: "2024-06-11T11:00:00Z"
description: "Learn how to test your ConnectRPC services."
cover: "cover.jpg"
title: "Unit Testing ConnectRPC Servers"
slug: "connectrpc-unittests"
type: "posts"
canonical_url: https://kmcd.dev/posts/connectrpc-unittests
---

If you've embarked on the journey of building efficient and scalable RPC systems with [ConnectRPC](https://connectrpc.com/), you might be pondering the best way to ensure the reliability and correctness of your services. Unit testing is the obvious tool for this, providing a safety net that catches bugs early and empowers you to refactor code fearlessly. In the ConnectRPC world, unit testing can be daunting due to its integration with Protocol Buffers and the client-server architecture. In this guide, we'll unravel the mysteries of unit testing ConnectRPC services, while arming you with practical examples and advanced techniques to fortify your codebase.

First off, the full source code can be found [on github](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2024/connectrpc-unittests). If it helps, feel free to download, run, and modify as you see fit!

## Why Unit Test?

Before we dive in, let's address the "why." Unit testing your ConnectRPC servers brings a multitude of benefits:

* **Isolation:** Focus on testing individual components in isolation, making it easier to pinpoint and fix issues.
* **Speed:** Unit tests execute quickly, providing fast feedback during development.
* **Refactoring Confidence:** When you have solid unit tests, you can refactor your code with confidence, knowing that the tests will catch any unintended consequences.
* **Documentation:** Well-written unit tests can serve as living documentation, illustrating how your code is meant to be used.
* **Bug Prevention:** A good suite of unit tests can help you catch bugs early on, before they become harder and more expensive to fix.

## Testing Strategies with ConnectRPC

ConnectRPC, built upon the Protocol Buffers ecosystem, offers a couple of primary approaches to unit testing:

1. **Direct Service Testing:** This is ideal for unit testing but it's not always possible. You directly call the methods of your service implementation (typically a struct in Go), bypassing any client and server networking.
2. **Server Testing:** This approach creates an actual ConnectRPC server with `net/http/httptest`. It's helpful when you want to test the interactions between your client and server code but is usually "overkill" unless you're wanting to test interceptors or HTTP middleware.

## Hands-On: Our example service
Here is the protobuf file that we're using for our example:
{{% render-code file="go/greet/v1/greet.proto" language="protobuf" %}}
{{< aside >}}
See the full source at Github: {{< github-link file="go/greet/v1/greet.proto" >}}.
{{</ aside >}}

And here is the resulting server implementation:
{{% render-code file="go/endpoints.go" language="go" start="// start" %}}
{{< aside >}}
See the full source at Github: {{< github-link file="go/endpoints.go" >}}.
{{</ aside >}}

Here we defined our `greetv1connect.GreetServiceHandler` implementation. It implements the `Greet` method defined in the protobuf file alove. Since this is for demonstration purposes, all we do is check to see if the given `name` is empty, sleep for 10 milliseconds to simulate a network call and returns the greeting as `"Hello, {name}"`.

A keep observer might notice that this file contains our first "test". The line `var _ greetv1connect.GreetServiceHandler = (*greeterService)(nil)` is a way of doing a type assertion in Go. It ensures that your `greeterService` struct correctly implements the `GreeterService` interface defined by the protobuf file above. This relies on a trick of the Go syntax that will try to bind a variable `_` using the `greetv1connect.GreetServiceHandler` type. If the given `greeterService` pointer doesn't implement the interface then the compiler should complain about what specific methods are missing and which method signatures don't match.

## Hands-On: Direct Service Testing Example

Let's write some unit tests for a simple ConnectRPC service:
{{% render-code file="go/direct_test.go" language="go" start="// start" %}}
{{< aside >}}
See the full source at Github: {{< github-link file="go/direct_test.go" >}}.
{{</ aside >}}

**Explanation:**
The `TestGreet` function creates an instance of your `greeterService` and directly calls its `Greet` method. We then assert that the response matches our expectations. This is, by far, the simplest method for testing a ConnectRPC service.

## Hands-On: Table-Driven Tests with Testify
Now that we wrote a single unit test, the next example will show you how to utilize table tests in order to easily write more test cases. You will see code that looks like this in well-tested Go repositories.

{{% render-code file="go/table_test.go" language="go" start="// start" %}}
{{< aside >}}
See the full source at Github: {{< github-link file="go/table_test.go" >}}.
{{</ aside >}}

**Explanation:**

1. **Table Setup:** A `testCases` slice defines scenarios with varying inputs (`req`), expected outputs (`want`), and potential errors (`wantErr`).
2. **Context Cancellation:** The test case "Context Cancelled" simulates a cancelled context by creating a context with `context.WithCancel` and immediately calling `cancel()`.
3. **Testify Assertions:** The `require` package is used for assertions that should stop the test if they fail (e.g., requiring an error). The `assert` package is used for assertions that are not critical for continuing the test. Typically, errors during test setup use `require` and assertions on the results of the test use `assert`.

## Hands-On: Server Testing Example

Here's how you can test the same service using `net/http/httptest` server:

{{% render-code file="go/server_test.go" language="go" start="// start" %}}
{{< aside >}}
See the full source at Github: {{< github-link file="go/server_test.go" >}}.
{{</ aside >}}

- **Server Setup:** We create a ConnectRPC handler and start it with `httptest.NewServer(mux)`.
- **Client Setup:** We create a ConnectRPC client that connects to the server that we just created.
- **Test Interaction:** We use the client to call the Greet method and assert the response, just like in the direct service testing example.

## Conclusion: Test with Confidence

In this guide, we've explored the "why" and "how" of unit testing your ConnectRPC services. By embracing unit testing as a core part of your development workflow, you'll create more robust, reliable, and maintainable RPC systems. Remember, effective testing isn't just about fixing bugs – it's about building confidence in your codebase and enabling you to iterate and evolve your services with ease.

The full source code can be found [on github](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2024/connectrpc-unittests).

**Next Steps:**

* **Go Beyond the Basics:** Explore more advanced testing techniques, such as mocking dependencies for more complex scenarios.
* **Integrate with Your CI/CD:** Automate your unit tests to run as part of your continuous integration and continuous delivery (CI/CD) pipeline for immediate feedback on code changes.
* **Share Your Knowledge:** Help the ConnectRPC community grow by sharing your own testing strategies and experiences!

Ready to put your newfound knowledge into action? Start writing those unit tests and watch your ConnectRPC projects thrive!
