---
categories: ["article"]
tags: ["http", "http2", "http3", "web", "webdev"]
date: "2024-08-06T10:00:00Z"
description: ""
cover: "cover.png"
images: ["/posts/yall-are-sleeping-on-http3/cover.png"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Y'all are sleeping on HTTP/3"
slug: "yall-are-sleeping-on-http3"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/yall-are-sleeping-on-http3/
draft: true
---

## Hear me out
I've been talking about HTTP/3 recently because I woke up one day and discovered HTTP/3 was widely supported and already used by over 30% of the web. Why did no one tell me about this? This is an incredibly large adoption rate and we should be talking about it more. It's no longer "one day it would be nice to use HTTP/3". It's now "Okay, so most major cloud providers support it and every single major browser supports it so I guess we're already using it? Interesting." I feel like the rollout of HTTP/3 has been so well done that many programmers and tech enthusiasts are simply unaware of it.

What is crazy to me is that despite the massive success of the rollout of HTTP/3, every time I mention HTTP/3 there's always someone who pops up who's completely unaware that it even exists or thinks that it's some minor change. HTTP/3 is a giant change. HTTP/3 abandons TCP in favor of a channel-aware UDP-based protocol called QUIC. To me, ***this feels important!*** This feels like people need to be talking about it, doing more experiments around QUIC, and writing more tooling and benchmarks.

* Introduce HTTP/3 as a significant advancement, but one that's happening behind the scenes
* Tease the benefits and the technical underpinnings

## What's wrong with TCP?
* **The Limitations of TCP:** Explain how TCP's congestion control and head-of-line blocking cause issues for modern web traffic.
* **Rise of Mobile:** Highlight how mobile devices and their frequent network switching expose TCP's weaknesses.
* **Enter QUIC:** Explain how QUIC, with its UDP foundation, solves these problems.

## Enter: QUIC and HTTP/3
* **Faster Connection Establishment:**  How QUIC reduces the handshakes needed for secure connections.
* **Zero Round Trip Time (0-RTT) Resumption:** Explain how returning visitors to a website can get nearly instant connections.
* **Multiplexing:** How multiple streams of data can be sent over a single connection without blocking each other.
* **Improved Congestion Control:** QUIC's more responsive congestion control leads to faster recovery from packet loss.
* **Enhanced Security:** Built-in TLS 1.3 encryption by default.

### "But I heard UDP is unreliable"
* **Overcoming TCP's Limitations:** Reiterate how UDP's flexibility allows QUIC to bypass TCP's problems.
* **Misconceptions About UDP:** Address concerns about UDP's unreliability and how QUIC implements its own reliability mechanisms.

## Let's see how far we've come
* **Current State of Adoption:** Provide statistics on websites and browsers that support HTTP/3.
* **Major Players:** Highlight big companies like Google and Facebook leading the way.
* **Future Outlook:** Discuss predictions for continued growth and eventual dominance of HTTP/3.

## Challanges Ahead
There are two main areas to focus on with QUIC: adding more tooling and language support for the protocol.

### Tooling/Language Support
Even though browsers and load balancers have good support for QUIC, most programming languages don't support HTTP/3 because QUIC presents a vastly different way to communicate. Without kernel support, adding QUIC to a language is a bit like re-implementing TCP in the language. TCP is usually relatively easy to add because OS kernels typically implement TCP for you and provide bindings. As far as I know, that isn't the case for QUIC.

### Will QUIC stay in userspace?
From what I've seen, there's no well-supported kernel module for QUIC. There exist some projects that do [add a kernel module for Linux](https://github.com/lxin/quic) but it doesn't seem like it's heavily used yet.

## The Future is QUIC
HTTP has a 
* Summarize the main benefits and why QUIC and HTTP/3 are game-changers for the web.
* Encourage readers to check if their browser and favorite sites support HTTP/3.

https://lwn.net/Articles/965134/
https://www.fastly.com/blog/measuring-quic-vs-tcp-computational-efficiency/
https://github.com/lxin/quic
https://www.chromium.org/quic/
https://codilime.com/blog/http3-protocol/
https://http3-explained.haxx.se/en
https://caniuse.com/http3

