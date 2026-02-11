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

For the past few years, I’ve been trying to make the physical reality of the internet visible. When updating the map I don't want to just pull the latest data and call it a day. I wanted to add more insight or, frankly, accuracy to the map.

For the 2026 edition of the map, I didn’t just want to show where the cables are or where open peering exchanges happen to be successful. I wanted to answer a harder question: where does the internet actually live? By analyzing 15 years of BGP routing tables alongside physical infrastructure data, I modeled the “Logical Dominance” of major cities around the world.

You can explore the map live at **[map.kmcd.dev](https://map.kmcd.dev)**.

---

### How the Internet Routes Traffic

Previous versions of the map focused on the physical infrastructure—the cables and exchange points. However, the physical path is only half the story. To understand how data moves, we have to look at **BGP (Border Gateway Protocol)**.

BGP is the protocol that distinct networks, known as **Autonomous Systems (AS)**, use to announce which IP addresses they own and how to reach them. If the cables are the hardware, BGP is the software that holds the internet together. For a deeper dive, [Cloudflare has an excellent primer](https://www.cloudflare.com/learning/security/glossary/what-is-bgp/).

When you load a webpage, your request doesn't just "know" the path. Your ISP’s routers consult the global BGP routing table to decide the best next hop.

#### Measuring Logical Dominance

In the 2026 edition of the map, I've used BGP data to estimate the "Logical Dominance" of cities: essentially, how many IP addresses are "homed" in a particular location. It represents the percentage of the global routing table originated by networks operating in that city.

I have a series of scripts that reconstructs the internet's "logical map" for every year between 2010 and 2026. This allows the map to visualize not just where the cables land, but where the *logical* weight of the internet resides based on real historical data.

#### Sources of BGP Data

To visualize this layer, we need access to routing tables. There are essentially four ways to get this data. I explored all of them, but for historical analysis, most aren't appropriate. I'm still going to talk about them here because... well, it's interesting!

#### Actually be a Router

The most direct method is to operate a router capable of speaking BGP and convince an ISP or network to "peer" with you. This involves setting up a BGP daemon (like BIRD or GoBGP), listening on TCP port 179, and handling the complex binary protocol. This requires significant trust, configuration, and usually your own Autonomous System Number (ASN). We aren't doing this today. That requires trust, coordination, and your own ASN, things I don't have.

#### Query a Looking Glass

We can connect to public routers provided by research projects like the [University of Oregon Route Views](http://www.routeviews.org/). These allow you to telnet in (TCP port 23) and run standard CLI commands like `show ip bgp` to see exactly what a backbone router sees.

Here is a trivial Go program that automates this interaction, logging into a Route Views server to query the path to Google's DNS (`8.8.8.8`). This code will feel extremely familiar to network engineers that did shell automation before there were APIs:

{{< details-md summary="view routes from routeviews.org" github_file="go/routeviews/main.go" >}}
{{% render-code file="go/routeviews/main.go" language="go" %}}
{{< /details-md >}}

This snippet connects to Route Views and dumps the path to `8.8.8.8`.

Here's the resulting output:

{{< details-md summary="BGP routes for 8.8.8.8" github_file="go/routeviews/output.txt" >}}
{{% render-code file="go/routeviews/output.txt" language="shell" %}}
{{< /details-md >}}

The output reveals that every path eventually terminates at **AS [15169](https://www.peeringdb.com/asn/15169)**, Google's Autonomous System number. While many networks peer directly with Google, the routing table also shows transit paths—for example, AS [20130](https://www.peeringdb.com/asn/20130) reaching Google by sending traffic through Hurricane Electric ([6939](https://www.peeringdb.com/asn/6939)).

While this is perfect for debugging today's internet, it doesn't help too much for my project because it lacks historical data. As much as I might want, I can't telnet into 2012 to check the status 14 years ago.

#### Subscribe to a Stream

For real-time visualization, we can use **RIPE RIS Live**. Instead of polling a router, this service aggregates BGP data from collectors around the world and streams it over a public WebSocket. This allows us to watch the internet "breathe" in real-time as routes are announced and withdrawn.

Here is a Go program that connects to the RIS Live firehose and prints out route announcements as they propagate across the globe:

{{< details-md summary="stream_bgp.go" github_file="go/stream_bgp/main.go" >}}
{{% render-code file="go/stream_bgp/main.go" language="go" %}}
{{< /details-md >}}

The script connects to the RIPE RIS Live firehose via WebSocket, subscribes to real-time `UPDATE` messages, and continuously streams every new route advertisement it receives across the globe.

This gives you a view of the internet's routing logic changing in real-time, often hundreds of times per second.

Here's example output:
{{< details-md summary="BGP route stream" github_file="go/stream_bgp/output.txt" >}}
{{% render-code file="go/stream_bgp/output.txt" language="shell" %}}
{{< /details-md >}}
 
The output above captures just one second of the global BGP firehose. Each line is a JSON-encoded `UPDATE` message that identifies the specific IP block being announced and the literal "map" of the internet—the AS Path—that traffic must travel through to reach it. The rapid-fire timestamps show the internet "breathing" in real-time as thousands of routers constantly negotiate the most efficient paths across the globe.

Again, this is fascinating for a live dashboard, but useless for historical analysis and data backfill.

#### Download Historical Snapshots

Finally, for deep analysis and building the logical dominance model (with backfilled data), I was left with the only remaining option: downloading and processing raw **RIB (Routing Information Base)** files. These are massive snapshots of the entire routing table as seen by a backbone router at a specific moment in time. 

Because BGP is a "chatter" protocol that only announces changes, these full table dumps are essential for reconstructing the state of the internet at any point in the past. Several research projects maintain these archives for decades:

*   **[University of Oregon Route Views](http://archive.routeviews.org/):** The most comprehensive archive, with MRT-formatted dumps dating back to the late 1990s from collectors globally.
*   **[RIPE RIS (Routing Information Service)](https://www.ripe.net/analyse/internet-measurements/routing-information-service-ris/ris-raw-data/):** Provides high-fidelity snapshots from a dense network of collectors, primarily in Europe and the Middle East.
*   **[CAIDA BGP Stream](https://bgpstream.caida.org/):** A framework for analyzing both real-time and historical BGP data from various sources.

---

### Showing BGP on the map
For this map, I processed over 15 years of these snapshots to build the logical dominance model.

Reconstructing the historical “logical” map was the hardest part of the project. I wanted to go back further, but the timeline effectively stops at 2010. Before then, the archival data for where networks physically peered was harder to find, so the cutoff point for this data on the map ended up being 2010.

Even covering the last 15 years required building a translation layer to handle massive schema drift in the PeeringDB archives. The pipeline has to dynamically normalize three distinct eras of data just to get a consistent timeline:

- **2018–2026** (JSON): The modern standard.
- **2016–2018** (SQLite v2): A transitional format.
- **2010–2016** (SQLite v1): The "Legacy" era. This was the most difficult to process, as it used a completely different schema (e.g., peerParticipants) that had to be manually mapped to modern concepts.

By unifying these disparate formats with the BGP routing tables from Route Views, the map can finally calculate "Logical Dominance" based on historical fact rather than projection.

#### What Changed When IP Dominance Was Added

When I layered IP dominance onto the physical map, many additional cities became more prominent.

In earlier versions, visibility depended heavily on submarine cable landings and registered Internet Exchange Points. That highlighted the traditional coastal hubs and major peering metros. But once routing table data was incorporated, additional cities began to “light up.” These are places with substantial address space and large originating networks, even if they do not host a major public exchange.

{{< figure src="usa.png" link="usa.png" alt="United States" attrlink="usa.png" description="United States on the map.">}}

In other words, the physical meeting points of networks tell only part of the story. The global routing table reveals where address space is actually controlled and originated. Some cities carry significant logical weight without being major public peering hubs. The IP dominance layer exposes that distinction.

This effect, however, was not uniform. One of the most striking—and frankly, baffling—patterns on the map is just how much China is under-represented compared to its actual internet footprint.

{{< figure src="china.png" link="china.png" alt="China" attrlink="china.png" description="China on the map." caption="Note that Hong Kong is the city with the large peering presence, not mainland China.">}}

The Chinese internet is giant: it has massive cable landings, dense domestic fiber networks, and a scale of internal activity that rivals any other region. Yet, when you switch to the logical layer, much of that presence simply vanishes. It’s as if a massive part of the internet is choosing to stay in the shadows. This blackout is likely due to the Great Firewall's peering restrictions, creating a localized intranet that barely touches the public BGP table. So from the perspective of the global routing firehose, parts of the country appear remarkably dark.

### The Data

One of the biggest requests I've had in previous years is for access to the raw data behind the visualizations. For the 2026 edition, I have exposed the underlying JSON datasets that power the map. These files are curated from **TeleGeography** (for modern cables), **PeeringDB** (for IXPs), and various archival sources for historical data.

You can access these directly to build your own visualizations or analyze the growth of global bandwidth.

| File Name | Description |
| --- | --- |
| [`all_cables.json`](https://map.kmcd.dev/data/all_cables.json) | **The Core Map Data.** A GeoJSON FeatureCollection containing all submarine cables. Each feature includes properties like `name`, `rfs_year` (Ready for Service), `decommission_year`, `owners`, and `landing_points`. This follows the standard [GeoJSON format](https://geojson.org/). You can drop this file directly into tools like QGIS, Mapbox Studio, or Google Earth to visualize the cable paths immediately. |
| [`perYearCableStats.json`](https://map.kmcd.dev/data/perYearCableStats.json) | Aggregated statistics for cables by year, including total count, total length added, and the longest cables deployed in that specific year. |
| [`perYearCityData.json`](https://map.kmcd.dev/data/perYearCityData.json) | Data for each city by year, tracking total peering capacity and added capacity. This drives the varying sizes of the city circles on the map. |
| [`yearlyTopCities.json`](https://map.kmcd.dev/data/yearlyTopCities.json) | A leaderboard listing the top 5 cities by peering capacity for every year in the dataset. |
| [`perYearRegionStats.json`](https://map.kmcd.dev/data/perYearRegionStats.json) | Regional aggregation of peering capacity statistics, useful for seeing how connectivity shifted from North America/Europe to Asia and Africa over time. |
| [`year-summaries.json`](https://map.kmcd.dev/data/year-summaries.json) | Brief textual descriptions of notable events or milestones for specific years (e.g., the dot-com boom, the 2008 cable cuts), displayed in the map's footer. |
| [`city-dominance/{year}.json`](https://map.kmcd.dev/data/city-dominance/2026.json) | Detailed per-city breakdown for a specific year, including total IP dominance, physical cable capacity, and the top 10 most influential ASNs. |

**[Explore the Map »](https://map.kmcd.dev)**
