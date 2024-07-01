---
categories: ["article", "tutorial"]
tags: ["connectrpc", "grpc", "tutorial", "golang", "rest"]
date: "2024-07-16"
description: ""
cover: "cover.jpg"
images: ["/posts/transitioning-from-rest-to-grpc/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Transitioning from REST to gRPC"
slug: "transitioning-from-rest-to-grpc"
type: "posts"
devtoPublished: false
devtoSkip: true
canonical_url: https://kmcd.dev/posts/transitioning-from-rest-to-grpc
draft: true
---

I've talked a lot [about ConnectRPC](/posts/connectrpc) and how I think it presents a developer experience that is a step above gRPC. However, I don't think I did a good job of highlighting how ConnectRPC (and gRPC) can slowly adopted. That is what I will tackle today.

Let's start off with a REST API.


Define REST API
Define gRPC with REST API Annotations
Show Go Code that does the translations