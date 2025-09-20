---
categories: ["article"]
tags: ["interviewing", "hiring", "networking"]
date: "2025-09-22T10:00:00Z"
description: "Why I love asking candidates to explain how the internet works."
cover: "cover.jpeg"
images: ["/posts/favorite-interview-question/cover.jpeg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "My Favorite Interview Question"
slug: "favorite-interview-question"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/favorite-interview-question/
draft: false
---

Interviews are a strange dance. As an interviewer, you're trying to get a signal on a candidate's skills, experience, and personality in a very short amount of time. As a candidate, you're trying to showcase your best self while under pressure. It's a tough situation for everyone involved.

Over the years, I've asked a lot of interview questions. Some have been great, some have been duds. But I always come back to my absolute favorite question:

{{< bigtext >}}"How does the internet work?"{{< /bigtext >}}

I know what you're thinking. That's a huge, open-ended question! And you're right, it is. That's exactly why I love it. You also might be thinking that this is a super common question. I agree! But unlike most questions, this one is still extremely useful despite its popularity.

## Why this question is so powerful

This question is a fantastic tool for a few reasons:

- **It's a blank canvas.** The candidate can start anywhere they want. Do they start with the physical layer? The application layer? The philosophy of interconnectedness? It gives them a chance to show me what they think is important.
- **It reveals their depth of knowledge.** The question can be answered at many different levels of abstraction. A junior engineer might give a high-level overview, while a senior engineer might be able to dive into the nitty-gritty details of TCP handshakes and BGP routing.
- **It tests communication skills.** Can the candidate take a complex topic and explain it in a clear, concise way? This is a crucial skill for any engineer, especially in a team environment.
- **It's a conversation starter.** The question often leads to a natural back-and-forth conversation. I can ask follow-up questions to probe deeper into areas where the candidate seems knowledgeable or to help them along if they're struggling.

Sometimes, the sheer open-endedness of the question can be a bit overwhelming for a candidate. If they're struggling to start or getting too philosophical, I'll give them a more concrete prompt: 

{{< bigtext >}}"What happens when you type google.com into your browser and press Enter?"{{< /bigtext >}}

This usually gets the ball rolling. Here's a breakdown of the kind of answer I'm looking for, with varying levels of detail depending on the candidate's seniority.

### What I expect from most candidates

For any software engineering role that involves web development, I expect the candidate to be able to walk me through the following steps:

- **DNS Lookup:** The browser needs to translate the human-readable domain name `google.com` into a machine-readable IP address. This involves checking the browser's cache, the OS's cache, and then querying a DNS resolver, which has its own upstream sources and cache.
- **TCP Connection:** Once the browser has the IP address, it establishes a TCP connection with the server. This involves establishing a connection using TCP's three-way handshake (SYN, SYN-ACK, ACK).
- **HTTP Request/Response:** The browser sends an HTTP `GET` request to the server. The server processes the request and sends back an HTTP response, which includes the HTML, CSS, and JavaScript files that make up the Google homepage.
- **Rendering:** The browser parses the HTML and renders the page. It also executes the JavaScript, which might make additional requests to the server to fetch more data.

### What I hope for from senior candidates

For a more senior role, especially at a company that deals with networking or infrastructure, I'm hoping for a deeper dive into some of the following topics:

- **ARP (Address Resolution Protocol):** Before the browser can even send a packet to the router, it needs to know the router's MAC address. This is where ARP comes in.
- **DHCP (Dynamic Host Configuration Protocol):** - How did our computer get an IP address in the first place? A senior candidate might explain that the machine likely requested an IP address, a subnet mask, the default gateway's IP, and the DNS server's IP from a DHCP server on the local network.
- **NAT (Network Address Translation):** - To conserve the limited global supply of IPv4 addresses, most home and office networks use a single public IP address for many devices. NAT is the mechanism that allows a router to translate between private, internal IP addresses and that single public one.
- **BGP (Border Gateway Protocol):** How does my request find its way from my local network to Google's servers, potentially halfway across the world? Mentioning BGP shows an understanding that the internet is a 'network of networks' and that routing between these large autonomous systems is a complex, solved problem.
- **HTTP/2 and HTTP/3 (QUIC):** Is the connection really just a single TCP connection? What about multiplexing and head-of-line blocking? A senior candidate might mention these newer protocols and their advantages.
- **TCP Keep-Alives:** How does the connection stay open for subsequent requests? What are the trade-offs of keep-alives?
- **Caching:** Where does caching happen in this whole process? DNS caching, browser caching, CDN caching... there are many layers.
- **Pre-flight Requests (CORS):** If the page makes requests to other domains, the browser might need to make a pre-flight `OPTIONS` request to check if the cross-origin request is allowed.
- **Packet Breakdown:** What does an actual packet look like? What are the different headers (Ethernet, IP, TCP, HTTP)?

## It's not about getting it "right"

I want to be clear: I don't expect any candidate to know everything about the internet. That's impossible. What I'm looking for is a candidate who has a solid foundation, can reason about complex systems, and is curious to learn more. How does the candidate behave when I press into an area that they don't know a lot about? Are they curious? Are they willing to say they don't know? Do they make things up? All of that is very useful for evaluating skill and knowledge level.

Sometimes a candidate's answer goes in a completely unexpected direction. I once had a candidate describe the hardware interrupts and OS context switches that happen just from typing 'google.com'. Even though it was a tangent, it was a valuable signal that showed me how they reason about systems from the hardware up.

This question is a fantastic way to gauge all of those things. It's a journey we go on together, and I often learn something new from the candidate's answer.

What's your go-to interview question? I'd love to hear it.
