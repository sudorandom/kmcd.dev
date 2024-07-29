---
categories: ["article"]
tags: ["networking", "http", "go", "golang", "tutorial"]
series: ["HTTP from Scratch"]
date: "2024-08-06"
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
- Recap the key points from the previous article about HTTP/0.9.
- Introduce HTTP/1.0 as the next major iteration of the protocol, released in 1996.
- Briefly highlight the key improvements in HTTP/1.0:
    - Headers for metadata.
    - Multiple HTTP methods (POST, HEAD, etc.).
    - Status codes for error handling and response information.
    - Content negotiation.
- State the goal of the article: to implement a simple server that can communicate using HTTP/1.0.

## Understanding HTTP/1.0

### Request Structure
- Explain the multi-line format of HTTP/1.0 requests:
    - Request line (method, path, protocol version).
    - Headers (key-value pairs for metadata).
    - Empty line to separate headers from the body (optional for some methods).
    - Request body (optional).
- Provide an example of a full HTTP/1.0 request.

### Response Structure
- Explain the multi-line format of HTTP/1.0 responses:
    - Status line (protocol version, status code, reason phrase).
    - Headers (key-value pairs for metadata).
    - Empty line to separate headers from the body.
    - Response body.
- Provide an example of a full HTTP/1.0 response.

### Headers
- Discuss the importance of headers in conveying metadata about requests and responses.
- Explain some common headers (Content-Type, Content-Length, User-Agent, etc.).

### HTTP Methods
- Explain the role of different HTTP methods (GET, POST, HEAD, etc.).
- Provide brief examples of how each method is used.

### Status Codes
- Discuss the importance of status codes in providing feedback to the client.
- Explain the different categories of status codes (1xx Informational, 2xx Success, 3xx Redirection, 4xx Client Error, 5xx Server Error).
- Provide examples of common status codes (200 OK, 404 Not Found, 500 Internal Server Error, etc.).

## Implementing an HTTP/1.0 Server
- Discuss the modifications needed in the server code to support HTTP/1.0:
    - Parsing and interpreting headers in the request.
    - Handling optional request bodies.
    - Generating status lines and headers in the response.
    - We're NOT going to maintain backward compatibility with HTTP/0.9 because the semantics of returning a response just isn't compatible and it can be dangerous to support it alongside later versions.
- Provide code snippets for the updated server implementation.

## Testing the Implementation
- Demonstrate how to run the server.
- Show examples of successful requests and responses using different HTTP methods and status codes.
- Explain how to test error scenarios and content negotiation.

## Beyond HTTP/1.0
- Briefly mention the subsequent versions of HTTP (1.1, 2, and 3) and the improvements they introduced.
- Tease the next article in the series, which will cover the implementation of HTTP/1.1.

## Conclusion
- Summarize the key takeaways from the article.
- Emphasize the added flexibility and capabilities of HTTP/1.0 compared to HTTP/0.9.
- Encourage readers to experiment with the code and explore the protocol further.
