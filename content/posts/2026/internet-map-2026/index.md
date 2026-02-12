---
categories: ["article", "project", "dataviz"]
tags: ["dataviz", "internet-map", "internet", "networking", "fiber-optics", "map", "world", "infrastructure", "peeringdb", "leaflet", "javascript"]
keywords: ["interactive internet map", "live internet map", "internet infrastructure map", "internet visualized", "global internet map", "internet exchange map", "internet exchange point map", "internet map of the world", "ixp map", "internet node map", "internet hub locations", "submarine cable map"]
series: ["Internet Map"]
date: "2026-03-12T10:00:00Z"
description: "Mapping global internet infrastructure and routing dominance over time"
cover: "cover.png"
images: ["posts/internet-map-2026/cover.png"]
featured: true
linktitle: ""
title: "Visualizing the Internet (2026)"
slug: "internet-map-2026"
type: "posts"
canonical_url: https://kmcd.dev/posts/internet-map-2026
---

For the past few years, I’ve been trying to make the physical reality of the internet visible with my Internet Infrastructure Map. I update the map each year, but I don’t want to just pull the latest data and call it a day.

For the 2026 edition of the map, I didn’t just want to show where the cables are or where open peering exchanges are located. I wanted to answer a harder question: where does the internet actually live? By analyzing 15 years of BGP routing tables alongside physical infrastructure data, I’m closer to answering it. The result is a concept that I'm calling “Logical Dominance” of major cities around the world. In practical terms, Logical Dominance measures the percentage of the global routing table originated by networks in each city.

You can explore the map live at **[map.kmcd.dev](https://map.kmcd.dev)**.

{{< diagram >}}
<a href="https://map.kmcd.dev" target="_blank">
{{< image src="screenshot.png" alt="Map of the Internet" >}}
</a>
{{< /diagram >}}

---

### How the Internet Routes Traffic

Previous versions of the map focused on the physical infrastructure: the cables and exchange points. However, the physical path is only half the story. To understand how data moves, we have to look at **BGP (Border Gateway Protocol)**.

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

These routes are critical for the internet to function, but they change thousands of times per second.

#### Measuring Logical Dominance

In the 2026 edition of the map, I use BGP data to estimate the "Logical Dominance" of cities: essentially, how many IP addresses are "homed" in a particular location. My goal was to represent the percentage of the global routing table originated by networks operating in each city.

I built a series of scripts that reconstruct the internet’s “logical map” for every year from 2010 to 2026. This allows the map to visualize not just where the cables land, but where the *logical* weight of the internet resides based on real historical data.

#### Sources of BGP Data

To visualize this layer, we need access to routing tables. There are four main ways to get this data. I explored all of them, but for historical analysis, most aren't appropriate. I'm still going to talk about them here because... well, it's interesting!

#### Actually be a Router

The most direct method is to operate a router capable of speaking BGP and convince an ISP or network to "peer" with you. This involves setting up a BGP daemon (like BIRD or GoBGP), listening on TCP port 179, and handling the complex binary protocol. This requires significant trust, configuration, and usually your own Autonomous System Number (ASN). We aren't doing this today. I'm not [Jared Mauch](https://arstechnica.com/tech-policy/2022/08/man-who-built-isp-instead-of-paying-comcast-50k-expands-to-hundreds-of-homes/).

#### Query a Looking Glass

We can connect to public routers provided by research projects like the [University of Oregon Route Views](http://www.routeviews.org/). These allow you to telnet in (TCP port 23) and run standard CLI commands like `show ip bgp` to see exactly what a backbone router sees.

Here is a trivial Go program that automates this interaction, logging into a Route Views server to query the path to Google's DNS (`8.8.8.8`). This code will feel extremely familiar to network engineers who did shell automation before there were APIs for these things:

{{< details-md summary="view routes from routeviews.org" github_file="go/routeviews/main.go" >}}
{{% render-code file="go/routeviews/main.go" language="go" %}}
{{< /details-md >}}

Here's the resulting output:

{{< details-md summary="BGP routes for 8.8.8.8" github_file="go/routeviews/output.txt" >}}
{{% render-code file="go/routeviews/output.txt" language="shell" %}}
{{< /details-md >}}

The output tells us that every path eventually terminates at **AS [15169](https://www.peeringdb.com/asn/15169)**, Google's Autonomous System number. While many networks peer directly with Google, the routing table also shows transit paths. For example, AS [20130](https://www.peeringdb.com/asn/20130) reaches Google by sending traffic through Hurricane Electric ([6939](https://www.peeringdb.com/asn/6939)).

While this is perfect for debugging today's internet, it doesn't help much for my project because it lacks historical data. As much as I might want to, I can’t telnet into 2012 to check what the routing table looked like 14 years ago.

#### Subscribe to a Stream

For real-time visualization, we can use **RIPE RIS Live**. Instead of polling a router, this service aggregates BGP data from collectors around the world and streams it over a public WebSocket. With this, we can watch the internet "breathe" in real-time as routes are announced and withdrawn.

Here is a Go program that connects to the RIS Live firehose and prints out route announcements as they propagate across the globe:

{{< details-md summary="stream_bgp.go" github_file="go/stream_bgp/main.go" >}}
{{% render-code file="go/stream_bgp/main.go" language="go" %}}
{{< /details-md >}}

This gives you a view of the internet's routing logic changing in real-time, often thousands of times per second.

Here's example output:
{{< details-md summary="BGP route stream" github_file="go/stream_bgp/output.txt" >}}
{{% render-code file="go/stream_bgp/output.txt" language="shell" %}}
{{< /details-md >}}
 
The output above captures just one second of the global BGP firehose. Each line is a JSON-encoded `UPDATE` message that identifies the specific IP block being announced and the AS Path, which is the path that traffic must travel through to reach your final destination. The rapid-fire timestamps show the internet "breathing" in real-time as thousands of routers constantly negotiate the most efficient paths across the globe.

Again, this is fascinating for a live dashboard, but useless for backfilling history.

#### Download Historical Snapshots

Finally, for deep analysis and building the logical dominance model (with backfilled data), I was left with the only option: downloading and processing raw **RIB (Routing Information Base)** files. These are massive snapshots of the entire routing table as seen by a backbone router at a specific moment in time.

Because BGP is a "chatter" protocol that only announces changes, these full table dumps are essential for reconstructing the state of the internet at any point in the past. Several research projects maintain these archives for decades:

*   **[University of Oregon Route Views](http://archive.routeviews.org/):** The most comprehensive archive, with MRT-formatted dumps dating back to the late 1990s from collectors globally.
*   **[RIPE RIS (Routing Information Service)](https://www.ripe.net/analyse/internet-measurements/routing-information-service-ris/ris-raw-data/):** Provides high-fidelity snapshots from a dense network of collectors, primarily in Europe and the Middle East.
*   **[CAIDA BGP Stream](https://bgpstream.caida.org/):** A framework for analyzing both real-time and historical BGP data from various sources.

---

### Showing BGP on the map
For this map, I processed over 15 years of these snapshots to build the logical dominance model.

Reconstructing the historical "logical" map was the hardest part of the project. I wanted to push the timeline back further, but reliable archival data for physical peering effectively vanishes before 2010.

Even covering the last 15 years required building a translation layer to handle massive schema drift in the PeeringDB archives. My scripts normalize three distinct eras of data formats to get a consistent timeline:

- **2018–2026** (JSON): The modern standard.
- **2016–2018** (SQLite v2): A transitional format.
- **2010–2016** (SQLite v1): The "Legacy" era. This was the most difficult to process, as it used a completely different schema (e.g., peerParticipants) that had to be manually mapped to modern concepts.

By combining the peering data with the BGP routing tables from Route Views, the map can finally calculate "Logical Dominance" for the last 15 years.

#### What Changed When IP Dominance Was Added

When I layered IP dominance onto the physical map, many additional cities became visible.

In earlier versions, visibility depended heavily on registered Internet Exchange Points. That highlighted the traditional coastal hubs and major peering metros. But once routing table data was incorporated, additional cities began to “light up.” These are places with substantial address space and large originating networks, even if they do not host a major public exchange.

{{< figure src="usa.png" link="usa.png" alt="United States" attrlink="usa.png" description="United States on the map.">}}

The physical meeting points of networks only tell us a part of the story of Internet infrastructure. The global routing table reveals where address space is actually controlled and originated. Some cities carry significant logical weight without being major public peering hubs. The IP dominance layer exposes that distinction.

This effect, however, was not uniform. One of the most striking and baffling patterns on the map is just how much China is under-represented compared to its actual internet footprint.

{{< figure src="china.png" link="china.png" alt="China" attrlink="china.png" description="China on the map." caption="Note that Hong Kong is the city with the large peering presence, not mainland China.">}}

The Chinese internet is giant: it has massive cable landings, dense domestic fiber networks, and a scale of internal activity that rivals any other region. Yet, when you switch to the logical layer, much of that presence simply vanishes. It is as if a massive portion of the internet has chosen to stay in the shadows. This blackout is likely due to the Great Firewall's peering restrictions, creating a localized intranet that barely touches the public BGP table. So from the perspective of the global routing firehose, parts of the country appear remarkably dark.

### The Data

One of the biggest requests I've had in previous years is for access to the raw data behind the visualizations. For the 2026 edition, I have exposed the underlying JSON datasets that power the map. These files are curated from **TeleGeography** (for modern cables), **PeeringDB** (for IXPs), and historical data is curated from various sources including **submarinenetworks.com** and archived maps.

You can access these directly to build your own visualizations, analyze the growth of global bandwidth, or double check my numbers.

- [`all_cables.json`](https://map.kmcd.dev/data/all_cables.json): **The Core Map Data.** A GeoJSON FeatureCollection containing all submarine cables. Each feature includes properties like `name`, `rfs_year` (Ready for Service), `decommission_year`, `owners`, and `landing_points`. This follows the standard [GeoJSON format](https://geojson.org/).
- [`year-summaries.json`](https://map.kmcd.dev/data/year-summaries.json): Brief textual descriptions of notable events or milestones for specific years, displayed in the footer.
- [`city-dominance/{year}.json`](https://map.kmcd.dev/data/city-dominance/2026.json): Per-year JSON files (e.g., 2026.json) with detailed city-level peering capacity, regional information, and coordinates. Used for rendering city markers and calculating regional statistics.
- [`meta.json`](https://map.kmcd.dev/data/meta.json): Metadata including the minimum and maximum years covered by the visualization.

**[Explore the Map »](https://map.kmcd.dev)**
