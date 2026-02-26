---
categories: ["article", "project"]
tags: ["dataviz", "internet-map", "internet", "networking", "bgp", "map", "world", "infrastructure", "ebitengine", "go"]
date: "2026-03-19T10:00:00Z"
description: "Building a cool looking, real-time BGP map"
cover: "map.png"
images: ["/posts/live-internet-map/map.png"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Live Internet Map"
slug: "live-internet-map"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/live-internet-map/
---

The internet runs on constant routing updates. BGP (Border Gateway Protocol) continuously announces and withdraws prefixes, adjusting how traffic moves between networks. Most people see URLs and apps. Routers see prefixes and AS paths.

I wanted to build a real-time visualization of that stream. I needed something that shows where routing activity is happening and how it shifts over time.

In my [last post](/posts/internet-map-2026/) about my [Internet Infrastructure Map](map.kmcd.dev), I mentioned a few alternative sources for BGP data that I didn't end up using. One of them was a [websocket-based streaming API](https://ris-live.ripe.net/) from RIPE. At the time, I set it aside. Soon, it became the center point of my curiosity and the live view was born.

You can check out the source code for this project on [GitHub](https://github.com/sudorandom/bgp-stream/) or watch the map in action on my [YouTube channel](https://www.youtube.com/channel/UCA9eO4Gt-Ua6lAEGzWQHQFA/live) or here:

{{< youtube-live channel="UCA9eO4Gt-Ua6lAEGzWQHQFA" >}}

## What are we looking at?

This map is a live visualization of the Border Gateway Protocol (BGP). This is the "language" routers use to talk to each other and decide the best path for your data to travel across the globe. 

To visualize this, imagine a single router trying to find the best way to send traffic to Google. It receives multiple path advertisements from its neighbors, and it has to pick the most efficient route:

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

In networking, flapping happens when a route rapidly appears and disappears. Imagine a misconfigured router or a loose fiber cable. The router yells to the internet, "I have a path to Google!" only to drop the connection a second later and say, "Never mind, it is gone." Because BGP is designed to spread information globally, that single localized hiccup does not stay local. It sends a ripple effect across the map as thousands of routers worldwide are forced to constantly recalculate their paths. To keep the whole system from grinding to a halt, modern routers use Route Flap Damping. This essentially puts the noisy network in a time-out until it proves it can stay stable.

### Decoding the Pulses

When you see those colored pulses popping off on the map, they represent specific BGP message types. "Updates" is the general term, but under the hood, the protocol is juggling a few distinct events:

* **Announcements (Green):** A network is declaring a new path is open for business. This happens when a new ISP goes live, a company spins up a new datacenter, or a router discovers a more efficient shortcut. On the map, these are the bright **green** pulses.
* **Withdrawals (Red):** The network equivalent of a dropped call. A router explicitly tells its neighbors that a previously advertised IP block is no longer reachable. This usually points to a link failure, a maintenance window, or a severed cable. These hit the map in **red**.
* **Path Attributes (Purple):** The destination is still online, but the directions to get there have changed. Maybe traffic now has to hop through three transit providers instead of two. I mapped these routing adjustments as **purple**.
* **Gossip and Keepalives (Blue):** Routers are incredibly chatty. They constantly send "heartbeats" to each other to prove they are still alive. If a router stops hearing these keepalives, it assumes its neighbor is dead and drops all associated routes. This constant background chatter shows up as **blue** pulses.

When you zoom out and look at all those colors firing at once, the scale starts to make sense.  The internet is a collection of over 70,000 independent networks coordinating through BGP. This map visualizes that global coordination as it happens.

## What else is on the map?

To make sure the map isn't just a wall of moving dots, I included several dashboard elements that provide context to the chaos:

* **Top Activity Hubs:** This ranks the countries currently experiencing the highest volume of network updates. It is an instant look at where in the world the most "routing churn" is happening at any given moment.
* **Most Active Prefixes:** While the hubs show countries, this looks at the specific individual network blocks causing the most noise. This is where you can often spot large-scale outages or specific maintenance windows as they happen.
* **Activity Trend (1m):** A rolling 60-second activity graph. It tracks whether activity is spiking or calming down, letting you see the difference between routine background noise and a massive routing event.
* **Beacon Analysis:** A dynamic donut chart that separates "Organic" traffic from "Beacons", which are special test signals sent out by researchers. It helps you understand how much of the activity you see is natural internet behavior versus intentional scientific measurement.
* **Now Playing:** For the stream, I added a section showing the current background music track.

---

## Tech Details

Handling 10,000+ BGP updates per second takes more than plotting points on a canvas. The project is written in Go for its concurrency model and relies on Ebiten for hardware-accelerated 2D rendering. Here is what makes the system work.

### Flattening IP Space with a Sweep-Line Algorithm

To map a BGP update to a geographic location, you need reliable IP-to-region data. Note that we are only dealing with IPv4 at the moment. That data comes from five Regional Internet Registries (RIRs), each publishing large, sometimes overlapping delegated stats files.

Doing fragmented lookups across raw datasets might be sufficient when doing offline processing of data. But this is live data, and we have a frame rate budget that we have to keep. If the engine had to search through five separate datasets for every single update, the visualization would immediately grind to a halt. At 10,000+ updates per second, efficiency is non-negotiable. 

To solve this, I preprocess all the data upfront using a sweep-line algorithm. Each IP range is treated as a segment on a 1D number line. The algorithm walks across this space, resolving any overlaps between the different registries, and collapses millions of ranges into a single, clean, non-overlapping index.

For example, if you have two overlapping registry entries:
* **Range A (ARIN):** `10.0.0.0` to `10.0.0.255`
* **Range B (RIPE):** `10.0.0.128` to `10.0.1.255`

The algorithm flattens these into three distinct, non-overlapping segments:
1. `10.0.0.0` to `10.0.0.127` (ARIN only)
2. `10.0.0.128` to `10.0.0.255` (Conflict resolved)
3. `10.0.1.0` to `10.0.1.255` (RIPE only)

The result is a compact structure that supports fast lookups in logarithmic time. The preprocessing is expensive, but lookups are cheap. I back this with BadgerDB and a DiskTrie for high-performance persistent storage, which allows the engine to track "seen" prefixes across different sessions.

### High-Precision Cloud Mapping

Relying solely on generic GeoIP data to map cloud providers usually results in inaccuracies. An AWS prefix might be officially registered to a corporate address in the US, while the actual infrastructure for that block is sitting in a datacenter in Tokyo. 

Drawing from the lessons I learned mapping [map.kmcd.dev](https://map.kmcd.dev), I built a CloudTrie structure to improve accuracy. It ingests official geofeeds and provider IP range JSONs like AWS and Google Cloud. When a route change happens inside a cloud prefix, the pulse appears near the actual physical infrastructure footprint rather than the corporate registration address.

It’s not perfect, but it’s more accurate than registry data alone.

### Managing the Firehose

BGP updates arrive continuously, and during route flapping events the volume spikes hard.

To keep the visualization readable and performant, the pipeline includes a multi-stage classification engine:

* **Deduplication:** Filters out redundant updates for the same prefix within a 15-second window.
* **Withdraw Resolution:** Uses a 10-second wait window to distinguish between a simple withdrawal and a rapid path re-convergence.
* **Event Classification:** Categorizes updates into New Paths, Path Changes, Withdrawals, and Propagation (Gossip).
* **Paced Emission:** BGP spikes are buffered and emitted into the visual queue every 500ms using logarithmic scaling to handle the massive dynamic range of BGP activity.

{{< d2 width="500px" >}}
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

The goal is to stay accurate without overwhelming the screen during instability.

## RIPE RIS Beacons and Anchors

BGP is not purely organic traffic. I actually discovered this by accident while building the "Most Active Prefixes" metric. Initially, a handful of prefixes completely dominated the top of the list, drowning out genuine network events. 

It turns out RIPE RIS operates [Routing Beacons](https://ris.ripe.net/docs/routing-beacons/). These are prefixes deliberately announced and withdrawn on a fixed schedule, typically every two hours. Researchers use them as a controlled signal inside the global routing table to study BGP propagation and convergence. To make the activity list useful, I had to write logic to classify and filter these beacons out of the ranking.

RIPE also runs "Anchors" alongside these beacons. While a beacon prefix constantly flips on and off, an anchor is a prefix permanently announced from the exact same physical router. This gives researchers a stable control group. They can compare the volatile beacon traffic against a baseline of stable routing from the identical location.

I eventually added a Beacon Analysis view that separates "organic" updates from beacon-driven ones. It makes the metrics more accurate and highlights how much traffic is deliberate measurement.

## Aesthetics, Motion, and Sound

Once the data pipeline was stable, the remaining work was about clarity and immersion.

Animations use interpolation instead of snapping to the next state. Country rankings slide into position. Percentages ease between values. Even small UI transitions are smoothed out. These details significantly improve the polish of the stream, but it is definitely a balancing act. Too much movement can distract from the visual effect of the map itself, so getting this right required some restraint.

The pulses are what actually bring the data to life. Under the hood, each pulse is a simple generated glow texture. I add a bit of spatial jitter so concurrent events do not stack perfectly on top of each other, and I scale their sizes logarithmically so massive data spikes do not turn the map into a solid wall of color.

The colors map directly to the event types: green for new paths, purple for updates, red for withdrawals, and blue for gossip. Because they use additive blending, overlapping pulses naturally create a bright hotspot over regions with a ton of routing activity. They pop onto the map, expand, and fade out smoothly. Managing this entire visual lifecycle efficiently is what keeps the map feeling dynamic without tanking the frame rate.

{{< diagram >}}
{{< image src="europe-animation.webp" caption="Animation of BGP events in Europe" animate="true" width="700px" >}}
{{< /diagram >}}

#### Projection Choice: Winkel Tripel

Mercator would have been easy, but it heavily distorts size near the poles. For a global activity map, that felt misleading. 

I chose the [Winkel Tripel projection](https://en.wikipedia.org/wiki/Winkel_tripel_projection).

{{< diagram caption="[By Justin Kunimune - Own work, CC0](https://commons.wikimedia.org/w/index.php?curid=66467590)" >}}
{{< image src="Winkel_Tripel.svg" width="700px" >}}
{{< /diagram >}}

 This is the same one used by National Geographic, and it balances distortion across area, direction, and distance. It produces a world view that feels familiar without exaggerating high-latitude regions.

Implementing it meant solving the projection's iterative trigonometric equations for every coordinate transformation. It was a bit of a headache, specifically because I had issues drawing the longitude and latitude lines. There is a mathematical singularity at the very top and bottom of the map that breaks the rendering, so I had to write custom workarounds to handle the poles gracefully.

---

All of that effort sums up into an amazing visualization, which I've gazed at for far too long already:

{{< a href="https://www.youtube.com/channel/UCA9eO4Gt-Ua6lAEGzWQHQFA/live" target="_blank" >}}
{{< diagram caption="🔴 [sudorandom on youtube](https://www.youtube.com/channel/UCA9eO4Gt-Ua6lAEGzWQHQFA/live)" >}}
{{< image src="map-animation.webp" animate="true" width="600px" >}}
{{< /diagram >}}
{{< /a >}}

This project turned into a deeper dive into BGP than I expected. Watching routing updates happening live exposes patterns that are impossible to find with a static snapshot.
