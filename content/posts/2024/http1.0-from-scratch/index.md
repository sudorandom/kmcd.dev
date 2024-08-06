---
categories: ["article"]
tags: ["networking", "http", "go", "golang", "tutorial"]
series: ["HTTP from Scratch"]
date: "2024-08-13"
description: "The final shape of the web forms."
cover: "cover.jpg"
images: ["/posts/http1.0-from-scratch/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "HTTP/1.0 From Scratch"
slug: "http1.0-from-scratch"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/http1.0-from-scratch/
draft: true
---

## Introduction
In our previous exploration, we delved into the simplicity of [HTTP/0.9](/posts/http0.9-from-scratch), a protocol that served as the web's initial backbone. However, as the internet evolved, so did its needs. Enter HTTP/1.0, a landmark version released in 1996 that laid the groundwork for the web we know today.  

HTTP/1.0 was a game-changer, introducing features that revolutionized web communication:

- **Headers:** Metadata that added context and control to requests and responses.
- **HTTP Methods:** A diverse set of actions (GET, POST, HEAD, etc.) beyond simple retrieval.
- **Status Codes:** Clear signals about the outcome of requests, paving the way for error handling.
- **Content Negotiation:** The ability to request specific formats or languages for content.

In this article, we'll journey through the intricacies of HTTP/1.0 and craft a simple Go server that speaks this influential protocol.

## Understanding HTTP/1.0

### Request Structure

HTTP/1.0 requests follow a structured format:

1. **Request Line:** Specifies the HTTP method (e.g., GET, POST), the requested path, and the protocol version (HTTP/1.0).
2. **Headers:** Key-value pairs that provide additional information (e.g., `User-Agent`, `Content-Type`, `Referer`).
3. **Empty Line:**  Signals the end of the headers.
4. **Request Body (Optional):** Data sent with the request (common with POST).

#### Example
```http
GET /index.html HTTP/1.0
User-Agent: Mozilla/5.0
Host: www.example.com

(Optional request body)
```

#### Response Structure
HTTP/1.0 responses mirror this structure:

1. **Status Line:**  Includes the protocol version, a status code (e.g., 200 OK, 404 Not Found), and a reason phrase.
2. **Headers:** Similar to request headers, providing metadata about the response.
3. **Empty Line:** Separates headers from the body.
4. **Response Body:** The actual content being sent back to the client.

#### Example

```http
HTTP/1.0 200 OK
Content-Type: text/html
Content-Length: 1354

(HTML content here)
```

### Headers
Headers act as messengers, conveying vital information about requests and responses. Some common headers include:

- `Content-Type`:  Indicates the format of the data (text/html, image/jpeg, etc.).
- `Content-Length`: Specifies the size of the response body.
- `User-Agent`: Identifies the client software making the request.

### HTTP Methods
HTTP/1.0 introduced a variety of methods:

- **GET:**  Requests a resource.
- **POST:**  Submits data to be processed by the server.
- **HEAD:**  Similar to GET, but only requests the headers, not the body.

### Status Codes
Status codes are essential for communication between the client and server. They fall into categories:
- **1xx:** Informational.
- **2xx:** Success.
- **3xx:** Redirection.
- **4xx:** Client Error (e.g., 404 Not Found).
- **5xx:** Server Error (e.g., 500 Internal Server Error).

## Implementing an HTTP/1.0 Server in Go
TODO: Implement request bodies...

TODO: Describe each section of the server here

## Testing the Implementation
We'll guide you through testing your server using tools like `curl` or a web browser, showcasing how to interact with different methods, status codes, and headers. We'll also explore how to simulate error scenarios to ensure your server handles them gracefully.

## Beyond HTTP/1.0
While HTTP/1.0 was a significant leap forward, the story doesn't end there. HTTP/1.1, HTTP/2, and HTTP/3 brought further enhancements. In our next article, we'll dive into the world of HTTP/1.1, exploring its advancements over HTTP/1.0; reusable connections, the host header to help with making one server handle multiple websites and TLS finally enters the scene.

## Conclusion
HTTP/1.0 marked a pivotal moment in the evolution of the web. By understanding its core principles and building a simple server, we gain valuable insights into the foundations of modern web communication. As you experiment and explore, remember that this is just the beginning – the web's journey is ongoing!

