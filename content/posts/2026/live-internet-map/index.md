---
categories: ["article", "project"]
tags: ["dataviz", "internet-map", "internet", "networking", "bgp", "map", "world", "infrastructure", "ebitengine", "go"]
date: "2026-03-02T10:00:00Z"
description: "Building a cool looking, real-time BGP map"
cover: "map.png"
images: ["/posts/live-internet-map/map.png"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
featured: true
title: "Live Internet Map"
slug: "live-internet-map"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/live-internet-map/
---

Right now, thousands of routers are arguing about how to reach each other. That’s expected. It’s how the Internet works. This website wouldn't load without it. BGP (Border Gateway Protocol) continuously announces and withdraws prefixes, adjusting how traffic moves globally. Most people see URLs and apps; routers see prefixes and AS paths.

I made a map that lets us listen in on this conversation, but in a relaxing, aesthetically pleasing way.

In my [last post](/posts/internet-map-2026/), I mentioned a [websocket-based streaming API from RIPE](https://ris-live.ripe.net/). At the time, I set it aside. Soon, it became my obsession and the live view was born. While this visualization occasionally stumbles into being practically useful for spotting global outages, my primary requirement was simply to build a really cool looking map.

You can check out the source code for this project on [GitHub](https://github.com/sudorandom/bgp-stream/) or watch the map in action on my [YouTube channel](http://livemap.kmcd.dev) or here:

{{< youtube-live channel="UCA9eO4Gt-Ua6lAEGzWQHQFA" >}}

## What are we looking at?

This map is a live visualization of the Border Gateway Protocol (BGP). This is the "language" routers use to talk to each other and decide the best path for your data to travel across the globe. 

Imagine a router trying to find the best way to send traffic to Google. It receives multiple path advertisements from its neighbors, and it has to pick the most efficient route:

{{< d2 >}}
direction: right

# Styles
classes: {
  bgp_peer: {
    shape: rectangle
    style: {border-radius: 5}
  }
}

# The Observer
Observer: Your Router {
  shape: cylinder
}

# The Destination
Destination: Google\nAS15169\n(8.8.8.0/24) {
  shape: cloud
}

# Path 1: The Best Path
AS8283: ISP A\nAS8283 {class: bgp_peer; style: {stroke: "#2ecc71"; stroke-width: 2}}
Observer -> AS8283: "Best Path >"
AS8283 -> Destination

# Path 2: A longer alternative
AS7018: ISP B\nAS7018 {class: bgp_peer}
AS6939: Transit Provider\nAS6939 {class: bgp_peer}
Observer -> AS7018: "Alternative"
AS7018 -> AS6939
AS6939 -> Destination
{{< /d2 >}}

Every pulse on the map represents a real routing update. Sometimes it’s routine churn. Sometimes it’s maintenance, an outage, or a path change somewhere along the way.

### Spotting a BGP Flap

If you are watching the map and suddenly see a wave of pulses lighting up all over the world at the exact same time, you might be witnessing a BGP flap.

{{< diagram >}}
{{< figure src="flappy-bird.png" width="200px" height="200px" loading="lazy" >}}
{{< /diagram >}}

In networking, flapping happens when a route rapidly appears and disappears. Imagine a misconfigured router or a loose fiber cable. The router yells to the Internet, "I have a path to Google!" only to drop the connection a second later and say, "Never mind, it is gone". That single localized hiccup doesn’t stay local. It ripples outward as routers everywhere recalculate their paths. To keep the whole system from grinding to a halt, modern routers use Route Flap Damping. This essentially puts the noisy network in a time-out until it proves it can stay stable.

### Decoding the Pulses

When you see those colored pulses popping off on the map, they represent specific BGP message types. "Updates" is the general term, but practically, the protocol is juggling a few distinct events:

{{< diagram >}}
{{< figure src="legend.png" alt="Legend" width="400px" loading="lazy" >}}
{{< /diagram >}}

| Event Type | Color | Description |
| :--- | :--- | :--- |
| **Propagation** | Blue | Routers frequently re-announce perfectly valid paths just to keep their tables current; this redundant background noise makes up the blue pulses on the map. |
| **Path Change** | Purple | The destination is still online, but the directions changed. If traffic suddenly has to detour through an extra transit provider to reach its goal, you'll see those routing adjustments flash purple. |
| **Withdrawals** | Red | Red means a route is dead. A router is explicitly telling the Internet that a previously advertised IP block is no longer reachable which is usually the result of a severed fiber cable, hardware failure, or a planned maintenance window. |
| **New Paths** | Green | Bright green pulses mean a new path just opened up. This could be a new ISP coming online, a fresh datacenter spinning up, or just a router discovering a better shortcut. |

{{< diagram >}}
{{< figure src="map-animation-noui.webp" caption="Animation of BGP events in Europe" animate="true" width="700px" animate="true" >}}
{{< /diagram >}}

When you zoom out and see all those colors firing at once, the true scale of the Internet comes to life. It tells the story of over 70,000 independent networks coordinating in real time.

## What else is on the map?

To make sure the map isn't just a wall of moving dots, I included several dashboard elements that provide context to the chaos:

| Preview | Description |
| :--- | :--- |
| {{< figure src="top-activity-hubs.png" alt="Top Activity Hubs" loading="lazy" >}} | Ranks the countries currently experiencing the highest volume of network updates. It is an instant look at where in the world the most "routing churn" is happening at any given moment. |
| {{< figure src="most-active-prefixes.png" alt="Most Active Prefixes" loading="lazy" >}} | Tracks the specific network blocks causing the most noise. Great for spotting outages or a flapping link. Another name for this is the networking "wall of shame". |
| {{< figure src="activity-trend.png" alt="Activity Trend" loading="lazy" >}} | A rolling 60-second activity graph. It tracks whether activity is spiking or calming down, letting you see the difference between routine background noise and a massive routing event. |
| {{< figure src="beacon-analysis.png" alt="Beacon Analysis" loading="lazy" >}} | A dynamic donut chart separating "Organic" traffic from "Beacons" (special test signals sent out by researchers). It helps show how much activity is natural versus intentional measurement. More on this below. |
| {{< figure src="now-playing.png" alt="Now Playing" loading="lazy" >}} | The current background music track. |

### Path Hunting and Anycast

When I first started watching the live data, I was confused by why a single localized outage would trigger a massive global explosion of pulses. 

I've since learned this is likely due to a phenomenon called ["Path Hunting."](https://blog.cloudflare.com/going-bgp-zombie-hunting/) When a route dies, the Internet doesn't instantly agree it's gone. Instead, routers desperately try to find backup paths. They'll try a longer route, fail, try an even longer one, fail again, and generate a new BGP update every single time.  Those massive bursts of purple pulses are basically the routers "thinking out loud" as they scramble to route around the damage.

{{< diagram caption="[Routers looking for Facebook's network on October 4, 2021](https://blog.cloudflare.com/october-2021-facebook-outage/)" >}}
{{< image src="asn32934.gif" >}}
{{< /diagram >}}

**Anycast** routing amplifies this chatter even further. Huge networks (like Google or Cloudflare) announce the exact same `/24` prefix from dozens of different physical locations globally so their services are fast everywhere.  But if a major transit provider drops a peering session, or a provider intentionally shifts traffic away from a datacenter for maintenance, thousands of routers might suddenly decide to shift their traffic to a different Anycast node all at once. The result is a sudden surge of routing adjustments across the map.

{{< diagram >}}
{{< image src="spiderman-meme.png" >}}
{{< /diagram >}}

### RIPE RIS Beacons and Anchors

While building the "Most Active Prefixes" list, I kept noticing the exact same thing: `/24` subnets were overrepresented on the leaderboard. 

A `/24` (256 IPs) is effectively the smallest globally routable unit, so most churn naturally happens at that granularity.

But there was another reason for seeing the *same* /24 subnets appearing on the list. Not all activity on the map comes from failing links or organic traffic shifts. There is also intentional 'breakage' happening behind the scenes to test BGP propagation.

It turns out RIPE RIS operates [Routing Beacons](https://ris.ripe.net/docs/routing-beacons/). Routing Beacons are prefixes deliberately announced and withdrawn on a fixed schedule, typically every two hours. One of them announces and withdraws every *10 minutes*. Researchers use these beacons as a controlled signal inside the global routing table to study BGP propagation and convergence. To make the activity list useful, I had to write logic to classify and filter these beacons out of the ranking.

RIPE also runs "Anchors" alongside these beacons. While a beacon prefix constantly flips on and off, an anchor is a prefix permanently announced from the exact same physical router. This gives researchers a stable control group. They can compare the volatile beacon traffic against a baseline of stable routing from the identical location.

I eventually added a Beacon Analysis view that separates "organic" updates from beacon-driven ones. It makes the metrics more accurate and highlights how much traffic is from deliberate live validation.

### BGP Babbling and Attribute Churn

So if a burst of updates isn't a dying link, a desperate search for a backup path, or a research beacon, what else could it be? Sometimes a network is just fidgeting. BGP engineers usually call this **babbling**.

I caught a great example of this while watching the stream. A Finnish fiber provider (`AS43016`) was firing off nearly 100 pulses per second, and this went on for days. The raw data showed the route wasn't actually dropping. Instead, a single piece of metadata called the Aggregator ID just kept flipping back and forth.

This creates a localized flurry of activity. Some router somewhere was probably misconfigured and couldn’t make up its mind about how to summarize its own network. Every time it changed its mind, even by a single bit, it had to update every other router on Earth. Standard monitoring tools usually miss these "attribute flaps" because the network stays perfectly reachable. But on the map, they paint a very clear picture: a constant, rhythmic heartbeat of blue gossip pulses.

I built [a tool](https://github.com/sudorandom/bgp-stream/blob/main/cmd/debug-prefix/main.go) to debug noisy prefixes like this. It aggregates BGP update stats and tries to diagnose the root cause, such as path oscillation, a flapping link, or heavy Anycast routing. Here is the output for our problem child over at `AS43016`:

```shell
$ just debug-prefix 195.155.146.0/24
BGP Prefix Monitor Stats (Running for 293.4s)
--------------------------------------------------
Announcements: 4576 (15.60/s)
Withdrawals:   1422 (4.85/s)
Total Msgs:    5101 (17.39/s)
Unique Peers:  310
--------------------------------------------------
GLOBAL CHURN EVENTS:
  AS-Path Changes:  2275
  Community Changes: 3259
  Next-Hop Changes:  0
  Aggregator Flaps:  0
  Path Length Flaps: 1255
--------------------------------------------------
LIKELY CONCLUSIONS:
  - Path Length Oscillation (Route is toggling between different path lengths)
  - BGP Babbling (Excessive update rate detected)
--------------------------------------------------
Top 5 Churning Peers:
  187.16.220.216: 149 attribute changes
  5.188.4.211: 142 attribute changes
  103.152.35.254: 142 attribute changes
  177.221.140.2: 138 attribute changes
  154.18.4.110: 132 attribute changes
```

At the time of publishing, this prefix is still babbling away.

---

{{< diagram >}}
{{< image src="spiderman-meme-2.jpg" >}}
{{< /diagram >}}

## Making the map

Handling 30,000+ BGP updates per second takes more than plotting points on a canvas. The project is written in Go for its concurrency model and relies on Ebitengine for hardware-accelerated 2D rendering. 

### Why a Stream?

I originally planned to build this as a standard web frontend, similar to my [previous map](https://kmcd.dev). However, I hit two massive walls almost immediately.

The first problem was the sheer volume of data. BGP updates can easily peak at over 30,000 events per second. Forcing a web browser to process that firehose while maintaining a smooth 30 FPS with complex blending is just not in the cards today.

The second problem was scaling. If the map actually got popular, having thousands of browsers opening individual websocket connections to the RIPE RIS-Live service would be a disaster. It is wildly inefficient, and accidentally DDoSing a service designed to monitor Internet stability was not on my to-do list.

Here is what that scenario looks like:

{{< d2 >}}
direction: right

classes: {
  cloud: {
    shape: cloud
    style: {
      fill: "#1a252f"
      stroke: "#34495e"
      font-color: "#ecf0f1"
    }
  }
  browser: {
    shape: page
    style: {
      fill: "#2c3e50"
      stroke: "#f1c40f"
      font-color: "#ecf0f1"
    }
  }
  bad_connection: {
    style: {
      stroke: "#e74c3c"
      stroke-width: 2
      stroke-dash: 5
      font-color: "#ecf0f1"
    }
  }
  scenario_box: {
    style: {
      fill: transparent
      stroke: "#555555"
      stroke-width: 1
      border-radius: 10
      font-color: "#ecf0f1"
    }
  }
  invisible_box: {
    style: {
      fill: transparent
      stroke: transparent
      font-color: "#ecf0f1"
    }
  }
}

"Scenario 1: Direct Browser Connections": {
  class: scenario_box
  direction: right

  RIPE: RIPE RIS-Live Service 😔 {class: cloud}

  Browsers: {
    class: invisible_box
    direction: down
    B1: Browser 1 (Renderer) {class: browser}
    B2: Browser 2 (Renderer) {class: browser}
    B3: ... {shape: circle; width: 20; height: 20; style: {stroke: transparent; fill: transparent; font-color: "#ecf0f1"}}
    BN: Browser N (Renderer) {class: browser}
  }

  RIPE -> Browsers.B1: "Many WS Connections" {class: bad_connection}
  RIPE -> Browsers.B2: {class: bad_connection}
  RIPE -> Browsers.BN: {class: bad_connection}
}
{{< /d2 >}}

To protect the RIPE service from being overwhelmed, the logical next step was to put a middleman in place to handle the multiplexing. This led me to a standard client-server architecture:

{{< d2 >}}
direction: right

classes: {
  cloud: {
    shape: cloud
    style: {
      fill: "#1a252f"
      stroke: "#34495e"
      font-color: "#ecf0f1"
    }
  }
  server: {
    shape: cylinder
    style: {
      fill: "#0b5345"
      stroke: "#1abc9c"
      font-color: "#ecf0f1"
    }
  }
  browser: {
    shape: page
    style: {
      fill: "#2c3e50"
      stroke: "#f1c40f"
      font-color: "#ecf0f1"
    }
  }
  good_connection: {
    style: {
      stroke: "#2ecc71"
      stroke-width: 2
      font-color: "#ecf0f1"
    }
  }
  neutral_connection: {
    style: {
      stroke: "#95a5a6"
      stroke-width: 2
      font-color: "#ecf0f1"
    }
  }
  scenario_box: {
    style: {
      fill: transparent
      stroke: "#555555"
      stroke-width: 1
      border-radius: 10
      font-color: "#ecf0f1"
    }
  }
  invisible_box: {
    style: {
      fill: transparent
      stroke: transparent
      font-color: "#ecf0f1"
    }
  }
}

"Scenario 2: Server Multiplexing": {
  class: scenario_box
  direction: right

  RIPE: RIPE RIS-Live Service {class: cloud}
  Server: Relay Server {class: server}

  Browsers: {
    class: invisible_box
    direction: down
    B1: Browser 1 (Renderer) {class: browser}
    B2: Browser 2 (Renderer) {class: browser}
    B3: ... {shape: circle; width: 20; height: 20; style: {stroke: transparent; fill: transparent; font-color: "#ecf0f1"}}
    BN: Browser N (Renderer) {class: browser}
  }

  RIPE -> Server: "Single WS" {class: good_connection}
  Server -> Browsers.B1: "Many WS Connections" {class: neutral_connection}
  Server -> Browsers.B2: {class: neutral_connection}
  Server -> Browsers.BN: {class: neutral_connection}
}
{{< /d2 >}}

Multiplexing solves the connection problem, but it completely ignores the browser rendering issues I was having. To guarantee a smooth 30 FPS for everyone without melting their CPUs, I decided to bypass the browser canvas entirely. I pivoted the architecture to a centralized video stream broadcasted to YouTube:

{{< d2 >}}
direction: right

classes: {
  cloud: {
    shape: cloud
    style: {
      fill: "#1a252f"
      stroke: "#34495e"
      font-color: "#ecf0f1"
    }
  }
  server: {
    shape: cylinder
    style: {
      fill: "#0b5345"
      stroke: "#1abc9c"
      font-color: "#ecf0f1"
    }
  }
  browser: {
    shape: page
    style: {
      fill: "#2c3e50"
      stroke: "#f1c40f"
      font-color: "#ecf0f1"
    }
  }
  mobile: {
    shape: rectangle
    style: {
      fill: "#2c3e50"
      stroke: "#f1c40f"
      font-color: "#ecf0f1"
      border-radius: 15
    }
  }
  tv: {
    shape: rectangle
    style: {
      fill: "#2c3e50"
      stroke: "#f1c40f"
      font-color: "#ecf0f1"
      border-radius: 2
    }
  }
  device: {
    shape: rectangle
    style: {
      fill: "#2c3e50"
      stroke: "#f1c40f"
      font-color: "#ecf0f1"
      border-radius: 5
    }
  }
  platform: {
    shape: rectangle
    style: {
      fill: "#641e16"
      stroke: "#e74c3c"
      border-radius: 10
      font-color: "#ecf0f1"
    }
  }
  good_connection: {
    style: {
      stroke: "#2ecc71"
      stroke-width: 2
      font-color: "#ecf0f1"
    }
  }
  scenario_box: {
    style: {
      fill: transparent
      stroke: "#555555"
      stroke-width: 1
      border-radius: 10
      font-color: "#ecf0f1"
    }
  }
  invisible_box: {
    style: {
      fill: transparent
      stroke: transparent
      font-color: "#ecf0f1"
    }
  }
}

"Scenario 3: Video Streaming": {
  class: scenario_box
  direction: right

  RIPE: RIPE RIS-Live Service {class: cloud}
  Server: Rendering (Ebitengine) {class: server}
  YouTube: YouTube Live {class: platform}

  Devices: {
    class: invisible_box
    direction: down
    D1: Web Browser {class: browser}
    D2: Mobile Phone {class: mobile}
    D3: Smart TV {class: tv}
    D4: ... {shape: circle; width: 20; height: 20; style: {stroke: transparent; fill: transparent; font-color: "#ecf0f1"}}
    DN: Any Screen {class: device}
  }

  RIPE -> Server: "Single WS" {class: good_connection}
  Server -> YouTube: "RTMP Video Stream" {class: good_connection}
  YouTube -> Devices.D1: "Video Stream" {class: good_connection}
  YouTube -> Devices.D2: {class: good_connection}
  YouTube -> Devices.D3: {class: good_connection}
  YouTube -> Devices.DN: {class: good_connection}
}
{{< /d2 >}}

Now I had a choice. Scenario 1 was dead on arrival because it could make the operators of RIPE RIS-Live *very sad* and potentially angry. That left me with the choice between building a complex backend service to multiplex that single RIPE connection to all my users (Scenario 2), or completely changing how people view the map by streaming to YouTube (Scenario 3). I went with the latter option.

Rendering the entire visualization on my own server and broadcasting it guarantees that every viewer gets the exact same high-fidelity experience, regardless of their hardware. It is easy to run on a TV where the browser version isn't really viable. This pivot also made the tech stack an obvious choice. Once I started experimenting with [Ebitengine](https://ebitengine.org/), hardware-accelerated rendering in Go gave me crisper, far more fluid visuals than I could ever squeeze out of a standard browser canvas.

The downside is reduced interaction: no zooming, no toggling UI, no customization. I think this tradeoff was ultimately worth it, but I just want to note what I lost from making this dramatic change in architecture.

### Flattening IP Space

To map a BGP update to a geographic location, you need reliable IP-to-region data. I am currently only focusing on IPv4, and that data comes from five Regional Internet Registries (RIRs). Each registry publishes large and sometimes overlapping delegated stats files.

Fragmented lookups across raw datasets might be fine for offline processing, but we have a strict frame rate budget. If the engine had to search through five separate datasets for every single update, the visualization would stutter. At 30,000+ updates per second, efficiency is pretty important.

To solve this, I preprocess all the data upfront using a sweep-line algorithm. Each IP range acts as a segment on a 1D number line. The algorithm walks across this space, resolves any overlaps between registries, and collapses millions of ranges into a single, clean, non-overlapping index.

For example, take two overlapping registry entries:
* **Range A (ARIN):** `10.0.0.0` to `10.0.0.255`
* **Range B (RIPE):** `10.0.0.128` to `10.0.1.255`

The algorithm flattens these into three distinct, non-overlapping segments:
1. `10.0.0.0` to `10.0.0.127` (ARIN only)
2. `10.0.0.128` to `10.0.0.255` (Conflict resolved)
3. `10.0.1.0` to `10.0.1.255` (RIPE only)

This preprocessing seems a bit complex, but it's worth it since it makes the live lookups super cheap. I back this index with BadgerDB and a [DiskTrie for high-performance persistent storage](https://github.com/sudorandom/bgp-stream/blob/main/pkg/utils/disk_trie.go). This allows the engine to track "seen" prefixes seamlessly across different sessions without eating up memory.

### Managing the Firehose

BGP updates arrive continuously, and during route flapping events the volume spikes hard.

To keep the visualization readable without becoming an incomprehensible mess, the pipeline waits 10 seconds to ensure a withdrawal isn't just a rapid path re-convergence, and paces the visual output so spikes are emitted smoothly every 500ms.

{{< d2 >}}
direction: down

classes: {
  process: {
    shape: rectangle
    style: {border-radius: 5}
  }
  db: {
    shape: cylinder
  }
}

Input: BGP Firehose\n(10k+ updates/sec) {
  shape: cloud
}

Batch: 10s Batch & Wait\n(Deduplication & Resolution) {
  class: process
}

Pace: 500ms Paced Emission\n(Log-scaled buffer) {
  class: process
}

Cache: Prefix Cache\n(Seen Prefixes) {
  class: db
}

Output: Live Map Canvas {
  shape: rectangle
}

Input -> Batch: Raw Stream
Batch -> Pace: Stable Events
Batch -> Cache: Store state
Pace -> Output: Smooth Render
{{< /d2 >}}

## Aesthetics, Motion, and Sound

Animations use interpolation instead of snapping to the next state. For parts of the map which update infrequently, I wanted to highlight that a change occurred. For that, I added a "glitch" effect to the "Top Activity Hubs" and "Most Active Prefixes" to make it more obvious and to add to the cyberpunk aesthetic. These effects add polish, but too much motion distracts. Finding that balance took restraint and a surprisingly large amount of experimentation.

The pulses are what actually bring the data to life. In the engine, each pulse is a simple generated glow texture. I add a bit of spatial jitter so concurrent events do not stack perfectly on top of each other, and I scale their sizes logarithmically so massive data spikes do not turn the map into a solid wall of color.

The colors map directly to the event types: green for new paths, purple for updates, red for withdrawals, and blue for gossip. Because they use additive blending, overlapping pulses naturally create a bright hotspot over regions with a ton of routing activity. They pop onto the map, expand, and fade out smoothly. Managing this entire visual lifecycle efficiently is what keeps the map feeling dynamic without tanking the frame rate.

{{< diagram >}}
{{< figure src="europe-animation.webp" caption="Animation of BGP events in Europe" animate="true" width="700px" animate="true" >}}
{{< /diagram >}}

#### The Mollweide Projection

Mercator would have been easy, but it heavily distorts size near the poles. For a global activity map, that felt misleading.

I chose the [Mollweide projection](https://en.wikipedia.org/wiki/Mollweide_projection).

{{< diagram caption="[By Justin Kunimune - Own work, CC BY-SA 4.0](https://commons.wikimedia.org/w/index.php?curid=66467569)" >}}
{{< figure src="mollweide.svg" width="700px" >}}
{{< /diagram >}}

This is an equal-area projection, which means it accurately represents the physical footprint of different regions. It produces a world view that still feels familiar without exaggerating high-latitude areas.

---

Here is the final result, which I've gazed at for far too long already:

{{< a href="http://livemap.kmcd.dev" target="_blank" >}}
{{< diagram >}}
{{< figure src="map-animation.webp" caption="🔴 [livemap.kmcd.dev](http://livemap.kmcd.dev)" animate="true" width="600px" >}}
{{< /diagram >}}
{{< /a >}}

This project turned into a deeper dive into BGP than I expected. Watching as routing updates happen live exposes patterns that are impossible to find with a static snapshot. It has been a rewarding project and I am extremely happy with the result.

So please, toss the live stream on your TV, sit back, relax, and watch the Internet route the world's network traffic as you listen to relaxing lofi in the background.
