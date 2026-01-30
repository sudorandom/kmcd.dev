---
categories: ["article"]
tags: ["mitmproxy", "grpc", "networking", "golang", "react"]
date: "2026-01-22T10:00:00Z"
description: "How to build a safer, service-oriented web UI for mitmproxy using mitmproxy-addon-grpc and mitmflow for traffic inspection."
cover: "cover.svg"
images: ["/posts/a-safer-web-interface-for-mitmproxy/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "A Safer, Service-Oriented Web UI for mitmproxy"
slug: "a-safer-web-interface-for-mitmproxy"
type: "posts"
devtoSkip: true
canonical_url: "https://kmcd.dev/posts/a-safer-web-interface-for-mitmproxy/"
draft: true
---

For a while now, I've been using `mitmproxy` to intercept and inspect traffic from various applications, particularly from my mobile devices. My setup involves routing traffic through a WireGuard VPN on my server, which then directs it through `mitmproxy`. This allows me to see what's happening under the hood.

While `mitmproxy` is an incredibly powerful tool, its default web interface, `mitmweb`, has always been a bit of a concern for me. `mitmweb` is designed for interactive use, giving you the ability to not only inspect but also modify and replay traffic, and even configure the `mitmproxy` server itself. These are powerful features, but they also represent a significant security risk if exposed, especially in a more persistent, service-oriented setup like mine. I wanted something "safer" that I could leave running.

My goal was to treat `mitmproxy` less like a command-line tool you run for a specific task and more like a continuous service that passively monitors and logs traffic for later inspection. I didn't need to modify requests on the fly; I just needed a read-only view of the data.

This led me to create two projects:

{{< d2 >}}
direction: down
style: {
  font-size: 14
}

# Actors
traffic_source: "Mobile App" {
  shape: person
}

# Systems
subsystem_mitmproxy: "mitmproxy" {
  style.stroke-dash: 2
  
  mitmproxy_instance: "mitmproxy instance"
  grpc_addon: "gRPC Addon"
}

subsystem_mitmflow: "mitmflow" {
  style.stroke-dash: 2
  
  go_backend: "Go Backend (ConnectRPC)"
  web_client: "React Web Client"
}

# Connections
traffic_source -> subsystem_mitmproxy.mitmproxy_instance: "Proxied Traffic"

subsystem_mitmproxy.grpc_addon -> subsystem_mitmflow.go_backend: "gRPC Stream"

subsystem_mitmflow.go_backend -> subsystem_mitmflow.web_client: "Connect RPC" {
  style.stroke-dash: 2
}

{{< /d2 >}}

### 1. `mitmproxy-addon-grpc`

[**`mitmproxy-addon-grpc`**](https://github.com/sudorandom/mitmproxy-addon-grpc) is an addon for `mitmproxy` that exports traffic data in real-time via a gRPC stream. Instead of saving flows to a file or handling them within Python, this addon pushes HTTP, TCP, UDP, and DNS data to an external gRPC server.

This approach decouples the data collection (`mitmproxy`) from the data storage and presentation. It allows for a more robust and flexible system. You can have a central server that receives flow data from multiple `mitmproxy` instances, processes it, and stores it in any way you see fit. The protobuf definition for this gRPC service can be found [here](https://github.com/sudorandom/mitmproxy-addon-grpc/blob/main/proto/mitmproxygrpc/v1/service.proto).

### 2. `mitmflow`

With the data streaming out over gRPC, I needed a new front end to view it. That's where [**`mitmflow`**](https://github.com/sudorandom/mitmflow) comes in.

`mitmflow` is a web application that connects to the gRPC server (which you have to implement, but a sample is provided) and displays the traffic flows in a clean, real-time interface. It's built with React and Go, and it's designed from the ground up to be a read-only inspector. It doesn't have the ability to modify traffic or configure the `mitmproxy` instance, which makes it much safer to expose on my network.

It provides a simple, searchable view of all the traffic, allowing me to see what my apps are doing without the risk of accidentally interfering with them or exposing powerful tools to others on my network.

### The Result

Together, these two projects create a `mitmproxy` setup that feels more like a permanent service. I can have `mitmproxy` and the gRPC addon running continuously on my server, and from any device on my network, I can pull up the `mitmflow` web interface to see what's going on. It achieves my goal of a "safer" `mitmproxy` web UI and moves it closer to a set-and-forget kind of service.

The source for both is on my GitHub. Check them out if you're looking for a similar setup!
