---
categories: ["article", "project", "dataviz"]
tags: ["dataviz", "internet-map", "internet", "networking", "bgp", "fiber-optics", "map", "world", "infrastructure", "peeringdb", "leaflet", "javascript"]
keywords: ["interactive internet map", "live internet map", "internet infrastructure map", "internet visualized", "global internet map", "internet exchange map", "internet exchange point map", "internet map of the world", "ixp map", "internet node map", "internet hub locations", "submarine cable map"]
series: ["Internet Map"]
date: "2026-03-12T10:00:00Z"
description: "Mapping global internet infrastructure and routing dominance over time"
cover: "cover.svg"
images: ["posts/internet-map-2026/cover.svg"]
featured: true
linktitle: ""
title: "Visualizing the Internet (2026)"
slug: "internet-map-2026"
type: "posts"
canonical_url: https://kmcd.dev/posts/internet-map-2026
---

For the past few years, I’ve been trying to make the physical reality of the internet visible with my Internet Infrastructure Map. I update the map each year, but I don’t want to just pull the latest data and call it a day.

For the 2026 edition, I wanted to answer a harder question: where does the internet actually live? By analyzing 15 years of BGP routing tables alongside physical infrastructure data, I’m closer to answering it. 

The result is a concept I call “Logical Dominance.” In technical terms, a city’s dominance is calculated by summing the IPv4 address space originated by networks located there. By comparing these totals against the global routing table, we can see where the internet’s weight actually settles. To keep the comparison consistent across the 15-year timeline, I calculate these scores from a full RIB snapshot taken on February 1st of each year. While IPv6 is becoming increasingly critical, I’ve limited this analysis to IPv4 for now due to the higher consistency of historical attribution data across the full window and because IPv6 space is massive and will overshadow IPv4. Like it or not, IPv4 is still extremely relevant to the Internet, even in 2026.

In short: this shows which cities control the largest share of the internet’s reachable address space.

You can explore the map live at **[map.kmcd.dev](https://map.kmcd.dev)**.

{{< diagram >}}
{{< compare before="map_dark.svg" after="map_light.svg" caption="Internet Infrastructure Map (2026) @ **[map.kmcd.dev](https://map.kmcd.dev)**" >}}
{{< /diagram >}}

### How the Internet Routes Traffic

Previous versions of the map focused on physical infrastructure: cables and exchange points. However, the physical path is only half the story. To understand how data moves, we have to look at **BGP (Border Gateway Protocol)**.

BGP is the protocol that distinct networks, known as **Autonomous Systems (AS)**, use to announce which IP addresses they own and how to reach them. If the cables are the hardware, BGP is the software that holds the internet together. For a deeper dive, [Cloudflare has an excellent primer](https://www.cloudflare.com/learning/security/glossary/what-is-bgp/).

When you load a webpage, your request doesn't just "know" the path. Your ISP’s routers consult the global BGP routing table to decide the best next hop. Visualized, it looks a little bit like this:

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
Observer: Router {
  shape: cylinder
}

# The Destination
Destination: Google\nAS15169\n(8.8.8.0/24) {
  shape: cloud
}

# --- Extracting Paths from your Log ---

# Path: 8283 15169 (Best)
AS8283: Netstream\nAS8283 {class: bgp_peer; style: {stroke: "#2ecc71"; stroke-width: 2}}
Observer -> AS8283: "Best >"
AS8283 -> Destination

# Path: 7018 15169 (AT&T)
AS7018: AT&T\nAS7018 {class: bgp_peer}
Observer -> AS7018
AS7018 -> Destination

# Path: 20130 6939 15169 (Via Hurricane Electric)
AS20130: AS20130 {class: bgp_peer}
HE: Hurricane Electric\nAS6939
Observer -> AS20130
AS20130 -> HE
HE -> Destination

# Path: 3333 1257 15169 (RIPE -> Tele2 -> Google)
AS3333: RIPE NCC\nAS3333 {class: bgp_peer}
Tele2: Tele2\nAS1257
Observer -> AS3333
AS3333 -> Tele2
Tele2 -> Destination

# Path: 49788 12552 15169
AS49788: AS49788 {class: bgp_peer}
AS12552: AS12552
Observer -> AS49788
AS49788 -> AS12552
AS12552 -> Destination
{{< /d2 >}}

These routes change thousands of times per second.

#### Sources of BGP Data

To visualize this layer, we need access to routing tables. I explored four ways to get this data, each with its own trade-offs between real-time visibility and historical context.

#### Query a Looking Glass

We can connect to public routers via projects like [University of Oregon Route Views](http://www.routeviews.org/). These allow you to telnet in and run standard CLI commands like `show ip bgp` to see exactly what a backbone router sees. 

{{< details-md summary="BGP routes for 8.8.8.8" github_file="routeviews-log.txt" >}}
{{% render-code file="routeviews-log.txt" language="shell" %}}
{{< /details-md >}}

Crucially, these paths often carry metadata called **BGP Communities**. These are optional tags that networks use to signal things like geographic origin or peering policy. While perfect for debugging today's internet, it lacks historical context; you can’t telnet into 2012 to check a routing table from 14 years ago.

#### Subscribe to a Stream

For real-time views, services like **RIPE RIS Live** aggregate BGP data from global collectors and stream it over a public WebSocket. You can watch the internet "breathe" as routes are announced and withdrawn thousands of times per second. This is fascinating for a live dashboard, but useless for backfilling history.

Here's an example script consuming this stream:
{{< details-md summary="go/stream_bgp/main.go" github_file="go/stream_bgp/main.go" >}}
{{% render-code file="go/stream_bgp/main.go" language="shell" %}}
{{< /details-md >}}

The output looks like this:
{{< details-md summary="Websocket Stream Output" github_file="go/stream_bgp/output.txt" >}}
{{% render-code file="go/stream_bgp/output.txt" language="shell" %}}
{{< /details-md >}}

#### Download Historical Snapshots

To build the historical model, I processed raw **RIB (Routing Information Base)** files. These are snapshots of the entire routing table as seen by a backbone router at a specific moment in time. Because BGP is a "chatter" protocol that only announces changes, these full table dumps are essential for reconstructing the state of the internet at any point in the past. 

I specifically fetch snapshots from February 1st at 12:00 UTC for every year in my timeline. To ensure a comprehensive view, I aggregate data from multiple global collectors maintained by the [University of Oregon Route Views](http://archive.routeviews.org/) project.

Other excellent resources for this kind of data include:

*   **[RIPE RIS (Routing Information Service)](https://www.ripe.net/analyse/internet-measurements/routing-information-service-ris/ris-raw-data/):** Provides high-fidelity snapshots from a dense network of collectors, primarily in Europe.
*   **[CAIDA BGP Stream](https://bgpstream.caida.org/):** A framework for analyzing both real-time and historical data from various sources.

---

### Showing BGP on the Map

For this edition, I processed over 15 years of BGP snapshots and PeeringDB archives to build the logical dominance model. Reconstructing this history was easily the hardest part of the project. I quickly realized that reliable archival data for physical peering effectively vanishes before 2010, which set a hard limit on how far back I could take the timeline.

#### Finding the Truth in the Noise

Mapping a BGP prefix to a specific city is notoriously difficult. A range might be registered to a corporate headquarters but serve users thousands of miles away. My solution uses prioritized attribution logic to resolve prefixes based on the highest-fidelity data available.

I start with high-quality [**Geofeeds (RFC 8805)**](https://datatracker.ietf.org/doc/html/rfc8805), where network operators explicitly self-report their locations. When those aren't available, I look for **Cloud Provider Ranges**. Major providers like [AWS](https://docs.aws.amazon.com/general/latest/gr/aws-ip-ranges.html) and [Google Cloud](https://docs.cloud.google.com/compute/docs/faq#find_ip_range) publish JSON feeds of their active IP ranges. I integrated these feeds and built a mapping layer to tie their logical regions to physical "home" cities—mapping ranges in `eu-west-1` to Dublin or `us-east-1` to Ashburn.

When those fail, I look at the network itself. I can map IPs to the city of the **IXP Next-Hop** where they are announced, or parse **BGP Communities** for geographical hints. My final fallback is a year-aware **WHOIS** ingestion. This tool now automatically switches data sources based on the year. For historical snapshots, it uses RIR delegation statistics appropriate to that year. For current data, it parses the full [APNIC database](https://www.apnic.net/manage-ip/using-whois/) for high-resolution city mapping.

I even had to add some safety checks to prevent "IP swallowing." For instance, there's a massive `0.0.0.0/0` block often pinned to Australia in the APNIC database. Without filtering for broad prefixes (anything with a mask length < 8), that one entry would incorrectly claim the entire global IP space for AU.

To handle any address space that remains unattributed, I duplicate those IPs across every city where the network maintains a physical peering presence. Even if a network doesn't announce every prefix from every point due to paid transit or internal long-haul links, this approach ensures that major connectivity hubs are credited for the logical weight they represent in the routing topology.

So... to recap, the data sources used for the 2026 map include:
- **Infrastructure:** TeleGeography, submarinenetworks.com, and historical archive maps.
- **Peering:** PeeringDB.
- **BGP Routing:** University of Oregon Route Views historical RIB archives.
- **IP Attribution:** RFC 8805 Geofeeds, AWS/Google Cloud IP ranges, BGP Communities, APNIC WHOIS database and historical RIR delegation statistics.

When making this pipeline, I've had many engineering challenges, but here are the ones worth mentioning:

#### The Local Cache

Downloading 15 years of archives is slow. I threw together a quick file-based cache to avoid hitting the network repeatedly. It was the simplest code I wrote but easily the most valuable, turning 30-minute download waits into near-instant local reads.

#### RAM remains stubbornly finite

Loading millions of IP prefixes, WHOIS records, PeeringDB entries and their associated metadata into a standard in-memory map consumes gigabytes of RAM instantly. Frustratingly, my laptop only has so much. To avoid out of memory errors I built a custom **on-disk Trie data structure** using [**BadgerDB v4**](https://github.com/dgraph-io/badger). I might show it off in a later blog post after I clean it up a little bit. By using IP prefixes as keys in a sorted KV store, I can perform efficient longest-prefix matching directly against the disk.

#### From Spaghetti to Pipeline

While investigating all of these different data sources, I ended up writing several programs that generated output of different shapes that would be used by other programs. It all made sense to me at the time but it spiraled out of control into a confusing mess. However, now I have one script for generating this city data. This was definitely enabled by some of the improvements, like caching and using on-disk data structures to make memory usage reasonable.


### What Changed When IP Dominance Was Added

When I layered IP dominance onto the physical map, many additional cities became visible.

In earlier versions, visibility depended heavily on registered Internet Exchange Points. That highlighted the traditional coastal hubs and major peering metros. But once routing table data was incorporated, additional cities began to “light up.” These are places with substantial address space and large originating networks, even if they do not host a major public exchange.

{{< compare before="us_before.svg" after="us_after.svg" caption="United States on the map (before and after)." >}}

The physical meeting points of networks only tell us a part of the story. The global routing table reveals where address space is actually controlled and originated. Some cities carry significant weight without being major public peering hubs. The IP dominance layer exposes that distinction.

{{< compare before="eu_before.svg" after="eu_after.svg" caption="Europe on the map (before and after)." >}}

This effect, however, was not uniform. One of the most striking patterns on the map is just how much China is under-represented in global routing tables relative to its actual footprint.

{{< compare before="cn_before.svg" after="cn_after.svg" caption="China on the map (before and after). Note how the 'before' view over-attributes weight to Hong Kong, while the 'after' view distributes it to mainland hubs." >}}

The Chinese internet is giant, but it presents a unique attribution challenge. Because so much of China’s domestic routing remains internal to national carriers, the global BGP "firehose" often only sees these massive networks when they peer at international hubs like Hong Kong, Los Angeles, or Frankfurt.

Initially, this caused a "flooding" effect: because I attribute IPs to the cities where they are announced, a single China Telecom node in Hong Kong would suddenly appear to "own" a staggering percentage of the global internet. To fix this, I had to implement specific logic for China-based networks. I used pattern matching to parse provincial hints and city names from the APNIC WHOIS database—mapping "CHINANET-BJ" to Beijing or "CHINANET-GD" to Guangdong—and then distributed the logical weight of these massive blocks across major domestic hubs. This prevents a few international peering points from unfairly skewing the map and provides a much more accurate (if still technically "logical") view of where that weight actually lives.


#### Case Study: Frankfurt’s "Network Gravity"

Frankfurt is the standout example in this new model. By 2026, it has solidified its position as the #1 city in the world by logical dominance, accounting for over **858 million IPs**.

This reveals a fascinating "center of mass" effect: while **Amsterdam** still holds the crown for raw physical peering bandwidth (244.63 Tbps vs Frankfurt's 200.75 Tbps), Frankfurt wins on logical weight. It acts as the primary intersection where Western hyperscale clouds meet East Asian transit networks, pulling the "logical" center of the internet toward the heart of Europe.

### UX and Rendering

Layering BGP data onto an already complex physical map created a major design challenge: **density**. With hundreds of new cities "lighting up" globally, the map became significantly cluttered when zoomed out.

To solve this, I implemented **Dynamic Cluster Grouping**. Close-by cities now group together into aggregate hubs at low zoom levels, which then split into individual markers as you dive deeper. This isn't just a visual fix; by reducing the number of active SVG shapes in the DOM, it significantly improves panning performance on mobile devices.

I also introduced **Viewport Culling**. The map now only renders assets currently within your bounds. As you pan to a new region, cities "pop in" dynamically, ensuring the browser isn't wasting resources on rendering things on the other side of the planet.

The visual size of cities on the map also now dynamically reflects their importance. Previously, cities were sized based on their relative peering bandwidth. Now, their size depends on a weighted combination of aggregate peering bandwidth and IP dominance, contributing 80% and 20% to the size calculation respectively. The reasoning behind this is that peering bandwidth is much more of a signal that there's more Internet activity than IP space being advertised.

### Better Exports

One of the most requested features for the map has been a way to export the current view for use in presentations, reports, posters, or just as a high-quality wallpaper.

Previously, I was using a standard Leaflet plugin for this, but it was not great. It would often fail in weird ways, leaving you with a glitched or incomplete rendering of the map. Also, it exported as a PNG, which meant the beautiful vector data of the cables and cities was flattened into a low-resolution raster format.

Now there's a new download button that renders an isolated SVG. Because the map itself is built on SVGs, this new export method is lossless. It respects your current zoom level and position, allowing you to focus on a specific region and generate an incredibly high-quality vector file that you can scale to any size without losing a single pixel of detail. All of the images above used this export!

### The Data

Another one of the biggest requests I've had in previous years is for access to the raw data behind the visualizations. For the 2026 edition, I have exposed the underlying JSON datasets that power the map. These files are curated from **TeleGeography** (for modern cables), **PeeringDB** (for IXPs), and historical data is curated from various sources including **submarinenetworks.com** and archived maps.

You can access these directly to build your own visualizations, analyze the growth of global bandwidth, or double check my numbers.

- [`all_cables.json`](https://map.kmcd.dev/data/all_cables.json): **The Core Map Data.** A GeoJSON FeatureCollection containing all submarine cables. Each feature includes properties like `name`, `rfs_year` (Ready for Service), `decommission_year`, `owners`, and `landing_points`. This follows the standard [GeoJSON format](https://geojson.org/).
- [`year-summaries.json`](https://map.kmcd.dev/data/year-summaries.json): Brief textual descriptions of notable events or milestones for specific years, displayed in the footer.
- [`city-dominance/{year}.json`](https://map.kmcd.dev/data/city-dominance/2026.json): Per-year JSON files (e.g., 2026.json) with detailed city-level peering capacity, regional information, and coordinates. Used for rendering city markers and calculating regional statistics.
- [`meta.json`](https://map.kmcd.dev/data/meta.json): Metadata including the minimum and maximum years covered by the visualization.

---

**[Explore the Map »](https://map.kmcd.dev)**
