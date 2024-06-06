---
categories: ["article"]
tags: ["connectrpc", "grpc", "unittest"]
date: "2024-06-11T11:00:00Z"
description: "Learn how to test your ConnectRPC services."
cover: "cover.jpg"
title: "Unit Testing ConnectRPC Servers"
slug: "connectrpc-unittests"
type: "posts"
canonical_url: https://kmcd.dev/posts/connectrpc-unittests
---

If you've embarked on the journey of building efficient and scalable RPC systems with [ConnectRPC](https://connectrpc.com/), you might be pondering the best way to ensure the reliability and correctness of your services. Unit testing is the obvious tool for this, providing a safety net that catches bugs early and empowers you to refactor code fearlessly. In the ConnectRPC world, unit testing can be daunting due to its integration with Protocol Buffers and the client-server architecture. In this guide, we'll unravel the mysteries of unit testing ConnectRPC services, while arming you with practical examples and advanced techniques to fortify your codebase.

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
2. **In-Memory Server Testing:** This approach simulates a ConnectRPC server running in memory. It's helpful when you want to test the interactions between your client and server code but is usually "overkill" unless you're wanting to test interceptors or HTTP middleware.

## Hands-On: Unit Testing Example

Let's write some unit tests for a simple ConnectRPC service:

```go
package main

import (
    "context"
    "errors"
    "testing"

    v1 "your/module/path/v1" // Replace with your protocol buffers package path
    v1connect "your/module/path/v1/v1connect"
)

type greeterService struct{}

var _ v1connect.GreeterService = (*greeterService)(nil)

func (s *greeterService) Greet(ctx context.Context, req *v1connect.GreetRequest) (*v1connect.GreetResponse, error) {
    if req.Msg.Name == "" {
        return nil, errors.New("missing name")
    }
    return &v1connect.GreetResponse{Greeting: "Hello, " + req.Name}, nil
}

func TestGreet(t *testing.T) {
    service := &greeterService{}

    // Direct service testing
    response, err := service.Greet(context.Background(), &v1connect.GreetRequest{Name: "Alice"})
    if err != nil {
        t.Fatalf("Greet failed: %v", err)
    }
    if response.Greeting != "Hello, Alice" {
        t.Errorf("Unexpected greeting: got %q, want %q", response.Greeting, "Hello, Alice")
    }
}

```

**Explanation:**

1. **Type Assertion:** The line `var _ v1connect.GreeterService = (*greeterService)(nil)` is a type assertion. It ensures that your `greeterService` struct correctly implements the `GreeterService` interface defined by your Protocol Buffers. This relies on a trick of the Go syntax that will try to bind a variable `_` using the `v1connect.GreeterService` type. If the given `greeterService` pointer doesn't implement the interface then the compiler should complain about what specific methods are missing and which method signatures don't match.
2. **Direct Service Testing:** The `TestGreet` function creates an instance of your `greeterService` and directly calls its `Greet` method. We then assert that the response matches our expectations.


## Hands-On: Table Tests
Now that we wrote a single unit test, the next example will show you how to utilize table tests in order to easily write more test cases. You will see code that looks like this in well-tested Go repositories.

```go
package main

import (
    "context"
    "testing"

    v1 "your/module/path/v1" 
    v1connect "your/module/path/v1/v1connect"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// ... (greeterService definition remains the same)

func TestGreetTable(t *testing.T) {
    service := &greeterService{}
    cancelledCtx, cancel := context.WithCancel(context.Background())
    cancel()

    testCases := []struct {
        name    string
        ctx     context.Context
        req     *v1connect.GreetRequest
        want    *v1connect.GreetResponse
        wantErr string
    }{
        {
            name:    "Success",
            req:     &v1connect.GreetRequest{Name: "Bob"},
            want:    &v1connect.GreetResponse{Greeting: "Hello, Bob"},
            wantErr: "",
        },
        {
            name:    "Empty Name",
            req:     &v1connect.GreetRequest{},
            want:    nil, // Expecting an error
            wantErr: "missing name",
        },
        {
            name:    "Context Cancelled",
            ctx:     cancelledCtx,
            req:     &v1connect.GreetRequest{Name: "Alice"},
            want:    nil,
            wantErr: "context canceled",
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            ctx := tc.ctx
            if ctx == nil {
                ctx = context.Background()
            }

            got, err := service.Greet(ctx, tc.req)
            if tc.wantErr != "" {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tc.wantErr)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tc.want, got)
            }
        })
    }
}

```

**Explanation:**

1. **Table Setup:** A `testCases` slice defines scenarios with varying inputs (`req`), expected outputs (`want`), and potential errors (`wantErr`).
2. **Context Cancellation:** The test case "Context Cancelled" simulates a cancelled context by creating a context with `context.WithCancel` and immediately calling `cancel()`.
3. **Testify Assertions:** The `require` package is used for assertions that should stop the test if they fail (e.g., requiring an error). The `assert` package is used for assertions that are not critical for continuing the test. Typically, errors during test setup use `require` and assertions on the results of the test use `assert`.

**Key Improvements:**

* **Table Tests:** More concisely expresses multiple test scenarios.
* **Error Simulation:** Shows how to test your service's behavior under error conditions like context cancellation.
* **Testify:** Enhances readability and provides more expressive assertions.

## 
```go
package main

import (
	"context"
	"errors"
	"net"
	"testing"

	v1 "your/module/path/v1"            // Replace with your protocol buffers package path
	v1connect "your/module/path/v1/v1connect"
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto" // for in-memory server

	// For in-memory server
	"github.com/bufbuild/connect-go/bufconn"
)

// ... (greeterService definition remains the same)

func TestGreetWithServer(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024) // In-memory listener
	t.Cleanup(func() { lis.Close() })  // Ensure listener is closed at the end

	server := v1connect.NewGreeterServiceHandler(&greeterService{},)
	go func() {
		if err := connect.Serve(lis, server); err != nil {
			t.Fatalf("Server exited with error: %v", err)
		}
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	client := v1connect.NewGreeterServiceClient(
		connect.NewClient[v1.GreetRequest, v1.GreetResponse](dialer),
	)

	response, err := client.Greet(context.Background(), &v1.GreetRequest{Name: "Alice"})
	if err != nil {
		t.Fatalf("Greet failed: %v", err)
	}
	if response.Greeting != "Hello, Alice" {
		t.Errorf("Unexpected greeting: got %q, want %q", response.Greeting, "Hello, Alice")
	}
}
```

## Conclusion: Test with Confidence

In this guide, we've explored the "why" and "how" of unit testing your ConnectRPC services. By embracing unit testing as a core part of your development workflow, you'll create more robust, reliable, and maintainable RPC systems. Remember, effective testing isn't just about fixing bugs – it's about building confidence in your codebase and enabling you to iterate and evolve your services with ease.

**Next Steps:**

* **Go Beyond the Basics:** Explore more advanced testing techniques, such as mocking dependencies for more complex scenarios.
* **Integrate with Your CI/CD:** Automate your unit tests to run as part of your continuous integration and continuous delivery (CI/CD) pipeline for immediate feedback on code changes.
* **Share Your Knowledge:** Help the ConnectRPC community grow by sharing your own testing strategies and experiences!

Ready to put your newfound knowledge into action? Start writing those unit tests and watch your ConnectRPC projects thrive!
