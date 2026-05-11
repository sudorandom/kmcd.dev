---
categories: ["article", "project"]
tags: ["dataviz", "internet-map", "networking", "bgp", "rust", "go", "grpc", "protobuf", "education"]
date: "2026-05-12T10:00:00Z"
description: "How a live BGP map evolved into an interactive explainer on internet routing."
cover: "cover.svg"
images: ["/posts/bgp-kmcd-dev/cover.svg"]
featuredalt: ""
featuredpath: "date"
title: "Let's Learn About BGP"
slug: "bgp-kmcd-dev"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/bgp-kmcd-dev/
---

Before we get into the weeds of how this was built, go check out [bgp.kmcd.dev](https://bgp.kmcd.dev) right now. Play around with the interactive elements:

[{{< image src="logo.png" width="500px" class="center" >}}](https://bgp.kmcd.dev)

* **Learn about BGP through interactive diagrams**
* **Try the RPKI safety test.** You might not like what you find about your own ISP's routing security.
* **Explore the interactive dashboard** at [bgp.kmcd.dev](https://bgp.kmcd.dev) to see the heartbeat of the internet's interconnected networks.

Once you have a feel for it, come back here. Or don't. I'm not your dad or anything.

### Why BGP Matters

BGP (Border Gateway Protocol) is essentially the "glue" that holds the internet together. It's the protocol that determines how data travels from one network to another across the globe. When you click a link, BGP is what decided the path those packets took to get to you.

It's also incredibly fragile. A single misconfiguration or a malicious "route leak" can accidentally divert traffic for an entire country or knock major services offline. Despite being the backbone of the global internet, much of it still relies on trust. This is why security measures like RPKI (Resource Public Key Infrastructure) are so critical yet inconsistently adopted.

## The Power of Interactive Explainers

I am really happy with how this project turned out. I have always found "interactive explainer" microsites to be super effective for learning complex technical concepts. Reading a whitepaper about the Border Gateway Protocol (BGP) is one thing, but actively playing with CIDR subnetting tools or triggering the steps in a BGP state machine makes the concepts actually stick.

Static documentation often fails to convey the *dynamic* nature of protocols. An interactive explainer unlocks a "feedback loop" that docs can't: you change a variable, and you see the consequence immediately. It bridges the gap between abstract theory and practical intuition. I find these sites are worth building whenever a concept involves state transitions, complex spatial relationships, or high-stakes edge cases that are hard to replicate in a lab.

### Embracing the Evolution

This whole thing started because I wanted to learn more about BGP, so I wrote [a visually cool (and mostly useless) 24/7 live stream](/posts/live-internet-map/). The natural next step was to leverage some of the insights I observed into a dashboard. But as I built the early version of the dashboard, the explanatory text became more interesting and powerful than the raw dashboard data.

What began as a simple monitoring visualization shifted into a massive interactive learning resource. This taught me a valuable lesson: **projects don't need to start useful.** The most valuable outcome isn't always the original goal. It's often the observations you make while building toward it. By staying flexible, I was able to reshape the project into something far more impactful than just another "live map."

This shift aligns with how I tend to learn best: by doing. I've always found that I don't truly understand a protocol until I've had to handle its edge cases in code. This is why I write about [HTTP from Scratch](/series/http-from-scratch/), [gRPC From Scratch](/series/grpc-from-scratch/) and [gRPC Over HTTP/3](/posts/grpc-over-http3/); to push myself to build one layer deeper than I strictly need for my day-to-day work. But there is another layer to it: I strongly believe that to properly **learn** something, you must be able to **teach** it, or at the very least, communicate it clearly to others. There is a strange shift that happens in my brain when I approach a topic with the intent to present it. It forces a level of rigor that I might otherwise skip. It's the same reason I am such a huge fan of self-reviews while a PR is in draft; looking at my own code through the lens of an external reviewer often reveals "perfect" code to be anything but. It's also 95% of the reason that I write this blog (the other 5% is vanity).

### Interactive Tools

I replaced static images with **interactive SVG diagrams** driven by the same data models used in the backend. You can interactively see different behaviors of BGP and the internet play out from advertisements, to withdrawals, to route leaks.

The most useful tool on the site is the **ISP RPKI Safety Test**. It lets you check if your own Internet provider is using RPKI to sign and validate routes. 

When I first ran this, I was shocked to see that **my own home ISP fails this check.** This means they are effectively trusting the "word" of any other network on the planet without cryptographically verifying it. It's a sobering reminder that the backbone of our digital lives is often held together by conventions and good faith rather than hard security. If your ISP fails, it's a great excuse to reach out to their support and ask *why*. This test is powered by [isbgpsafeyet.com](https://isbgpsafeyet.com/), and they encourage you to tweet about your ISP if they fail.

{{< rpki-check >}}

## How It Was Made

Getting a live global heartbeat of the internet to run smoothly required completely rethinking the architecture. The [original implementation](/posts/live-internet-map/) handled data collection and GPU rendering in a single Go process. It worked fine at first, but garbage collector pauses during high-volume routing bursts (30,000+ updates per second) caused dropped frames in the 24/7 live stream.

The new architecture separates these concerns into two distinct outputs: a **real-time 4K live stream** on YouTube and an **interactive microsite** at [bgp.kmcd.dev](https://bgp.kmcd.dev).

Here is the high-level flow: raw BGP updates from global sensors come in, a Rust backend processes and validates them in real-time. This processed data is then broadcast to a Go client (which renders the 60 FPS YouTube stream) and a Go indexer (which generates the static data for the microsite).

Here is a look at the architecture that solved this:

{{< d2 width="100%" max-height="140vh" >}}
direction: down

classes: {
  rust_box: {
    style: { fill: "#f4e4d4"; stroke: "#a72145"; stroke-width: 2 }
  }
  go_box: {
    style: { fill: "#e0f7fa"; stroke: "#00acc1"; stroke-width: 2 }
  }
}

Sources: {
  RIPE: RIPE RIS-Live
  RV: RouteViews
  Kafka: Kafka
  RV -> Kafka
}

Collector: Rust Backend {
  class: rust_box
  BGPKit: BGPKit Parser
  Classifier: Classification Engine
  RPKI: RPKI Validator
}

Viewer: Go Frontend {
  class: go_box
  Ebitengine: Ebitengine
}

Indexer: Go Indexer {
  class: go_box
  Snapshots: Hourly Snapshots
}

GitHub: GitHub Repository
Cloudflare: Cloudflare Pages

OBS: OBS Studio {
  Encoder: RTMP Output
}

Sources -> Collector: Raw Telemetry
Collector -> Viewer: "gRPC Stream"
Collector -> Indexer: "Summary Data"
Indexer -> GitHub: "Commit Snapshots"
GitHub -> Cloudflare: "CI/CD Trigger"
Cloudflare -> "bgp.kmcd.dev": "Serve Static Data"
Viewer -> OBS: "Window Capture\n(4K 60FPS)"
OBS -> YouTube: "4K 60FPS Video"
{{< /d2 >}}

### The Rust Rewrite

I rewrote the telemetry collector in **Rust** using the [BGPKit](https://bgpkit.com/) ecosystem. Offloading the heavy lifting of parsing BMP and RIS-Live streams to a language built for high-throughput, memory-safe concurrency completely solved the performance bottlenecks. 

Could I have just optimized the Go version? Probably. But the sheer volume of small allocations during BGP parsing was a "worst-case scenario" for Go's garbage collector. Moving to Rust allowed me to manage memory exactly where it mattered, ensuring that even the most massive routing bursts wouldn't stutter the visualization. Plus, I had been wanting to dip my toes into Rust, and this proved to be a great project for it.

### Go and Ebitengine for the Live Stream

With Rust handling the data ingestion, the Go viewer was freed up to focus entirely on the **24/7 YouTube live stream**. Using the [Ebitengine](https://ebitengine.org/) game engine, the Go application is now just a lean client that renders a 2D Mollweide projection of the globe at 60 FPS. This output is captured by OBS and pushed to YouTube. That 60 FPS target is nearly always reached now, when it was a pipedream with the first architecture.

Yes, I know how insane I sound when I say "oh, and Go is used for the frontend", but I learned to respect the performance and robustness of Ebitengine for this specific real-time visualization task. This is what personal projects are for: to do things you wouldn't normally do in ways you wouldn't normally do it.

### Unifying on Protobuf

To manage the complex schema between Go, Rust, and TypeScript, I leveraged **Protocol Buffers** and **gRPC**. 

Defining the interface between the Rust collector and the Go viewer in Protobuf simplified the Go code significantly. Instead of managing internal channels, it just subscribes to a gRPC stream of events. I can even restart the Rust collector to update logic without the visualizer dropping a single frame.

### Static Hourly Snapshots

To keep the web platform fast without maintaining a live database to service requests, I built a Go indexer. It generates snapshots of the global routing state every hour and commits them to a GitHub repository. This triggers a build on Cloudflare Pages, which deploys the updated snapshots as static assets. This has proven 'reliable enough' for this project. Because of this, the data referenced in the website are updated hourly.

### Why a Microsite?

I chose to build [bgp.kmcd.dev](https://bgp.kmcd.dev) as a standalone microsite rather than integrating it directly into this Hugo blog for a few key reasons:

- **Freedom of Choice:** Starting fresh gave me complete control over the HTML, CSS, and JavaScript. I wasn't constrained by the blog's existing design or Hugo's template system, allowing me to use the best tools for this specific project.
- **Cohesion:** It makes more sense for the frontend to live in the same repository as the data collection and processing code. Since they are part of the same system, they can evolve together without being tied to the blog's codebase.
- **Deployment:** By keeping it separate, the microsite has its own build and deployment pipeline. It can be updated or refactored independently, which is much cleaner than jamming dynamic data features into a static blog.

## The Result

This project shifted from a monolithic live map to a distributed educational tool. Choosing specialized tools for each layer (Rust for throughput, Go for rendering, and Protobuf for data delivery) made the system more stable and capable.

I set out to visualize the internet, but ended up understanding it. I even built something that might help others do the same. If there's one takeaway here, it's that "learning by building" involves more than the code you write. It's also about the clarity you gain when you try to explain that code to the rest of the world.

Explore the tools and live data at [bgp.kmcd.dev](https://bgp.kmcd.dev). Source code is on [GitHub](https://github.com/sudorandom/livemap.kmcd.dev/).
