---
categories: ["article", "project", "dataviz"]
tags: ["dataviz", "internet-map", "internet", "networking", "bgp", "fiber-optics", "map", "world", "infrastructure", "peeringdb", "leaflet", "javascript"]
keywords: ["interactive internet map", "live internet map", "internet infrastructure map", "internet visualized", "global internet map", "internet exchange map", "internet exchange point map", "internet map of the world", "ixp map", "internet node map", "internet hub locations", "submarine cable map"]
series: ["Internet Map"]
date: "2026-02-18T10:00:00Z"
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

For the past few years, I’ve been trying to make the physical reality of the internet visible with my Internet Infrastructure Map. I update the map each year, but I don’t want to just pull the latest data and call it a day. In this post I discuss how the map changed for this year and what I did to make it happen, but you can skip to the good part by viewing it here: **[map.kmcd.dev](https://map.kmcd.dev)**.

For the 2026 edition, I wanted to answer a harder question: where does the internet actually live? By analyzing 15 years of BGP routing tables alongside physical infrastructure data, I’m now closer to answering that question.

The result is a concept I call "Logical Dominance." Each city’s dominance is calculated by summing the number of IPv4 addresses that are "homed" in that city. How can I tell where IP addresses are homed? Well, read on, because I talk about this quite a bit in a later section.

{{< diagram >}}
{{< compare before="map_dark.svg" after="map_light.svg" caption="Internet Infrastructure Map (2026) @ **[map.kmcd.dev](https://map.kmcd.dev)**" >}}
{{< /diagram >}}

### How the Internet Routes Traffic

Previous versions of the map focused on physical infrastructure: cables and exchange points. However, the physical path is only half the story. To understand how data moves, we have to look at **BGP (Border Gateway Protocol)**.

BGP is the protocol that distinct networks, known as **Autonomous Systems (AS)**, use to announce which IP addresses they own and how to reach them. If the cables are the hardware, BGP is the software that ties the Internet together. For a deeper dive, [Cloudflare has an excellent primer](https://www.cloudflare.com/learning/security/glossary/what-is-bgp/).

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
AS20130: DePaul University\nAS20130 {class: bgp_peer}
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
AS49788: Nexthop AS\nAS49788 {class: bgp_peer}
AS12552: GlobalConnect\nAS12552
Observer -> AS49788
AS49788 -> AS12552
AS12552 -> Destination
{{< /d2 >}}

These routes change thousands of times per second, constantly reshaping the internet’s topology.

#### Sources of BGP Data

To visualize this layer, we need access to routing tables. I explored three ways to get this data, each with its own trade-offs between real-time visibility and historical context.

#### Query a Looking Glass

We can connect to public routers via projects like [University of Oregon Route Views](http://www.routeviews.org/). These allow you to telnet in and run standard CLI commands like `show ip bgp` to see exactly what a backbone router sees. 

{{< details-md summary="BGP routes for 8.8.8.8" github_file="routeviews-log.txt" >}}
{{% render-code file="routeviews-log.txt" language="shell" %}}
{{< /details-md >}}

Crucially, these paths often carry metadata called **BGP Communities**. These are optional tags that networks use to signal things like geographic origin or peering policy. While perfect for debugging today’s internet, this approach lacks historical context; you can’t telnet into 2012 to check a routing table from 14 years ago.

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

I specifically fetched snapshots from February 1st at 12:00 UTC for every year in my timeline. To ensure a comprehensive view, I aggregated data from multiple global collectors maintained by the [University of Oregon Route Views](http://archive.routeviews.org/) project.

Other excellent resources for this kind of data include:

*   **[RIPE RIS (Routing Information Service)](https://www.ripe.net/analyse/internet-measurements/routing-information-service-ris/ris-raw-data/):** Provides high-fidelity snapshots from a dense network of collectors, primarily in Europe.
*   **[CAIDA BGP Stream](https://bgpstream.caida.org/):** A framework for analyzing both real-time and historical data from various sources.

---

### Showing BGP on the Map

For this edition, I processed over 15 years of BGP snapshots and PeeringDB archives to build the Logical Dominance model. Reconstructing this history was easily the hardest part of the project. I quickly realized that reliable archival data for physical peering effectively vanishes before 2010, which set a hard limit on how far back I could take the timeline.

#### IPv4
Earlier, I mentioned that I’m only looking at IPv4. You might be asking why I avoided IPv6.

While IPv6 is critical, I’ve excluded it for now because its sheer scale breaks the "dominance" calculation. I measured dominance by counting unique IP addresses, and IPv6 is simply too vast to compare 1:1 with IPv4.

Consider this: The smallest standard IPv6 assignment is a `/64`. That single subnet contains `18,446,744,073,709,551,616` addresses. You could fit the entire global IPv4 routing table inside that one subnet **4.3 billion times over**.

If I treated every IP equally, a single home router with IPv6 would statistically obliterate a city hosting the entire legacy IPv4 internet.

- Total IPv6 Addresses: `340,282,366,920,938,463,463,374,607,431,768,211,456`
- Total IPv4 Addresses: `4,294,967,296`

Like it or not, IPv4 remains foundational for the Internet's geography. Maybe next year I can tackle the normalization problem, but not today!

#### Finding the Truth in the Noise

Mapping a BGP prefix to a specific city is not as straightforward as you might think. A range might be registered to a corporate headquarters but serve users thousands of miles away. My solution used prioritized attribution logic to resolve prefixes based on the highest-fidelity data available.

I started with high-quality [**Geofeeds (RFC 8805)**](https://datatracker.ietf.org/doc/html/rfc8805), where network operators explicitly self-report their locations. When those weren't available, I looked for **Cloud Provider Ranges**. Major providers like [AWS](https://docs.aws.amazon.com/general/latest/gr/aws-ip-ranges.html) and [Google Cloud](https://docs.cloud.google.com/compute/docs/faq#find_ip_range) publish JSON feeds of their active IP ranges. I integrated these feeds and built a mapping layer to tie their logical regions to physical "home" cities—mapping ranges in `eu-west-1` to Dublin or `us-east-1` to Ashburn.

When a prefix didn't match any of those sources, I looked at the network itself. I could use the **IXP Next-Hop** where they are announced, or parse **BGP Communities** for geographical hints. If THAT fell short, my final fallback leveraged historical **WHOIS** backups.

To handle address space that remained unattributed to a specific city, I applied a 'footprint' heuristic which assigned those IPs to every city where the network maintained a physical peering presence. While a network might not literally announce every prefix at every IXP, this approach ensures that major connectivity hubs were credited for the logical weight they are capable of serving.

There were many issues that I stumbled headfirst into when trying to attribute certain prefixes. For example, I had to add some safety checks to prevent "IP swallowing." For instance, there's a massive `0.0.0.0/0` block often pinned to Australia in the APNIC database. The `0.0.0.0/0` prefix would match *every single IPv4 address*. Without filtering for broad prefixes (anything with a mask length < 8), that one entry would incorrectly claim the entire global IP space for Australia.

So... to recap, the data sources used for the 2026 map include:
- **Infrastructure:** [TeleGeography](https://www2.telegeography.com/), [submarinenetworks.com](https://www.submarinenetworks.com/), and historical archive maps.
- **Peering:** [PeeringDB](https://www.peeringdb.com/).
- **BGP Routing:** [University of Oregon Route Views historical RIB archives](https://www.routeviews.org/routeviews/).
- **IP Attribution:** [RFC 8805 Geofeeds](https://datatracker.ietf.org/doc/html/rfc8805), [AWS](https://docs.aws.amazon.com/vpc/latest/userguide/aws-ip-ranges.html)/[Google Cloud](https://docs.cloud.google.com/vpc/docs/ip-addresses) IP ranges, BGP Communities, and [APNIC WHOIS database](https://www.apnic.net/about-apnic/whois_search/)

Building this pipeline presented unique engineering hurdles; here are the most significant ones:

#### The Local Cache

Downloading 15 years of archives is slow. I threw together a quick file-based cache to avoid hitting the network repeatedly. It was the simplest code I wrote but easily the most valuable, turning 30-minute download waits into near-instant local reads.

#### RAM remains stubbornly finite

Loading millions of IP prefixes, WHOIS records, PeeringDB entries, and their associated metadata into a standard in-memory map consumes gigabytes of RAM instantly. Frustratingly, my laptop only has so much. To avoid out-of-memory errors I built a custom **on-disk [trie data structure](https://en.wikipedia.org/wiki/Trie)** using [**BadgerDB v4**](https://github.com/dgraph-io/badger). I might show it off in a later blog post after I clean it up a little bit. By using IP prefixes as keys in a sorted KV store, I can perform efficient longest-prefix matching directly against the disk.

#### Cleaning up the spaghetti

While investigating all of these different data sources, I ended up writing several programs that generated output of different shapes that would be used by other programs. It all made sense to me at the time but it spiraled out of control into a confusing mess. Now, I have one script for generating this city data. I was only able to do this because of the improvements mentioned above: caching and using on-disk data structures. Now, the script has clear stages of:

- **Fetch:** Downloads and caches raw data (WHOIS, BGP, PeeringDB).
- **Index:** Builds searchable on-disk tries and resolves authoritative network names from RIRs.
- **Process:** Scans BGP routes and attributes each prefix using the various data sources mentioned above.
- **Output:** Produces clean, normalized city results without duplicate entries (e.g., merging "Seoul" and "SEOUL").

### What Changed When IP Dominance Was Added

When I layered IP dominance onto the physical map, many additional cities became visible.

{{< compare before="map_2026_before.svg" after="map_2026.svg" caption="World (before and after adding BGP data)." >}}

In earlier versions, visibility depended heavily on registered Internet Exchange Points. That highlighted the traditional coastal hubs and major peering metros. But once routing table data was incorporated, the map revealed cities without major IXPs. These are places with substantial address space and large originating networks, even if they do not host a major public exchange. This is most noticeable in India, Japan, China, Indonesia, and in secondary metros beyond traditional hubs in the EU and United States.

{{< compare before="us_before.svg" after="us_after.svg" caption="United States on the map (before and after adding BGP data)." >}}

The physical meeting points of networks only tell us a part of the story. The global routing table reveals where address space is actually controlled and originated. Some cities carry significant weight without being major public peering hubs. The IP dominance layer exposes that distinction.

{{< compare before="eu_before.svg" after="eu_after.svg" caption="Europe on the map (before and after adding BGP data)." >}}

The Chinese internet is giant, but it presents a unique attribution challenge. Because so much of China’s domestic routing remains internal to national carriers, the global BGP table often only sees these massive networks when they peer at international hubs like Hong Kong, Los Angeles, or Frankfurt. An earlier version of my attribution code ended up adding all of China's IP space to these select few international hubs, which was clearly incorrect. It looked like China Telecom was the biggest ISP in Germany, which made it appear that China Telecom dominated Germany. It does not, at least not yet. To fix this, I implemented specific logic for China-based networks. I used pattern matching to parse provincial hints from APNIC WHOIS data. This mapped prefixes like `GD` or `SH` to their respective provincial capitals. I also linked ASNs to their parent organizations in PeeringDB to prevent Chinese networks from being misattributed to foreign exchange points. This resolved attribution for the vast majority of prefixes. Any remaining IP space attributed only at the country level is distributed across major domestic hubs.

{{< compare before="cn_before.svg" after="cn_after.svg" caption="China on the map (before and after adding BGP data)." >}}

The result is a far more realistic view of China’s internal internet topology.

### UX and Rendering

Layering BGP data onto an already complex physical map created a major design challenge: **density**. With hundreds of new cities "lighting up" globally, the map became significantly cluttered when zoomed out.

To solve this, I implemented **Dynamic Cluster Grouping**. Close-by cities now group together into aggregate hubs at low zoom levels, which then split into individual markers as you dive deeper. This isn't just a visual fix; by reducing the number of active SVG shapes in the DOM, it significantly improves panning performance on mobile devices.

{{< diagram >}}
{{< compare before="clustering_no.png" after="clustering_yes.png" caption="Before and after dynamic cluster groupings." >}}
{{< /diagram >}}

Dynamic Cluster Grouping ensures the map remains legible, preventing the increased data density from overwhelming the map. When you click on a cluster, the details panel expands to list every city contained within that group.

{{< diagram >}}{{< image src="group-screenshot.png" class="center" width="500px"  >}}{{< /diagram >}}

I also introduced **Viewport Culling**. The map now only renders assets currently within your bounds. As you pan to a new region, cities "pop in" dynamically, ensuring the browser isn't wasting resources on rendering things on the other side of the planet.

The visual size of cities on the map also now dynamically reflects their importance. Previously, cities were sized based on their relative peering bandwidth. Now, their size depends on a weighted combination of aggregate peering bandwidth and IP dominance, contributing 80% and 20% to the size calculation respectively. Peering bandwidth is a stronger signal of real traffic concentration than raw IP space alone.

### Better Exports

One of the most requested features for the map has been a way to export the current view for use in presentations, reports, posters, or just as a high-quality wallpaper.

Previously, I was using a standard Leaflet plugin for this, but it was not great. It would often fail in weird ways, leaving you with a glitched or incomplete rendering of the map. Also, it exported as a PNG, which meant the beautiful vector data of the cables and cities was flattened into a low-resolution raster format.

Now there's a new download button that renders an isolated SVG. Because the map itself is built on SVGs, this new export method is lossless. It respects your current zoom level and position, allowing you to focus on a specific region and generate an incredibly high-quality vector file that you can scale to any size without losing a single pixel of detail. All images in this post were generated using this new export feature.

### The Data

Another one of the biggest requests I've had in previous years is for access to the raw data behind the visualizations. For the 2026 edition, I have exposed the underlying JSON datasets that power the map. These files are curated from **TeleGeography** (for modern cables), **PeeringDB** (for IXPs), and historical data is curated from various sources including **submarinenetworks.com** and archived maps.

You can access these directly to build your own visualizations, analyze the growth of global bandwidth, or double check my numbers.

- [`all_cables.json`](https://map.kmcd.dev/data/all_cables.json): **The Core Map Data.** A GeoJSON FeatureCollection containing all submarine cables. Each feature includes properties like `name`, `rfs_year` (Ready for Service), `decommission_year`, `owners`, and `landing_points`. This follows the standard [GeoJSON format](https://geojson.org/).
- [`year-summaries.json`](https://map.kmcd.dev/data/year-summaries.json): Brief textual descriptions of notable events or milestones for specific years, displayed in the footer.
- [`city-dominance/{year}.json`](https://map.kmcd.dev/data/city-dominance/2026.json): Per-year JSON files (e.g., 2026.json) with detailed city-level peering capacity, regional information, and coordinates. Used for rendering city markers and calculating regional statistics.
- [`meta.json`](https://map.kmcd.dev/data/meta.json): Metadata including the minimum and maximum years covered by the visualization.

---

You might ask why I burned so much time manually attributing IP space when services like [MaxMind](https://www.maxmind.com) or [IPInfo](https://ipinfo.io/) already exist. The honest answer? Buying the data isn't fun. The joy of this project comes from the archaeology and the work involved in bringing order to chaotic and disjointed datasets and transforming them into something beautiful.

This was a great project, and I am extremely happy with the results. If you've gotten this far without checking out the map, I'm impressed with your restraint, but here's one more link for you to take a look:

**[Explore the Map »](https://map.kmcd.dev)**
