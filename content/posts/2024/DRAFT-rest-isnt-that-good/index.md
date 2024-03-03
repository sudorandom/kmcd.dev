+++
categories = ["opinion"]
tags = ["rest", "api", "rpc", "grpc", "http"]
date = "2024-03-10"
description = "Hot take about REST"
cover = "cover.jpg"
images = ["/posts/rest-isnt-that-good/cover.jpg"]
featured = ""
featuredalt = ""
featuredpath = "date"
linktitle = ""
title = "Rest Isn't That Good"
slug = "rest-isnt-that-good"
type = "posts"
draft = true
+++

Hot take coming in... hot. Rest isn't that great.

Should this be a PUT or POST? Let's argue about it for an hour before looking it up.
It's hard to come up with objects that match what you want to do
Versioning is a wild west
 - Should you put the version in the path? At the beginning (version the entire API) or the end (version a single endpoint?)
Usually, REST implies JSON nowadays.
 - JSON sucks
   - compare against protobuf memory
   - compare against protobuf speed
   - Just let me use a trailing comma
OpenAPI and jsonschema doesn't cover complicated use-cases like streaming
Errors are the wild west

Alternatives:
- GraphQL
  - How complicated is your data model that you need a DSL to decide what data to return? Just write some specific endpoints for specific use-cases.
- RPC
  - We're just calling code on a remote server. Let's stop with the philosophical nonsense around resources and let's stop pretending there are only 6 verbs (realistically people only use 2 HTTP verbs but whatever)
  - Contract-based gRPC and variants
  - Obviously superior
  - Can still be used with JSON: ConnectRPC
