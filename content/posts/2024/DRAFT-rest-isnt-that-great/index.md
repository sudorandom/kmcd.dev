+++
categories = ["opinion"]
tags = ["rest", "api", "rpc", "grpc", "http"]
date = "2024-03-10"
description = "Hot take about REST"
cover = "cover.jpg"
images = ["/posts/rest-isnt-that-great/cover.jpg"]
featured = ""
featuredalt = ""
featuredpath = "date"
linktitle = ""
title = "REST Isn't That Great"
slug = "rest-isnt-that-great"
type = "posts"
draft = true
+++

**Alright, gather 'round developers, for I bring a scorching hot take:** REST APIs, the ubiquitous workhorse of the web, are overrated. Don't get me wrong, they've served us well, but like that comfy pair of jeans with holes in the knees, they might be due for an upgrade.

**Let's unpack this unpopular opinion with some real-world pain points:**

### Objects? More like "objnoxious."

Imagine building a social media API endpoint to retrieve a user's feed. Using a single REST object to represent a feed item can get messy. This object would need to encompass:

* User information (username, profile picture)
* Post content (text, images, videos)
* User interactions (likes, comments, shares) with timestamps
* Additional data like post visibility or author verification status

This becomes a bloated object, especially if the feed contains many posts. Fetching and updating this complex object for every feed interaction can be inefficient. You can split the object up into many child objects but you are likely creating the need for clients to make more requests.

This highlights a limitation of REST: forcing real-world entities (like a social media feed) into rigid object structures can lead to cumbersome data management.

### Versioning is weird

Let's say you introduce a new field to your product data model in a REST API. Versioning in the path (e.g., `/api/v2/products`) forces you to update every single endpoint URL that uses that data. Versioning at the end (e.g., `/api/products?version=2`) seems more targeted, but what if only a specific endpoint needs the new version? Do you maintain a list of versions per endpoint? Do you assume bump the version for the entire API and default to the latest version? This is open to interpretation and every solution seems awkward.

### Limited Options, Clever Solutions

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

Most of these are used in niche, hyper-specific ways. I suspect most developers don't know what `HEAD`, `TRACE`, `OPTIONS`, and `CONNECT` do. None of these are incredibly useful when developing web APIs. So let's ditch them. Let's also ditch PATCH because it's just `PUT` in a trenchcoat.

Now we have: `GET`, `POST`, `DELETE`, and `PUT`. Sweet, we're left with enough methods to make a CRUD application. Roughly speaking:

- Create = `POST`
- Retreive = `GET`
- Update = `PUT`
- Delete = `DELETE`

Wow! Such simple. So elegant. I'm sure glad that everything web developers do boils down to these 4 simple actions... Oh wait, that's **totally wrong**. There are so many more actions you can do. Just think of how crazy it would be if you were using a programming language where you could only have four pre-defined methods in each class. Think of all of the extra container classes and weird abstractions we'd make on top of that. This sounds like utter insanity and is exactly what REST gives us.

Let's stop pretending that there are only 9 (but actually 4) things you can do to a resource.

### JSON is also bad

Take a social media app with a large user base. Sending a user's entire profile information, including their lengthy bio and high-resolution profile picture, in JSON with every request can be wasteful. Protobuf, on the other hand, is a compact and efficient binary format specifically designed for data serialization. Even factoring in GZIP, JSON loses out to encoded protobuf [in pretty much every way](https://auth0.com/blog/beating-json-performance-with-protobuf/): CPU usage, memory usage, message size and speed. This just shows that there are better formats than JSON.

### OpenAPI? More like "Open... to Interpretation."

Imagine a real-time multiplayer game API. OpenAPI shines for static data like player stats but struggles with the constant updates of a game world (player movement, item interactions). Defining every possible scenario with OpenAPI schemas gets messy. Alternatives like WebSockets, built for real-time communication, might be better for keeping players in sync with the ever-changing game world.

Here are 5 examples of why REST with OpenAPI might fall short:

1. **Real-time Data Streams (e.g., Stock Ticker, Live Chat):** REST relies on request-response cycles, making it clunky for continuously flowing data. OpenAPI struggles to define the ever-changing structure of a live feed, leading to cumbersome schema updates and inefficient data transfer. 
2. **Fine-grained Data Updates (e.g., Social Media Likes, Game State Changes):** REST methods often operate on entire resources. Imagine liking a specific post comment; deleting the whole post with DELETE is inefficient. OpenAPI schemas struggle to represent these partial updates, forcing developers into workarounds like custom flags in requests or having many, many different kinds of nested objects.
3. **Complex Data Structures (e.g., Shopping Cart, User Profile):** Modeling intricate entities as single REST objects can be messy. A shopping cart with items, discounts, and customer information becomes a bloated object. OpenAPI schemas can become complex and challenging to maintain for these scenarios.
4. **Unforeseen Events and Interactions (e.g., Multiplayer Games, Collaborative Workspaces):** OpenAPI excels at defining expected data structures, but struggles with the unpredictable. Imagine a multiplayer game with emergent player behavior. Defining schemas for every possible action becomes impractical.
5. **Out-of-date specs:** Many OpenAPI specs end up being partially or completely hand-written. The best way to avoid this is to generate servers and clients from OpenAPI specs so that you use it as a single source of truth for the API but generators for many languages are... bad and OpenAPI doesn't have clear semantics around how these types map to each programming language.

While OpenAPI serves a valuable purpose for many APIs, its limitations become apparent when dealing with dynamic data streams. This is where alternatives like WebSockets, designed for real-time communication, might be a better fit.

**So, what are the alternatives? Buckle up, because we're venturing beyond REST:**

### GraphQL

The "Choose Your Own Adventure" API. Still with the social media app example, imagine fetching only the user's name and profile picture for the feed view, and then requesting their full profile and friend list separately when a user clicks on their profile. GraphQL allows you to specify exactly the data you need for each view, reducing unnecessary data transfer.

This method also has its downsides like making the backend API extremely complex.

### gRPC

The "Cut the Drama" API. Consider a mobile game that communicates with a game server. gRPC allows you to define remote procedures (like `attackEnemy` or `usePowerUp`) that the client can call directly on the server. This removes the need for complex REST resource mapping and makes the communication intent clear.

### WebSockets

Need real-time updates like a live chat or stock ticker? WebSockets offer a persistent two-way communication channel, ideal for constantly flowing data between client and server. This is different from REST's request-response cycle, allowing for a more dynamic connection.

### Server-Sent Events (SSE)

SSE allows the server to push updates to the client without the client needing to constantly ask. Imagine live sports scores or social media notifications. SSE is simpler to implement than WebSockets, but is one-way (server to client).

Here's a conclusion that summarizes the key points and offers a final thought:

**The Verdict: REST vs. the Rest**

REST APIs have served us faithfully for years, but as our applications become more intricate and data-hungry, it's worth considering the alternatives. GraphQL offers flexibility in data fetching, gRPC provides clear and efficient communication, WebSockets enable real-time data flow, and SSE simplifies server-to-client updates. 

The choice ultimately depends on your specific needs. But remember, the API landscape is ever-evolving. So, keep an open mind, explore the options, and don't be afraid to break free from the comfy (but maybe slightly threadbare) jeans of REST when a more fitting alternative emerges. 
