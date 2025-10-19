---
categories: ["article"]
tags: ["api", "web", "http"]
date: "2025-09-22T10:00:00Z"
description: "A gentle introduction to the world of web APIs."
cover: "cover.jpg"
images: ["/posts/what-is-a-web-api/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "What is a web API?"
slug: "what-is-a-web-api"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/what-is-a-web-api/
draft: true
---

You've probably heard the term "API" thrown around a lot, especially if you're in the tech world. It's one of those acronyms that gets used everywhere, but what does it actually mean? And more specifically, what is a *web* API? In this post, we're going to demystify these terms and give you a solid understanding of what web APIs are and how they power the internet as we know it.

## What is the Internet?

Before we can talk about web APIs, we need to understand the foundation they're built on. The internet is a global network of computers. That's it. It's a massive, interconnected web of machines that can talk to each other. Think of it like a giant postal system, but for digital information.

## What is the Web?

The World Wide Web, or simply "the web," is a service that runs on the internet. It's a system of interlinked hypertext documents accessed via the internet. When you open your browser and go to a website, you're using the web. The web is what makes the internet user-friendly for most people. It's the collection of websites, videos, images, and other content that you interact with daily.

## What is an API?

API stands for **Application Programming Interface**. That might sound complicated, but the concept is actually quite simple. An API is a set of rules and tools that allows different software applications to communicate with each other.

Imagine you're at a restaurant. You, the customer, are like one application, and the kitchen is another application. You can't just walk into the kitchen and start making your own food. Instead, you have a waiter who acts as an intermediary. You give your order to the waiter, the waiter takes the order to the kitchen, the kitchen prepares the food, and the waiter brings it back to you.

In this analogy, the waiter is the API. They provide a clear and defined way for you to interact with the kitchen without needing to know all the complex details of how the kitchen works. You just need to know how to place an order (the API's "rules").

## What is a Web API?

A web API is an API that is accessed over the web using the HTTP protocol. This is the same protocol that your web browser uses to fetch websites. This means that web APIs are designed to be used by a wide range of clients, including web browsers, mobile apps, and other servers.

Web APIs are the backbone of the modern internet. They're what allow different services to talk to each other and share data. When you use an app on your phone to check the weather, that app is likely using a web API to get the latest weather data from a weather service. When you log in to a website using your Google or Facebook account, that's a web API at work.

## What is HTTP?

HTTP stands for **Hypertext Transfer Protocol**. It's the protocol that powers the web. It's a request-response protocol, which means that a client (like your web browser) sends a request to a server, and the server sends back a response.

An HTTP request and response are made up of a few key components:

### HTTP Verbs (Methods)

The HTTP verb, or method, tells the server what action the client wants to perform. The most common verbs are:

*   **`GET`**: Retrieve data from the server. This is what your browser does when you visit a website.
*   **`POST`**: Send data to the server to create a new resource. For example, when you submit a form on a website.
*   **`PUT`**: Update an existing resource on the server.
*   **`PATCH`**: Partially update an existing resource on the server.
*   **`DELETE`**: Delete a resource from the server.

Here's an example of a `GET` request using `curl`:

```bash
curl -X GET "https://api.github.com/users/kmcd"
```

### Status Codes

The status code is a three-digit number that the server sends back in its response. It tells the client whether the request was successful, and if not, what went wrong. Some common status codes include:

*   **`200 OK`**: The request was successful.
*   **`201 Created`**: The request was successful and a new resource was created.
*   **`400 Bad Request`**: The server couldn't understand the request.
*   **`401 Unauthorized`**: The client is not authorized to access the resource.
*   **`404 Not Found`**: The server couldn't find the requested resource.
*   **`500 Internal Server Error`**: Something went wrong on the server.

You can see the status code in the response from `curl` by adding the `-i` flag:

```bash
curl -i "https://api.github.com/users/kmcd"
```

```http
HTTP/2 200
server: GitHub.com
content-type: application/json; charset=utf-8
...
```

### Headers

Headers are key-value pairs that are sent along with the request and response. They provide additional information about the request or response. For example, the `Content-Type` header tells the client what kind of data is in the response body (e.g., `application/json`, `text/html`).

Here's an example of a request with a custom header:

```bash
curl -H "X-Custom-Header: My-Header-Value" "https://api.github.com/users/kmcd"
```

### Paths

The path identifies the specific resource that the client wants to access on the server. For example, in the URL `https://api.github.com/users/kmcd`, the path is `/users/kmcd`.

### Query Parameters

Query parameters are used to filter, sort, or paginate the results from an API. They are added to the end of the URL after a `?` and are separated by `&`. For example, in the URL `https://api.github.com/users/kmcd/repos?sort=created&direction=desc`, the query parameters are `sort=created` and `direction=desc`.

```bash
curl "https://api.github.com/users/kmcd/repos?sort=created&direction=desc"
```

This request would get all the repositories for the user `kmcd`, sorted by the created date in descending order.

## Conclusion

Web APIs are the unsung heroes of the internet. They work behind the scenes to allow the apps and services you use every day to communicate with each other and share data. By understanding the basics of web APIs and the HTTP protocol they're built on, you've taken a big step towards understanding how the modern web works.

In future posts, we'll dive deeper into the world of web APIs, exploring topics like REST, GraphQL, and how to build your own web APIs. Stay tuned!