---
categories: ["article"]
tags: ["grpc", "protobuf", "api", "rpc", "webdev", "humor", "http2", "http3"]
date: "2024-07-02"
description: "gRPC isn't perfect but who is?"
cover: "cover.jpg"
images: ["/posts/grpc-the-good-parts/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC: The Good Parts"
slug: "grpc-the-good-parts"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-the-good-parts
---

## Performance
Is the protobuf encoding the most performant serialization ever? No, of course not. Is it *way* more efficient to JSON or XML?

## Strongly Typed Contracts

## Streaming Support

## Cross-Language Support

## Pioneered HTTP/2
gRPC co-evolved with HTTP/2. There's evidence of that around today, with the Go

## Adoption Options
### JSON/HTTP Transcoding
gRPC now has several more ways to slowly transition into gRPC. If you're replacing a REST based platform it's now possible to define the entire service in protobuf but also expose a REST-like interface using [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway), [Google's transcoding service](https://cloud.google.com/endpoints/docs/grpc/transcoding) or other tools that use the same annotations. The idea is that your backend can be fully powered by gRPC while also exposing a REST-like API when you don't have the ability or desire to update the clients to gRPC.

### gRPC-Web
Now it's possible to use gRPC without HTTP/2!

### ConnectRPC
Automatically get a JSON/HTTP API