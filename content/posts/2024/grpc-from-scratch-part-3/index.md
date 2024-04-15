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
title: "gRPC From Scratch: Part 3 - Streaming"
slug: "grpc-from-scratch-part-3"
type: "posts"
devtoSkip: true
canonical_url: https://sudorandom.dev/posts/grpc-from-scratch-part-3
draft: true
---

> This is part three of a series. [Click here to see gRPC From Scratch: Part 1 where I build a simple gRPC client](/posts/grpc-from-scratch/) and [gRPC From Scratch: Part 2 where I build a simple gRPC server.](/posts/grpc-from-scratch-part-2/)

In the last two sections I showed how to make an extremely simple client and server that... kind-of works. Now we're going to tackle ***streaming***. And we're actually going to make it harder than before. Both our streaming client will not rely on protobuf code. We will be using the `protowire` library directly to write our message. Here's a disclaimer on [the documentation for the library](https://pkg.go.dev/google.golang.org/protobuf/encoding/protowire):

> For marshaling and unmarshaling entire protobuf messages, use the google.golang.org/protobuf/proto package instead. 

Am I going to listen to this solid advice? No! I want to know how this works! Using this library is one step closer to what is actually happening on the wire. No reflection and no reliance on generated code.

## Rewriting our unary client/server
Okay, let's revisit the client we made last time.

## Client Streaming

## Server Streaming

## Bidirectional Streaming
