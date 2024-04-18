---
categories: ["article"]
tags: ["networking", "grpc", "http", "go", "golang", "tutorial", "protobuf", "connectrpc"]
date: "2024-12-01"
description: "We've made the world's simplest gRPC client and server for unary RPCs. Now let's tackle ~streaming~."
cover: "cover.jpg"
images: ["/posts/grpc-from-scratch-part-3/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC From Scratch: Part 3 - Protobuf Encoding"
slug: "grpc-from-scratch-part-3"
type: "posts"
devtoSkip: true
canonical_url: https://sudorandom.dev/posts/grpc-from-scratch-part-3
draft: true
---

> This is part three of a series. [Click here to see gRPC From Scratch: Part 1 where I build a simple gRPC client](/posts/grpc-from-scratch/) and [gRPC From Scratch: Part 2 where I build a simple gRPC server.](/posts/grpc-from-scratch-part-2/)

In the last two parts, I showed how to make an extremely simple client and server that... kind-of works. But I punted on a topic last time that is pretty important: I used generated protobuf types and the Go protobuf library to do all of the heavy lifting of protobufs for me. That ends today. I'll start by using using the [`protowire`](https://pkg.go.dev/google.golang.org/protobuf/encoding/protowire) library directly, which is a bit closer to what is actually happening on the wire. They include a fun disclaimer:

> For marshaling and unmarshaling entire protobuf messages, use the google.golang.org/protobuf/proto package instead.

Am I going to listen to this solid advice? No! I want to know how this works! No reflection and no reliance on generated code. So let's get started:

## protowire
Protowire provides

### strings

### integers

### array of integers
packed values
