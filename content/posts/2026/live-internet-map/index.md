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
title: "Live Internet Map"
slug: "live-internet-map"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/live-internet-map/
---

The internet runs on constant routing updates. BGP (Border Gateway Protocol) continuously announces and withdraws prefixes, adjusting how traffic moves between networks. Most people see URLs and apps. Routers see prefixes and AS paths.

I wanted to build a real-time visualization of that stream. I needed something that shows where routing activity is happening and how it shifts over time.

In my [last post](/posts/internet-map-2026/) about my [Internet Infrastructure Map](map.kmcd.dev), I mentioned a few alternative sources for BGP data that I didn't end up using. One of them was a [websocket-based streaming API](https://ris-live.ripe.net/) from RIPE. At the time, I set it aside. Soon, it became my obsession and the live view was born.

I do want to clarify one thing up front. While this visualization does occasionally stumble into being practically useful for spotting global outages or routing leaks, let us be completely honest with ourselves. The primary requirement for this project was simply to build a really cool looking map.

You can check out the source code for this project on [GitHub](https://github.com/sudorandom/bgp-stream/) or watch the map in action on my [YouTube channel](https://www.youtube.com/channel/UCA9eO4Gt-Ua6lAEGzWQHQFA/live) or here:

{{< youtube-live channel="UCA9eO4Gt-Ua6lAEGzWQHQFA" >}}

## What are we looking at?

This map is a live visualization of the Border Gateway Protocol (BGP). This is the "language" routers use to talk to each other and decide the best path for your data to travel across the globe. 

Imagine a single router trying to find the best way to send traffic to Google. It receives multiple path advertisements from its neighbors, and it has to pick the most efficient route:

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

In networking, flapping happens when a route rapidly appears and disappears. Imagine a misconfigured router or a loose fiber cable. The router yells to the internet, "I have a path to Google!" only to drop the connection a second later and say, "Never mind, it is gone". Because BGP is designed to spread information globally, that single localized hiccup does not stay local. It sends a ripple effect across the map as thousands of routers worldwide are forced to constantly recalculate their paths. To keep the whole system from grinding to a halt, modern routers use Route Flap Damping. This essentially puts the noisy network in a time-out until it proves it can stay stable.


### Decoding the Pulses

When you see those colored pulses popping off on the map, they represent specific BGP message types. "Updates" is the general term, but practically, the protocol is juggling a few distinct events:

* **Announcements (Green):** Bright green pulses mean a new path just opened up. This could be a new ISP coming online, a fresh datacenter spinning up, or just a router discovering a better shortcut.
* **Withdrawals (Red):** Red means a route is dead. A router is explicitly telling the internet that a previously advertised IP block is no longer reachable—usually the result of a severed fiber cable, hardware failure, or a planned maintenance window.
* **Path Attributes (Purple):** The destination is still online, but the directions changed. If traffic suddenly has to detour through an extra transit provider to reach its goal, you'll see those routing adjustments flash purple.
* **Gossip (Blue):** Routers frequently re-announce perfectly valid paths just to keep their tables current; this redundant background noise makes up the blue pulses on the map.

When you zoom out and look at all those colors firing at once, the scale starts to make sense. The internet is a collection of over 70,000 independent networks coordinating through BGP. This map visualizes that global coordination as it happens.

## What else is on the map?

To make sure the map isn't just a wall of moving dots, I included several dashboard elements that provide context to the chaos:

* **Top Activity Hubs:** This ranks the countries currently experiencing the highest volume of network updates. It is an instant look at where in the world the most "routing churn" is happening at any given moment.
* **Most Active Prefixes:** While the hubs show countries, this tracks the specific network blocks causing the most noise. Great for spotting large-scale outages.
* **Activity Trend (1m):** A rolling 60-second activity graph. It tracks whether activity is spiking or calming down, letting you see the difference between routine background noise and a massive routing event.
* **Beacon Analysis:** A dynamic donut chart that separates "Organic" traffic from "Beacons", which are special test signals sent out by researchers. It helps you understand how much of the activity you see is natural internet behavior versus intentional scientific measurement.
* **Now Playing:** The current background music track.


### The Ubiquity of /24s

While building the "Most Active Prefixes" dashboard, I kept noticing the exact same thing: `/24` subnets were overrepresented on the leaderboard. 

From what I've learned, a `/24` (256 IP addresses) is the smallest block of IPs that major ISPs will actually accept and pass along. Because `/24` is the standard unit of the global routing table, most routing churn, whether it is a small office link flapping or a major datacenter shifting traffic, happens at this granularity.

### Path Hunting and Anycast

When I first started watching the live data, I was confused by why a single localized outage would trigger a massive global explosion of pulses. 

I've since learned this is likely due to a phenomenon called ["Path Hunting."](https://blog.cloudflare.com/going-bgp-zombie-hunting/) When a route dies, the internet doesn't instantly agree it's gone. Instead, routers desperately try to find backup paths. They'll try a longer route, fail, try an even longer one, fail again, and generate a new BGP update every single time.  Those massive bursts of purple pulses are basically the routers "thinking out loud" as they scramble to route around the damage.

{{< diagram >}}
{{< image src="asn32934.gif" >}}
{{< /diagram >}}

**Anycast** routing amplifies this chatter even further. Huge networks (like Google or Cloudflare) announce the exact same `/24` prefix from dozens of different physical locations globally so their services are fast everywhere.  But if a major transit provider drops a peering session, or a provider intentionally shifts traffic away from a datacenter for maintenance, thousands of routers might suddenly decide to shift their traffic to a different Anycast node all at once. The result is a massive visual wave of routing adjustments ripping across the map.

{{< diagram >}}
{{< image src="spiderman-meme.png" >}}
{{< /diagram >}}

### RIPE RIS Beacons and Anchors

Not all activity on the map comes from failing links or organic traffic shifts. There is also intentional 'breakage' happening behind the scenes to test BGP propagation.

It turns out RIPE RIS operates [Routing Beacons](https://ris.ripe.net/docs/routing-beacons/). Routing Beacons are prefixes deliberately announced and withdrawn on a fixed schedule, typically every two hours. One of them announces and withdrawals every *10 minutes*. Researchers use these beacons as a controlled signal inside the global routing table to study BGP propagation and convergence. To make the activity list useful, I had to write logic to classify and filter these beacons out of the ranking.

RIPE also runs "Anchors" alongside these beacons. While a beacon prefix constantly flips on and off, an anchor is a prefix permanently announced from the exact same physical router. This gives researchers a stable control group. They can compare the volatile beacon traffic against a baseline of stable routing from the identical location.

I eventually added a Beacon Analysis view that separates "organic" updates from beacon-driven ones. It makes the metrics more accurate and highlights how much traffic is deliberate measurement.

---

## Tech Details

Handling 30,000+ BGP updates per second takes more than plotting points on a canvas. The project is written in Go for its concurrency model and relies on Ebitengine for hardware-accelerated 2D rendering. 

### Why a Stream?

I originally planned to build this as a standard web frontend, similar to my [previous map](https://kmcd.dev). However, I hit two massive walls almost immediately.

The first problem was the sheer volume of data. BGP updates can easily peak at over 30,000 events per second. Forcing a web browser to process that firehose while maintaining a smooth 60 FPS with complex blending is a great way to melt a user's laptop.

The second problem was scaling. If the map actually got popular, having thousands of browsers opening individual websocket connections to the RIPE RIS-Live service would be a disaster. It is wildly inefficient, and accidentally DDoSing a service designed to monitor internet stability was not on my to-do list.

I had a choice. I could build a complex backend service to multiplex that single RIPE connection to all my users, or I could completely change how people view the map. I chose the latter and pivoted to a live video stream.

Rendering the entire visualization on my own server and broadcasting it guarantees that every viewer gets the exact same high-fidelity experience, regardless of their hardware. This pivot also made the tech stack an easy choice. Once I started experimenting with [Ebitengine](https://ebitengine.org/), hardware-accelerated rendering in Go gave me crisper, far more fluid visuals than I could ever squeeze out of a standard browser canvas.

### Flattening IP Space with a Sweep-Line Algorithm

To map a BGP update to a geographic location, you need reliable IP-to-region data. I am currently only focusing on IPv4, and that data comes from five Regional Internet Registries (RIRs). Each registry publishes large and sometimes overlapping delegated stats files.

Fragmented lookups across raw datasets might be fine for offline processing. But this is live data, and there is a strict frame rate budget. If the engine had to search through five separate datasets for every single update, the visualization would immediately grind to a halt. At 30,000+ updates per second, efficiency is non-negotiable.

To solve this, I preprocess all the data upfront using a sweep-line algorithm. Each IP range acts as a segment on a 1D number line. The algorithm walks across this space, resolves any overlaps between registries, and collapses millions of ranges into a single, clean, non-overlapping index.

For example, take two overlapping registry entries:
* **Range A (ARIN):** `10.0.0.0` to `10.0.0.255`
* **Range B (RIPE):** `10.0.0.128` to `10.0.1.255`

The algorithm flattens these into three distinct, non-overlapping segments:
1. `10.0.0.0` to `10.0.0.127` (ARIN only)
2. `10.0.0.128` to `10.0.0.255` (Conflict resolved)
3. `10.0.1.0` to `10.0.1.255` (RIPE only)

This preprocessing seems a bit complex, but it's worth it since it makes the live lookups dirt cheap. I back this index with BadgerDB and a DiskTrie for high-performance persistent storage. This allows the engine to track "seen" prefixes seamlessly across different sessions without eating up memory.

### High-Precision Cloud Mapping

Relying solely on generic GeoIP data to map cloud providers usually leads to glaring inaccuracies. An AWS prefix might be officially registered to a corporate address in the US, but the actual infrastructure for that block could be sitting in a datacenter in Tokyo.

Drawing from the lessons I learned building [map.kmcd.dev](https://map.kmcd.dev), I designed yet another trie data structure to fix this, but this time it's in memory. It ingests official geofeeds and provider IP range JSONs from networks like AWS and Google Cloud.

Now, when a route change happens inside a known cloud prefix, the pulse appears near its actual physical footprint instead of a random corporate headquarters. It's vastly more accurate than relying on registry data alone.

### Managing the Firehose

BGP updates arrive continuously, and during route flapping events the volume spikes hard.

To keep the visualization readable without melting the screen, the pipeline filters out redundant updates (within 15 seconds), waits 10 seconds to ensure a withdrawal isn't just a rapid path re-convergence, and paces the visual output so spikes are emitted smoothly every 500ms using a logarithmic scale.

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

## Aesthetics, Motion, and Sound

With some data issues resolved, I could focus on making it look good.

Animations use interpolation instead of snapping to the next state. Country rankings slide into position. Percentages ease between values. Even small UI transitions are smoothed out. These details significantly improve the polish of the stream, but it is definitely a balancing act. Too much movement can distract from the visual effect of the map itself, so getting this right required some restraint.

The pulses are what actually bring the data to life. In the engine, each pulse is a simple generated glow texture. I add a bit of spatial jitter so concurrent events do not stack perfectly on top of each other, and I scale their sizes logarithmically so massive data spikes do not turn the map into a solid wall of color.

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

Here is the final result, which I've gazed at for far too long already:

{{< a href="https://www.youtube.com/channel/UCA9eO4Gt-Ua6lAEGzWQHQFA/live" target="_blank" >}}
{{< diagram caption="🔴 [sudorandom on youtube](https://www.youtube.com/channel/UCA9eO4Gt-Ua6lAEGzWQHQFA/live)" >}}
{{< image src="map-animation.webp" animate="true" width="600px" >}}
{{< /diagram >}}
{{< /a >}}

This project turned into a deeper dive into BGP than I expected. Watching routing updates happening live exposes patterns that are impossible to find with a static snapshot.

So please, toss the live stream on your TV, sit back, relax, and watch the Internet route the world's network traffic as you listen to relaxing lofi in the background.
