+++
categories = ["opinion"]
tags = ["rest", "api", "rpc", "grpc", "http"]
date = "2024-03-26"
description = "Web APIs are the backbone of the modern web, but the ever-evolving landscape demands a rethink. This article explores alternatives to the traditional REST approach, diving into solutions like GraphQL, gRPC, and WebSockets. Unlock the full potential of your APIs and discover a world beyond REST!"
cover = "cover.jpg"
images = ["/posts/restless/cover.jpg"]
featured = ""
featuredalt = ""
featuredpath = "date"
linktitle = ""
title = "RESTless: Web APIs After REST"
slug = "restless"
type = "posts"
+++

The "RESTful API" has been the workhorse of the web for many years. It has been an ever-changing religion with tenants that developers try their hardest to adhere to. But as web applications evolve, user demands grow and our industry experience with API design grows, it's time to re-evaluate this approach. This article explores the limitations of REST and delves into modern alternatives that can unlock a world of possibilities beyond.

### Objects? More like "objnoxious"

Imagine building a social media API endpoint to retrieve a user's feed. Using a single REST object to represent a feed item can get messy. This object would need to encompass:

* User information (username, profile picture)
* Post content (text, images, videos)
* User interactions (likes, comments, shares) with timestamps
* Additional data like post visibility or author verification status

This "feed" object becomes bloated, especially if the feed contains many posts. Fetching and updating this complex object for every feed interaction can be inefficient. You can split the object up into many child objects but you are likely creating the need for clients to make more requests and greatly increasing the complexity of the API.

This highlights a limitation of REST: forcing real-world entities (like a social media feed) into rigid object structures with a strict hierarchy can lead to cumbersome data management.

### Versioning is weird

Let's say you introduce a new field to your product data model in a REST API. Versioning in the path (e.g., `/api/v2/products`) forces you to update every single endpoint URL that uses that data. Versioning each path by adding a query parameter (e.g., `/api/products?version=2`) or header seems more targeted, but what if only a specific endpoint needs the new version? Do you maintain a list of versions per endpoint? Do you bump the version for the entire API and default to the latest version? This is open to interpretation and every solution seems awkward to me.

### Limited Options, Clever Solution

REST only offers a handful of methods (GET, POST, etc.) to handle data. There are **9** methods in total. Let's talk about each one.

- CONNECT
- DELETE
- GET
- HEAD
- OPTIONS
- PATCH
- POST
- PUT
- TRACE

Most of these are used in niche, hyper-specific ways. I suspect most developers don't know what `HEAD`, `TRACE`, `OPTIONS`, and `CONNECT` do. None of these are incredibly useful when developing web APIs. So let's ditch them. Let's also ditch `PATCH` because it's just `PUT` in a trenchcoat.

Now we have: `GET`, `POST`, `DELETE`, and `PUT`. Sweet, we're left with enough methods to make a CRUD application. Roughly speaking:

- **C**reate = `POST`
- **R**ead = `GET`
- **U**pdate = `PUT`
- **D**elete = `DELETE`

Wow! Such simple. So elegant. I'm sure glad that everything web developers do boils down to these 4 simple actions... Oh wait, that's **_totally_ wrong**. There are so many more actions you can do to an object. Just think of how crazy it would be if you were using a programming language where you could only have four pre-defined methods in each class. Think of all of the extra container classes and weird abstractions we'd make on top of that. This sounds like utter insanity and is exactly what REST gives us.

Let's stop pretending that there are only 9 (but actually 4) things you can do to a resource.

### Inefficiency of JSON

JSON is slow and inefficient. It'd be insane if we built the internet around this format. Wait, we did? Really?  JSON is wasteful in many ways. It's text-based, which has an inherent cost in payload size and processing. JSON also will include key names over and over again and the length of the keys directly translates to longer payloads. This is not ideal if we're trying to save on data transfer. Protobuf, on the other hand, is a compact and efficient binary format specifically designed for data serialization. Even factoring in gzip, JSON loses out to encoded protobuf [in pretty much every way](https://auth0.com/blog/beating-json-performance-with-protobuf/): CPU usage, memory usage, message size and speed. This just shows that there are better formats than JSON.

### OpenAPI?

[OpenAPI](https://www.openapis.org/) is a specification for describing RESTful APIs. It acts as a contract between API providers and consumers, defining the available resources, their properties, and the allowed operations (GET, POST, PUT, DELETE) for each. A common way OpenAPI is used is by generating OpenAPI specifications from the source of the backend service. Depending on the level of library integration these tools can automatically discover the HTTP method, route, request and response types, etc. This can help keep documentation up-to-date compared to manually creating OpenAPI spec, which is pretty incredible.

Okay, now here's where I get philosophical. If we're going to have a declarative specification for our APIs, I believe the specification should be the source of truth rather than the output. In my mind, OpenAPI specification should be the very first thing you write and agree upon and the servers and clients should be. However, many people don't do this because the tooling isn't amazing. Developers have grown fond of specific libraries and frameworks to develop our APIs so the target for generating language/framework/library code is vast and appears to be an incredibly hard problem. OpenAPI tries to solve so many problems at once. It's coming in after we've designed our APIs and is trying to describe what's already there. It's an afterthought.

I've attempted to go down the route of using OpenAPI to generate server stubs and clients. It didn't end well. There were too many issues generating clients and servers, even when the OpenAPI specification was valid. So I had to edit our OpenAPI spec to fit the limitations of the code generators just to get code generation to work. And that was just the beginning of my problems. I contend that this is a natural result of the design goals of the project. OpenAPI was not designed for generating code this way as a priority, so with many complex scenarios, it can be unclear how to map the spec to the semantics of the language/framework/library. OpenAPI wasn't designed for that, it was only designed to describe the API, not the server or client that produces or consumes it. Now consider just how many "targets" for client and server stubs you want. Now consider that target libraries and the OpenAPI spec itself are evolving. This results in a compatibility matrix from hell.

## So what other options do we have?
REST _has_ served us well, but the modern web demands more. Let's explore the exciting alternatives that offer a range of benefits and functionalities:

### GraphQL

The "Choose Your Own Adventure" API. Still with the social media app example, imagine fetching only the user's name and profile picture for the feed view, and then requesting their full profile and friend list separately when a user clicks on their profile. GraphQL allows you to specify exactly the data you need for each view, reducing unnecessary data transfer.

This method also has its downsides like making the backend API extremely complex.

### gRPC

The "Cut the Drama" API. Consider a mobile game that communicates with a game server. gRPC allows you to define remote procedures (like `attackEnemy` or `usePowerUp`) that the client can call directly on the server. This removes the need for complex REST resource mapping and makes the communication intent clear. There are variants of gRPC like [ConnectRPC](/posts/connectrpc/) that allow for leveraging of HTTP GET requests so you can fully leverage browser caching.

### WebSockets

Need real-time updates like a live chat or stock ticker? WebSockets offer a persistent two-way communication channel, ideal for constantly flowing data between client and server. This is different from REST's request-response cycle, allowing for a more dynamic connection.

### Server-Sent Events (SSE)

SSE allows the server to push updates to the client without the client needing to constantly ask. Imagine live sports scores or social media notifications. SSE is simpler to implement than WebSockets, but is one-way (server to client).

Here's a conclusion that summarizes the key points and offers a final thought:

**The Verdict: REST vs. the Rest**

REST APIs have served us faithfully for years, but as our applications become more complex and data-hungry, it's worth considering the alternatives. GraphQL offers flexibility in data fetching, gRPC provides clear and efficient communication, WebSockets enable bidirectional real-time data flow and great browser support, and SSE simplifies server-to-client updates.

The choice ultimately depends on your specific needs. But remember, the API landscape is ever-evolving. HTTP/2 and HTTP/3 have opened up some new functionality that we have yet to fully tap into with our API designs. So, keep an open mind, explore the options, and don't be afraid to break free from the comfy (but maybe slightly threadbare) jeans of REST when a more fitting alternative emerges. 
