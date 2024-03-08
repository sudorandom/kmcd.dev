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

### Limited Options, Creative Solutions (or Problems)

REST only offers a handful of methods (GET, POST, etc.) to handle data. This can feel limiting when you want more specific control. For instance, imagine liking a specific photo in a social media feed. A POST request (meant for creating new resources) might seem like an odd fit for this action. Developers sometimes resort to workarounds, like including an "action" parameter set to "like" within a POST request. While this can function, it can make the API less intuitive and harder to understand for those unfamiliar with these conventions.

Let's stop pretending that there are only 6 things you can do to a resource.

### JSON? More like "Just Slow, No?"

Take a social media app with a large user base. Sending a user's entire profile information, including their lengthy bio and high-resolution profile picture, in JSON with every request can be wasteful. Protobuf, on the other hand, is a compact and efficient binary format specifically designed for data serialization. Even factoring in GZIP, JSON clearly loses out to encoded protobuf [in pretty much every way](https://auth0.com/blog/beating-json-performance-with-protobuf/): CPU usage, memory usage, message size and speed.

### OpenAPI? More like "Open... to Interpretation."

Imagine a real-time multiplayer game API. OpenAPI shines for static data like player stats but struggles with the constant updates of a game world (player movement, item interactions).  Defining every possible scenario with OpenAPI schemas gets messy.  Alternatives like WebSockets, built for real-time communication, might be better for keeping players in sync with the ever-changing game world.

Here are 5 examples of why REST with OpenAPI might fall short:

1. **Real-time Data Streams (e.g., Stock Ticker, Live Chat):** REST relies on request-response cycles, making it clunky for continuously flowing data. OpenAPI struggles to define the ever-changing structure of a live feed, leading to cumbersome schema updates and inefficient data transfer. 
2. **Fine-grained Data Updates (e.g., Social Media Likes, Game State Changes):**  REST methods often operate on entire resources. Imagine liking a specific post comment; deleting the whole post with DELETE is inefficient. OpenAPI schemas struggle to represent these partial updates, forcing developers into workarounds like custom flags in requests.
3. **Complex Data Structures (e.g., Shopping Cart, User Profile):**  Modeling intricate entities as single REST objects can be messy. A shopping cart with items, discounts, and customer information becomes a bloated object. OpenAPI schemas can become complex and challenging to maintain for these scenarios.
4. **Unforeseen Events and Interactions (e.g., Multiplayer Games, Collaborative Workspaces):**  OpenAPI excels at defining expected data structures, but struggles with the unpredictable. Imagine a multiplayer game with emergent player behavior. Defining schemas for every possible action becomes impractical. 
5. **Limited Error Handling:**  REST error codes might not effectively convey the nuances of issues. Imagine network errors during a live stream - a single generic error code might not be helpful for troubleshooting. OpenAPI cannot define custom error types with specific details for these situations.

While OpenAPI serves a valuable purpose for many APIs, its limitations become apparent when dealing with dynamic data streams. This is where alternatives like WebSockets, designed for real-time communication, might be a better fit.

**So, what are the alternatives? Buckle up, because we're venturing beyond REST:**

### GraphQL

The "Choose Your Own Adventure" API. Still with the social media app example, imagine fetching only the user's name and profile picture for the feed view, and then requesting their full profile and friend list separately when a user clicks on their profile. GraphQL allows you to specify exactly the data you need for each view, reducing unnecessary data transfer.

### gRPC

The "Cut the Drama" API. Consider a mobile game that communicates with a game server. gRPC allows you to define remote procedures (like `attackEnemy` or `usePowerUp`) that the client can call directly on the server. This removes the need for complex REST resource mapping and makes the communication intent clear.

**Looking beyond these two options, there's a whole world of possibilities:**

### WebSockets

Need real-time updates like a live chat or stock ticker? WebSockets offer a persistent two-way communication channel, ideal for constantly flowing data between client and server. This is different from REST's request-response cycle, allowing for a more dynamic connection.

### Server-Sent Events (SSE)

SSE allows the server to push updates to the client without the client needing to constantly ask. Imagine live sports scores or social media notifications. SSE is simpler to implement than WebSockets, but is one-way (server to client).

Here's a conclusion that summarizes the key points and offers a final thought:

**The Verdict: REST vs. the Rest**

REST APIs have served us faithfully for years, but as our applications become more intricate and data-hungry, it's worth considering the alternatives. GraphQL offers flexibility in data fetching, gRPC provides clear and efficient communication, WebSockets enable real-time data flow, and SSE simplifies server-to-client updates. 

The choice ultimately depends on your specific needs. But remember, the API landscape is ever-evolving. So, keep an open mind, explore the options, and don't be afraid to break free from the comfy (but maybe slightly threadbare) jeans of REST when a more fitting alternative emerges. 
