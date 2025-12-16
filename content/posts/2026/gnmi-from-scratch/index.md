---
categories: ["article"]
tags: ["go", "gnmi", "connectrpc", "grpc", "networking", "telemetry"]
date: "2026-01-06T10:00:00Z"
description: "Building a gNMI server and client from scratch in Go using ConnectRPC."
cover: "cover.svg"
images: ["/posts/gnmi-from-scratch/cover.svg"]
featuredalt: "A diagram showing a client and server communicating via gNMI"
featuredpath: "date"
linktitle: ""
title: "gNMI From Scratch"
slug: "gnmi-from-scratch"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/gnmi-from-scratch/
draft: true
---

The gRPC Network Management Interface (gNMI) is a powerful standard for streaming telemetry and configuration management on network devices. While many excellent libraries and agents exist, building a small gNMI server and client from scratch is a fantastic way to understand the protocol's mechanics deeply.

In this post, we'll do just that. We'll implement a basic gNMI server and client in Go, using the modern and flexible [ConnectRPC](https://connectrpc.com/) framework instead of native gRPC-go.

### Why build it from scratch?

1.  **Deeper Understanding:** You'll gain a deeper understanding of the underlying RPC calls, data structures, and streaming patterns, moving beyond just user-level APIs.
2.  **Custom Implementations:** You might need a lightweight gNMI agent for a custom piece of hardware or a test stub for a CI pipeline.
3.  **Appreciation for the Tools:** It gives you a new appreciation for production-grade tools like `gNMIc` and the reference implementations.

### Why ConnectRPC?

Connect is a newer RPC framework that offers full compatibility with gRPC and gRPC-Web, but also adds its own protocol that works over plain HTTP/1.1 and HTTP/2.

One of its main advantages is simplicity, as the API is often more streamlined and "Go-idiomatic" than gRPC-go.

It also offers wider compatibility. Clients can communicate over simple POST requests, making it easier to debug and use in environments where gRPC is tricky, like web browsers.

Finally, it's built on Go's standard `net/http` package, which is widely used and highly performant.

### The Plan

1. **Define the Test Environment:** Outline the tools we'll use to test our implementations.
2. **Explore the Protobufs:** Look at the core `gnmi.proto` messages.
3. **Build a Server:** Implement the `gNMI` service using ConnectRPC, serving mock CPU and memory metrics.
4. **Build a Client:** Write a simple client to `Get` data and `Subscribe` to a stream of updates.
5. **Discuss Limitations:** Review what's missing for a production-ready system.

---

### Our Testing Toolkit

Before we write code, let's define our testing strategy.

We will use `gNMIc`, a versatile gNMI client, to validate our server's behavior. It's a key tool for testing gNMI compliance.

Another useful tool is `FauxRPC`, which can create mock gRPC (and Connect) servers. This is great for testing our client against a predictable server without needing our own implementation running.

For more advanced scenarios, `containerlab` can spin up a virtual network of devices, allowing us to test our client or server in a more realistic topology.

For this post, we'll focus on using `gNMIc` to test the server we build.

---

### gNMI and OpenConfig

It's important to understand the relationship between the two. gNMI is the protocol; it defines *how* a client and server exchange data, including the RPCs for `Get`, `Set`, and `Subscribe`.

OpenConfig, on the other hand, provides the data models; it defines *what* the data looks like. It offers a set of vendor-neutral, standardized data models written in the YANG modeling language. These models provide a predictable, tree-like structure for device configuration and operational state.

While you can use gNMI to transport data for any data model, it is most commonly used with OpenConfig. In our example, we will reference OpenConfig-style paths and models to simulate a real-world scenario.

### The gNMI Protobufs

The protocol is defined by `.proto` files, which you can view directly on GitHub in the [openconfig/gnmi](https://github.com/openconfig/gnmi/blob/master/proto/gnmi/gnmi.proto) repository. The core of gNMI is the `gNMI` service, which defines four RPCs: `Capabilities`, `Get`, `Set`, and `Subscribe`.

We'll focus on `Capabilities`, `Get`, and the streaming `Subscribe` RPC.

A few key messages to understand:

- **`Path`**: Represents a path to a data element in a tree-like data model (e.g., `/system/cpu/utilization`).
- **`TypedValue`**: A wrapper for the actual value, which can be a string, int, bool, etc.
- **`Notification`**: A container for an update, containing a timestamp, a path, and a set of updated values.
- **`SubscribeRequest`**: Sent by the client to initiate or modify a subscription. It specifies paths and a subscription mode (`ONCE`, `POLL`, or `STREAM`).
- **`SubscribeResponse`**: Sent by the server to the client. It can contain a `Notification` with data or a synchronization message.



---

### Step 1: Code Generation with `buf`

First, let's set up our project to generate the necessary Go code from the `gnmi.proto` file. The most reliable way to do this with `buf` is to create a `buf.yaml` file to define our dependencies, and a `buf.gen.yaml` to configure the output.

This approach correctly resolves the import paths within the `openconfig/gnmi` repository and is the standard practice for `buf`-based projects.

1.  Create a `buf.yaml` file in your project root. This tells `buf` about our dependency on `openconfig/gnmi` from the Buf Schema Registry.

    ```yaml
    version: v1
    deps:
      - buf.build/openconfig/gnmi:v0.14.1
    ```

2.  Create the `buf.gen.yaml` file. This uses `managed mode` to prefix our generated Go code with our module path, and places the output in a `gnmi/gen` directory.

    ```yaml
    version: v2
    managed:
      enabled: true
      go_package_prefix:
        default: github.com/sudorandom/kmcd.dev
    plugins:
      - remote: buf.build/protocolbuffers/go:v1.36.11
        out: gnmi/gen
      - remote: buf.build/connectrpc/go:v1.19.1
        out: gnmi/gen
    ```

Now, run `buf generate` against the dependency:

```bash
buf generate buf.build/openconfig/gnmi
```

This will fetch the dependency, resolve its imports, and create the Go files in the `gnmi/gen` directory.

---

### Step 2: Building the gNMI Server

Let's start by building the server. Our server needs to listen for incoming connections and respond to gNMI RPCs.

#### Server Responsibilities

A gNMI server has a few core jobs. First, it must listen for connections by running an HTTP server that can understand gRPC-style requests. We'll use Connect's `h2c` package to allow both HTTP/1.1 and HTTP/2 (for gRPC compatibility) without TLS. Second, it needs to provide concrete implementations for the methods defined in `gnmi.proto`, like `Capabilities`, `Get`, and `Subscribe`. Finally, it must have access to some data (even if it's just mock data) that it can serve to clients based on the paths they request.

#### Core Components

Our server implementation has two main parts.

First, we need a struct that will hold our server's state and implement the `GNMIServiceHandler` interface. For now, it can be empty.

```go
type gnmiServer struct{
	// In a real server, you'd have a data store, cache, etc.
}
```

Second, we need a `main` function to wire everything up. This function creates an instance of our server, registers it as a Connect RPC handler, and starts an HTTP server.

```go
func main() {
	server := &gnmiServer{}
	mux := http.NewServeMux()
	path, handler := gnmiv1connect.NewGNMIServiceHandler(server)
	mux.Handle(path, handler)

	log.Println("Starting gNMI server on :8080...")
	// Use h2c to support gRPC clients that don't use TLS
	err := http.ListenAndServe(
		"localhost:8080",
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		log.Fatalf("listen and serve failed: %v", err)
	}
}
```

#### Program Skeleton

Putting that together, a skeleton of our server looks like this. The RPC methods are defined, but they just return an "unimplemented" error. This is a great starting point to make sure the server runs and is listening correctly.

{{% render-code file="go/server-skeleton/main.go" language="go" %}}

#### Complete Implementation

Now, we'll fill in the RPC methods with our mock logic. `Capabilities` will return a static list of features, `Get` will return a one-off value for CPU, and `Subscribe` will start a ticker to stream memory usage data.

{{< details-md summary="server/main.go" github_file="go/server/main.go" >}}
{{% render-code file="go/server/main.go" language="go" %}}
{{< /details-md >}}

Now run this server. You can test it with `gNMIc`:

```bash
# Test Capabilities
gnmic -a localhost:8080 --insecure capabilities

# Test Get
gnmic -a localhost:8080 --insecure get --path "/system/cpu/utilization"

# Test Subscribe
gnmic -a localhost:8080 --insecure subscribe --path "/system/memory/state/used" --stream-mode stream --stream-sample-interval 2s
```

---

### Step 3: Building the gNMI Client

With the server running, let's build a client to interact with it.

#### Client Responsibilities

The client's job is simpler. It needs to connect to the server's address and create a gNMI client stub using the code generated by Connect. From there, it can call RPCs by building request messages, sending them to the server, and processing the responses.

#### Core Components

First, we create a new client from the generated `gnmiv1connect` package, pointing it at our server's address.

```go
client := gnmiv1connect.NewGNMIServiceClient(
    http.DefaultClient,
    "http://localhost:8080",
)
```

Calling a unary RPC like `Get` is straightforward. We create a request object, wrap it in a `connect.Request`, and pass it to the client method.

```go
getReq := &gnmiv1.GetRequest{
    Path: []*gnmiv1.Path{getPath}, // where getPath is a *gnmiv1.Path
    Encoding: gnmiv1.Encoding_JSON_IETF,
}
getResp, err := client.Get(ctx, connect.NewRequest(getReq))
```

Calling a streaming RPC like `Subscribe` involves getting a stream object from the client, sending an initial request on it, and then entering a loop to receive responses.

```go
stream := client.Subscribe(ctx)
stream.Send(subReq) // where subReq is a *gnmiv1.SubscribeRequest

for stream.Receive() {
    resp := stream.Msg()
    // ... process response ...
}
```

#### Program Skeleton

A skeleton for the client would establish a connection and prepare to make calls, but might only log that the calls were made without processing the results.

{{% render-code file="go/client-skeleton/main.go" language="go" %}}

#### Complete Implementation

Finally, here is the complete client code that builds the `Get` and `Subscribe` requests and processes their responses.

{{< details-md summary="client/main.go" github_file="go/client/main.go" >}}
{{% render-code file="go/client/main.go" language="go" %}}
{{< /details-md >}}

---

### Caveats and Work Left To Do

This implementation is a starting point. A production-grade gNMI server would need to connect to a real data source, properly parse and validate gNMI paths, and adhere to a schema with actual OpenConfig models. It would also need much more robust error handling for all RPCs, an implementation of the `Set` RPC for configuration changes, and security features like TLS and authentication.

### Wrap Up

We've successfully built a basic gNMI server and client using Go and ConnectRPC. We saw how Connect simplifies setting up a server that's compatible with standard gRPC tools like `gNMIc`. Even with this simple implementation, the core concepts of gNMI's RPCs, especially the bidirectional `Subscribe` stream, are much clearer.

### References

- [gNMI Specification (GitHub)](https://github.com/openconfig/gnmi/blob/master/proto/gnmi/gnmi.proto)
- [OpenConfig](https://www.openconfig.net/)
- [ConnectRPC Documentation](https://connectrpc.com/docs/go/)
- [`gNMIc` Client](https://gnmic.openconfig.net/)
