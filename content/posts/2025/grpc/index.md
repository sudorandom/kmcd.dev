---
categories: ["article"]
tags: ["grpc", "protobuf", "connectrpc", "buf", "fauxrpc"]
date: "2025-06-12T10:00:00Z"
description: "Why standard gRPC hurts in the browser, and how the modern stack (Connect, Buf, FauxRPC) fixes the developer experience."
cover: "cover.jpg"
images: ["/posts/modern-grpc/cover.jpg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Modern gRPC: Connect, Buf, and the End of Fragility"
slug: "modern-grpc"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/modern-grpc/
draft: true
---

{{< toc >}}

## **I. The gRPC Paradox: Great Tech, Terrible UX**

If you have used standard gRPC in anger, you know the cycle:

1. **The Honeymoon:** You define a Protobuf schema. It feels clean. You dream of type-safe nirvana.
2. **The Reality:** You try to call it from a browser. You realize you need `grpc-web` and an Envoy proxy sidecar just to make a `POST` request.
3. **The Debugging:** You try to `curl` your endpoint. You can't. You download distinct CLI tools (`grpcurl`) just to see if your server is alive.
4. **The Ops:** Your load balancer doesn't understand HTTP/2 frames.

gRPC is a masterpiece of engineering, but for a long time, its Developer Experience (DX) felt like it was designed strictly for server-to-server communication inside a Google datacenter.

The industry has evolved. We have moved beyond "raw" gRPC. We are entering the era of **Schema-Driven Development** powered by tools that actually like developers.

---

## **II. ConnectRPC: gRPC for the Rest of Us**

The biggest friction point in gRPC has always been the transport. Standard gRPC requires HTTP/2 trailers and specific framing that browsers (and many corporate firewalls) hate.

**ConnectRPC** (by the creators of Buf) changes the mental model.

It allows a single server to speak three protocols simultaneously on the same port:
1. **gRPC:** For your backend microservices.
2. **gRPC-Web:** For legacy browser clients.
3. **Connect:** A simple, HTTP/1.1 friendly protocol that supports JSON.

### **A. The "cURL-ability" Factor**

With standard gRPC, you cannot use Postman or cURL without significant friction. With Connect, your gRPC handler is just a POST endpoint handling JSON.

**The Schema:**

```protobuf
service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply);
}
```

**The Call (Standard HTTP):**

```bash
curl \
    --header "Content-Type: application/json" \
    --data '{"name": "Jane"}' \
    [https://api.example.com/greet.v1.Greeter/SayHello](https://api.example.com/greet.v1.Greeter/SayHello)
```

**The Response:**

```json
{"message": "Hello, Jane!"}
```

No binary framing. No proxy. No `grpcurl`.

### **B. Why this matters**
This unifies your infrastructure. You don't need a "Public REST API" and a "Private gRPC API" anymore. You write one Connect handler.
- **Frontend** uses the generated Connect client (lightweight, typesafe).
- **Backend** uses the gRPC protocol (high perf).
- **Scripts/Debug** use standard HTTP/JSON.

---

## **III. Governance First: The Power of `buf breaking`**

Before we write code, we must agree on the contract.

In the old days, you ran `protoc` manually. If you renamed a field from `user_id` to `id`, `protoc` would happily compile it. You would push to production, and instantly break every mobile client running the old version of the app.

Enter **Buf**.

Buf isn't just a compiler; it is a linter and a breaking change detector.

### **A. The Safety Net**

The command `buf breaking` compares your current schema against your git `main` branch (or a remote registry). It understands binary compatibility rules better than you do.

```bash
# You try to change a field type from int32 to string
$ buf breaking --against ".git#branch=main"

req.proto:5:1:Field "1" on message "GetUserRequest" changed type from "int32" to "string".
```

It catches:
- Reused field numbers.
- Changed types on existing fields.
- Renamed packages.
- Deleted fields (if configured to strict mode).

This turns API governance from a "hope and pray" manual review process into a CI/CD failure. **If `buf breaking` passes, you are safe to deploy.**

---

## **IV. FauxRPC: Mocking Without the Mockery**

Frontend developers often sit idle waiting for the Backend team to implement the API.

In the REST world, you might hardcode some JSON in a file.
In the Schema world, we can do better.

**FauxRPC** allows you to generate a fully functioning mock server directly from your Protobuf definitions.

```d2
direction: right
Proto Files -> FauxRPC: Load Schema
FauxRPC -> Mock Server: Spawns HTTP/2 & HTTP/1.1
Client -> Mock Server: Call GetUser(id: 123)
Mock Server -> Client: Returns randomized, valid Protobuf data
```

Because Protobuf is strongly typed, FauxRPC knows exactly what the data *should* look like.

- If the field is `int32`, it returns a number.
- If the field is `repeated string`, it returns an array of strings.
- If you use `protovalidate` (e.g., `string email = 1 [(buf.validate.field).string.email = true]`), FauxRPC can even generate valid email addresses.

### **A. The Workflow**

1. Define `api.proto`.
2. Run `fauxrpc run api.proto`.
3. Frontend team builds the UI against `localhost:8080`.
4. Backend team implements the real logic.
5. Swap the URL.

This decouples the teams entirely. The schema is the only dependency.

---

## **V. Conclusion: It’s About the Contract**

The argument is no longer "REST vs. gRPC."

The argument is **"Loose Contracts vs. Strict Contracts."**

If you choose Strict Contracts (Protobuf), you used to pay a heavy tax in complexity.
- **Buf** removed the complexity of file management and breaking changes.
- **ConnectRPC** removed the complexity of the network transport and browser incompatibility.
- **FauxRPC** removed the complexity of prototyping.

You can now have the rigorous safety of gRPC with the ease of use of REST/JSON. There is no longer a reason to settle for less.

Just remember to check your trailers (or, if you use Connect, just check your HTTP status codes).
