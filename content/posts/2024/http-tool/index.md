---
categories: ["article"]
tags: ["http", "http2", "http3"]
date: "2024-07-23T10:00:00Z"
description: "Find out if your browser using the latest and greatest."
cover: "cover.png"
images: ["/posts/http-tool/cover.png"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "What version of HTTP are you using?"
slug: "http-tool"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/http-tool/
mastodonID: "112835242941123154"
---

[Click here to see what version of HTTP your browser is using to load this website](/http/).

## HTTP/3
HTTP/3 is the newest version of the protocol for powering the web. It offers the same features as HTTP/1.1 and HTTP/2 but has a drastically different foundation: using a protocol called QUIC that is built on UDP instead of using the good-ol TCP protocol.

Recently I've been talking [about HTTP/3 and how it might be good to use it with gRPC](/posts/grpc-over-http3/). That's a good idea, but based on many of the comments that I saw in response to that post I feel like many people don't know how much work has been done for HTTP/3 already. The HTTP/3 spec was published by the IETF and has since been adopted by *so many companies*. Here's just a few:

- Cloudflare
- Google Cloud CDN and Load Balancer
- AWS CloudFront
- Akamai CDN
- Azure CDN
- CDN77
- Fastly CDN
- Azure Application Gateway
- nginx
- Caddy
- HAProxy
- and tons more

So as you can see, the server side of HTTP/3 is getting more and more ready for prime time. And the browser support is also pretty much there. You can see on [caniuse.com](https://caniuse.com/http3) that HTTP/3 is supported on all major browsers including all of the familiar names: Chrome, Firefox, Safari, and Edge.

So because web browsers and many web load balancer services already have support for HTTP/3 it shouldn't be surprising to know that [30% of websites now support HTTP/3](https://w3techs.com/technologies/details/ce-http3). This is an impressive figure, given how different the foundations of HTTP/3 are compared to HTTP/2 and HTTP/1.1.

As a reminder, here are the benefits of HTTP/3:
- **Faster connection establishment**, which is especially useful with slower, unstable or congested connections.
- Completely avoids the **head-of-line blocking** problem, allowing browsers to use a single connection to load everything at full speed.

## The tool
So, are you curious to know if your browser is leveraging the latest web technology? I built a simple tool that will reveal the HTTP version you're using to access this website.  It'll even let you know if you're enjoying the full benefits of HTTP/3's speed and efficiency. Give it a try and see how your browser stacks up!

### How It Is Made
This is a static blog, which obviously makes it a little harder to make a tool like this. In addition to that, the protocol used to connect from the load balancer isn't necessarily the protocol used by clients. In my case, I use Cloudflare as a host and load balancer, so I configured the endpoint to add a new HTTP header that contains the HTTP version used between the client and load balancer. It might look like:

```http
x-kmcd-http-request-version: HTTP/3
```

### Javascript/Browser Limitations
So, I have this new header being returned, but I need to do something to display the value. One thing that's a little crazy to me is that Javascript on a page *cannot read the headers that were returned for the current page.* So... When you load the page, Javascript runs and calls the [Fetch API](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API) to retrieve the page again. The response object that is returned *does* contain the full list of headers, which is what I used to determine the output. By the way, you might ask "doesn't the response object also contain the version of HTTP used?" and the answer is, unfortunately, no. It seems like it would be simple to add but it's just not there.

## Check it out
So go check out the tool. If you don't get `HTTP/3` at first you may have to refresh a few times. There are [a few different ways](https://http3-explained.haxx.se/en/h3/h3-altsvc) to hint to browsers that HTTP/3 is supported and I use all of them... but many browsers will avoid attempting HTTP/3 for the first few requests to a website until it trusts that HTTP/3 is available. So go on and try it and let me know if it's useful at all:

[Click here to go to the tool.](/http/)
