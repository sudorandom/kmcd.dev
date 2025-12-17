---
categories: ["article"]
tags: ["go", "gnmi", "connectrpc", "grpc", "networking", "telemetry"]
date: "2026-01-06T10:00:00Z"
description: "Building a gNMI server from scratch in Go using ConnectRPC."
cover: "cover.svg"
images: ["/posts/gnmi-from-scratch/cover.svg"]
featuredalt: "A diagram showing a client and server communicating via gNMI"
featuredpath: "date"
linktitle: ""
title: "gNMI Server From Scratch"
slug: "gnmi-from-scratch"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/gnmi-from-scratch/
draft: true
---

The gRPC Network Management Interface (gNMI) is a powerful standard for streaming telemetry and configuration management on network devices. While many excellent libraries and agents exist, building a small gNMI server from scratch is a fantastic way to understand the protocol's mechanics deeply.

In this post, we'll do just that. We'll implement a basic gNMI server in Go, using the modern and flexible [ConnectRPC](https://connectrpc.com/) framework instead of native gRPC-go.

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

1. **Define the Test Environment:** Outline the tools we'll use to test our server.
2. **Explore the Protobufs:** Look at the core `gnmi.proto` messages.
3. **Build a Server:** Implement the `gNMI` service using ConnectRPC, serving mock CPU and memory metrics.
4. **Discuss Limitations:** Review what's missing for a production-ready system.

### A Note on gRPC Reflection

gRPC Reflection is a service that allows clients to query a server's gRPC services at runtime. Instead of requiring clients to have a local copy of the `.proto` files, reflection allows them to discover the available services, methods, and message types directly from the server.

This is incredibly useful for ad-hoc testing with tools like `grpcurl` and `buf curl`, as it removes the need to manually specify the location of the Protobuf schema files for every command.

ConnectRPC provides a simple way to add reflection to a server. In our `main` function, we can add the following:

```go
reflector := grpcreflect.NewStaticReflector(gnmiv1connect.GNMIName)
mux.Handle(grpcreflect.NewHandlerV1(reflector))
mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
```

With this, our server will now advertise its services, making it much easier to explore and test.

---

### Our Testing Toolkit

Before we write code, let's define our testing strategy.

We will use a few tools to test our server's behavior. Because ConnectRPC can serve traffic over standard HTTP, we can use `curl` for basic checks. For gRPC-specific interactions, we can use `grpcurl` or `gNMIc`, a versatile gNMI client for testing gNMI compliance.

---

### gNMI and OpenConfig

It's important to understand the relationship between the two. gNMI is the protocol; it defines *how* a client and server exchange data, including the RPCs for `Get`, `Set`, and `Subscribe`.

OpenConfig, on the other hand, provides the data models; it defines *what* the data looks like. It offers a set of vendor-neutral, standardized data models written in the YANG modeling language. These models provide a predictable, tree-like structure for device configuration and operational state.

While you can use gNMI to transport data for any data model, it is most commonly used with OpenConfig. In our example, we will reference OpenConfig-style paths and models to simulate a real-world scenario.

### The gNMI Protobufs

The protocol is defined by `.proto` files, which you can view directly on GitHub in the [openconfig/gnmi](https://github.com/openconfig/gnmi/blob/master/proto/gnmi/gnmi.proto) repository. The core of gNMI is the `gNMI` service, which defines four RPCs: `Capabilities`, `Get`, `Set`, and `Subscribe`.

We'll focus on `Capabilities`, `Get`, and the streaming `Subscribe` RPC.

A few key messages to understand:

- **`Path`**: Represents a path to a data element in a tree-like data model. In gNMI, a path is structured as a sequence of `PathElem` messages, each typically containing a `name` field. For example, the path `/system/cpu/utilization` would be represented in JSON as `{"elem": [{"name": "system"}, {"name": "cpu"}, {"name": "utilization"}]}`.
- **`TypedValue`**: A wrapper for the actual value, which can be a string, int, bool, etc.
- **`Notification`**: A container for an update, containing a timestamp, a path, and a set of updated values.
- **`SubscribeRequest`**: Sent by the client to initiate or modify a subscription. It specifies paths and a subscription mode (`ONCE`, `POLL`, or `STREAM`).
- **`SubscribeResponse`**: Sent by the server to the client. It can contain a `Notification` with data or a synchronization message.



---

### Step 1: Code Generation with `buf`

First, let's set up our project to generate the necessary Go code from the `gnmi.proto` file. While `buf` supports fetching dependencies from the Buf Schema Registry, the `openconfig/gnmi` repository has some complex import paths. A more reliable and self-contained approach is to vendor the required `.proto` files directly into our project.

1.  **Vendor the Protobuf files.** Create a `proto` directory. Inside, we'll place `gnmi.proto` and its dependency `gnmi_ext.proto`. We will also strip the `go_package` option from the files so that we can use `buf`'s `managed` mode to generate idiomatic Go packages.

2.  **Configure `buf` for local generation.** Create a `buf.yaml` file that tells `buf` to use our local `proto` directory as the root for discovery.

    ```yaml
    version: v1
    name: buf.build/kmcd/gnmi-from-scratch
    roots:
      - proto
    ```

3.  **Define the generation output.** Create a `buf.gen.yaml` file. This uses `managed mode` to automatically prefix our generated Go code with our module path, and places the output in a `gnmi/gen` directory.

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

Now, run `buf generate`:

```bash
buf generate
```

This will use our local `proto` files, generate Go code according to the `buf.gen.yaml` configuration, and place the output in the `gnmi/gen` directory.

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

Now run this server.

### A Note on gRPC Reflection

gRPC Reflection is a service that allows clients to query a server's gRPC services at runtime. Instead of requiring clients to have a local copy of the `.proto` files, reflection allows them to discover the available services, methods, and message types directly from the server.

This is incredibly useful for ad-hoc testing with tools like `grpcurl` and `buf curl`, as it removes the need to manually specify the location of the Protobuf schema files for every command.

ConnectRPC provides a simple way to add reflection to a server. In our `main` function, we can add the following:

```go
reflector := grpcreflect.NewStaticReflector(gnmiv1connect.GNMIName)
mux.Handle(grpcreflect.NewHandlerV1(reflector))
mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
```

With this, our server will now advertise its services, making it much easier to explore and test.

---

### Our Testing Toolkit

Before we write code, let's define our testing strategy.

We will use a few tools to test our server's behavior. Because our server speaks standard gRPC and the Connect protocol, and now has reflection enabled, we can use a variety of tools to interact with it.

#### `grpcurl`

`grpcurl` is a command-line tool that lets you interact with gRPC servers. It's like `curl`, but for gRPC. With reflection enabled, we no longer need to specify the `--import-path` and `--proto` flags.

**Capabilities**
```bash
grpcurl -plaintext localhost:8080 gnmi.gNMI/Capabilities
```

**Get**
```bash
grpcurl -plaintext -d '{
    "path": [{
        "elem": [
            {"name": "system"},
            {"name": "cpu"},
            {"name": "utilization"}
        ]
    }]
}' localhost:8080 gnmi.gNMI/Get
```

**Subscribe**
```bash
grpcurl -plaintext -d '{
    "subscribe": {
        "subscription": [{
            "path": {
                "elem": [
                    {"name": "system"},
                    {"name": "memory"},
                    {"name": "state"},
                    {"name": "used"}
                ]
            },
            "mode": "STREAM"
        }]
    }
}' localhost:8080 gnmi.gNMI/Subscribe
```

#### `buf curl`

`buf curl` is part of the `buf` CLI and can be used to send requests to a Connect, gRPC, or gRPC-Web server.

**Capabilities**
```bash
buf curl --protocol grpc \
    -X POST \
    --http2-prior-knowledge \
    http://localhost:8080/gnmi.gNMI/Capabilities
```

**Get**
```bash
buf curl --protocol grpc \
    -d '{
        "path": [{
            "elem": [
                {"name": "system"},
                {"name": "cpu"},
                {"name": "utilization"}
            ]
        }]
    }' \
    -X POST \
    --http2-prior-knowledge \
    http://localhost:8080/gnmi.gNMI/Get
```

**Subscribe**
```bash
buf curl --protocol grpc \
    -d '{
        "subscribe": {
            "subscription": [{
                "path": {
                    "elem": [
                        {"name": "system"},
                        {"name": "memory"},
                        {"name": "state"},
                        {"name": "used"}
                    ]
                },
                "mode": "STREAM"
            }]
        }
    }' \
    -X POST \
    --http2-prior-knowledge \
    http://localhost:8080/gnmi.gNMI/Subscribe
```

#### gNMIc
You can also use a gNMI-specific client like `gNMIc`.
```bash
# Test Capabilities
gnmic -a localhost:8080 --insecure capabilities

# Test Get
gnmic -a localhost:8080 --insecure get --path "/system/cpu/utilization"

# Test Subscribe
gnmic -a localhost:8080 --insecure subscribe --path "/system/memory/state/used" --stream-mode stream --stream-sample-interval 2s
```

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
