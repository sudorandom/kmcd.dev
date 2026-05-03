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

Before we get into the weeds of how this was built, go check out [bgp.kmcd.dev](https://bgp.kmcd.dev) right now. Play around with the interactive elements, test your ISP's routing security, and watch the live map for a minute.

Once you have a feel for it, come back here. Or don't. I'm not your boss or anything.

## The Power of Interactive Explainers

I am really happy with how this project turned out. I have always found "interactive explainer" microsites to be super effective for learning complex technical concepts. Reading a whitepaper about the Border Gateway Protocol (BGP) is one thing, but actively playing with CIDR subnetting tools or triggering the steps in a BGP state machine makes the concepts actually stick.

This whole thing started because I wanted to learn more about BGP, so I wrote [a visually cool (and mostly useless) live map](/posts/live-internet-map/). The natural next step was to leverage some of the insights I observed into a dashboard. But as I built the early version of the dashboard, the explanatory text became more interesting and powerful than the raw dashboard data.

What began as a simple monitoring visualization shifted into a massive interactive learning resource.

### Interactive Tools

I replaced static images with **interactive SVG diagrams** driven by the same data models used in the backend. You can interactively see different behaviors of BGP and the internet play out from advertisements, to withdrawals, to route leaks.

The most useful tool on the site is the **ISP RPKI Safety Test**. It lets you check if your own Internet provider is using RPKI to sign and validate routes. Sadly, my own home ISP fails this check, which proves to me that internet security still has a ways to go.

{{< rpki-check >}}

## How It Was Made

Getting a live global heartbeat of the internet to run smoothly required completely rethinking the architecture. The [original implementation](/posts/live-internet-map/) handled data collection and GPU rendering in a single Go process. It worked fine at first, but garbage collector pauses during high-volume routing bursts (30,000+ updates per second) caused dropped frames in the visualization.

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

I rewrote the telemetry collector in **Rust** using the [BGPKit](https://bgpkit.com/) ecosystem. Offloading the heavy lifting of parsing BMP and RIS-Live streams to a language built for high-throughput, memory-safe concurrency completely solved the performance bottlenecks. I had been wanting to dip my toes into Rust, and this proved to be a great project for it.

### Go and Ebitengine

With Rust handling the data ingestion, the Go viewer was freed up. Using the [Ebitengine](https://ebitengine.org/) game engine, the Go application is now just a lean client that focuses entirely on rendering a 2D Mollweide projection of the globe at 60 FPS. That 60 FPS target is nearly always reached now, when it was a pipedream with the first architecture.

Yes, I know how instane I sound when I say "oh, and Go is used for the frontend", but I learned to respect the performance and robustness of Ebitengine. This is what personal projects are for: to do thinks you wouldn't normally do in ways you wouldn't normally do it.

### Protobuf as the Glue

To manage the complex schema between Go, Rust, and TypeScript, I leveraged **Protocol Buffers** and **gRPC**. 

Defining the interface between the Rust collector and the Go viewer in Protobuf simplified the Go code significantly. Instead of managing internal channels, it just subscribes to a gRPC stream of events. I can even restart the Rust collector to update logic without the visualizer dropping a single frame.

More importantly, dropping JSON for Protobuf on the web frontend resulted in **10x smaller file sizes**. There isn't a ton of data displayed on the site anymore, but it is cool to see this amount of payload size reduction, even with relatively simple data.

### Static Hourly Snapshots

To keep the web platform fast without maintaining a live database for every user request, I built a Go indexer. It generates snapshots of the global routing state every hour and commits them to a GitHub repository. This triggers a build on Cloudflare Pages, which deploys the updated snapshots as static assets. This has proven 'reliable enough' for this project.

### Why a Microsite?

I chose to build [bgp.kmcd.dev](https://bgp.kmcd.dev) as a standalone microsite rather than integrating it directly into this Hugo blog for a few key reasons:

- **Freedom of Choice:** Starting fresh gave me complete control over the HTML, CSS, and JavaScript. I wasn't constrained by the blog's existing design or Hugo's template system, allowing me to use the best tools for this specific project.
- **Cohesion:** It makes more sense for the frontend to live in the same repository as the data collection and processing code. Since they are part of the same system, they can evolve together without being tied to the blog's codebase.
- **Deployment:** By keeping it separate, the microsite has its own build and deployment pipeline. It can be updated or refactored independently, which is much cleaner than jamming dynamic data features into a static blog.

## The Result

This project shifted from a monolithic live map to a distributed educational tool. Choosing specialized tools for each layer (Rust for throughput, Go for rendering, and Protobuf for data delivery) made the system more stable and capable.

Explore the tools and live data at [bgp.kmcd.dev](https://bgp.kmcd.dev). Source code is on [GitHub](https://github.com/sudorandom/livemap.kmcd.dev/).