---
categories: ["article", "project", "dataviz"]
tags: ["dataviz", "internet-map", "internet", "networking", "fiber-optics", "map", "world", "infrastructure", "peeringdb", "leaflet", "javascript"]
keywords: ["interactive internet map", "live internet map", "internet infrastructure map", "internet visualized", "global internet map", "internet exchange map", "internet exchange point map", "internet map of the world", "ixp map", "internet node map", "internet hub locations", "submarine cable map"]
series: ["Internet Map"]
date: "2025-06-18"
description: "An all-new interactive map of the Internet, showing the evolution of undersea cables and internet exchanges with year-by-year animation and detailed statistics."
cover: "cover.png"
images: ["posts/internet-map-2025/cover.png"]
featured: true
featuredalt: "A screenshot of the new interactive internet map, showing undersea cables and peering cities across Europe."
featuredpath: "date"
linktitle: ""
title: "Visualizing the Internet (2025)"
slug: "internet-map-2025"
type: "posts"
canonical_url: https://kmcd.dev/posts/internet-map-2025
---

For the past couple of years, I've been creating visualizations of the internet's physical infrastructure. This project pieces together data from a few sources, and for me, seeing this data visualized together is compelling. These maps show the undersea fiber optic cables that form the backbone of global connectivity and the Internet Exchange Points (IXPs) where networks meet. This year, I'm thrilled to announce a major evolution of the project. Instead of static images and pre-rendered videos, the Internet Map is now a fully interactive, animated map that you can see online.

You can explore the new map live at [**map.kmcd.dev**](https://map.kmcd.dev).

The new version lets you take control. You can pan, zoom, and step through time from the earliest days of the subsea cable network to the latest deployments in 2025. A new statistics panel provides a detailed snapshot for any given year, offering a richer understanding of how our connected world has grown.

{{< diagram >}}
<a href="https://map.kmcd.dev" target="_blank">
{{< image src="screenshot.png" alt="Map of the Internet" >}}
</a>
{{< /diagram >}}

### What You're Looking At

The map visualizes two critical components of the internet's physical layer.

#### A Note on What You're Seeing (and Not Seeing)
It's important to note that this map visualizes publicly available data, which doesn't capture the full picture of global connectivity. The city peering information, for instance, is sourced from [PeeringDB](https://www.peeringdb.com/), which tracks publicly advertised connections at IXPs. A vast amount of internet traffic also flows through private peering arrangements and paid transit links that are not publicly documented and therefore do not appear here.

Similarly, the map focuses on the *intercontinental* backbone of submarine cables. It does not show the incredibly dense web of terrestrial fiber optic cables that run under our streets and alongside major roads. While that data would be fascinating, visualizing it would be overwhelming, and acquiring a complete dataset is nearly impossible as network providers rarely share this proprietary information.

#### Submarine Cables
The lines snaking across the ocean floors are [submarine communications cables](https://en.wikipedia.org/wiki/Submarine_communications_cable). These bundles of fiber optic strands are the high-speed data arteries that connect continents. Laying and maintaining them is a modern marvel of engineering, involving everything from specialized cable-laying ships to underwater robots for repairs. As you explore the map, you can see how the web of these cables has become denser over time, enabling the global, real-time communication we now take for granted. By the start of 2025, the network has grown to **599** cables, spanning a staggering **1,602,092 kilometers**.

{{< diagram >}}
{{< image src="alwayshasbeen.png" alt="Map of the Internet" >}}
{{< /diagram >}}

##### A Physical Target: Vulnerabilities and Sabotage

While these cables are heavily armored, especially in shallower coastal waters where most damage occurs, their isolation on the seabed makes them vulnerable. For decades, the most common threat has been accidental damage from fishing trawlers and dragged anchors. However, in recent years, a more alarming trend has emerged: intentional sabotage. The increasing frequency of suspicious cable cuts suggests that these vital arteries of communication are becoming targets in geopolitical conflicts, a reality that may have brought many new visitors to this map.

Here are just a few of the many recent incidents:

- **[The Balticconnector Pipeline and Cable Damage (October 2023)](https://jamestown.org/program/strangers-on-a-seabed-sino-russian-collaboration-on-undersea-cable-sabotage-operations/)**: The Balticconnector gas pipeline and two telecom cables between Finland and Estonia were damaged by a dragged anchor from the Hong Kong-flagged ship *Newnew Polar Bear*. China later admitted its vessel was responsible but claimed it was an accident.

- **[The C-Lion1 and BCS East-West Interlink Cuts (November 2024)](https://www.cbsnews.com/news/undersea-cables-cut-europe-finland-germany-hint-russia-sabotage/)**: Two key telecom cables in the Baltic Sea, C-Lion1 (Finland-Germany) and BCS East-West Interlink (Lithuania-Sweden), were severed. The Chinese-owned cargo ship *Yi Peng 3* was the primary suspect after it was observed making anomalous movements in the area.

- **[The Christmas Day Baltic Cable Cuts (December 2024)](https://apnews.com/article/nato-france-russia-baltic-cables-ships-damage-764964a275530915c2cc5af1125ec125)**: On Christmas Day 2024, the Estlink 2 power cable and other telecom cables between Finland and Estonia were damaged by a dragged anchor. Finnish authorities seized the suspected vessel, the oil tanker *Eagle S*, which was identified as part of Russia's "shadow fleet".

- **[The Matsu Islands Blackout (February 2023)](https://www.vice.com/en/article/taiwan-internet-cables-matsu-china/)**: Two undersea cables connecting Taiwan to its outlying Matsu Islands were severed by Chinese vessels, leaving the 14,000 residents with severely disrupted internet for over 50 days. The incident highlighted the societal impact of such disruptions and was seen as part of a broader pressure campaign by China.

- **[The Trans-Pacific Express Cable Cut (January 2025)](https://www.twz.com/news-features/taiwan-coast-guard-blames-chinese-owned-ship-for-cutting-undersea-communications-cable)**: The major Trans-Pacific Express international cable was cut near Taiwan, with suspicion falling on the Chinese-owned cargo ship *Shunxin 39*. The vessel had a history of using multiple identities to evade tracking and sailed erratically over the cable's location before the incident.

- **[The Red Sea Cable Disruption (February 2024)](https://www.csis.org/analysis/red-sea-cable-damage-reveals-soft-underbelly-global-economy)**: Three critical cables in the Red Sea were severed by the anchor of the sinking cargo ship *Rubymar*, which had been struck by a Houthi missile. The incident disrupted a significant portion of Europe-Asia data traffic and highlighted the vulnerability of infrastructure in contested maritime chokepoints.

- **[The Gulf of St. Lawrence Sabotage (December 2023 & 2024)](https://www.cbc.ca/news/canada/nova-scotia/bell-subsea-fibre-optic-cable-newfoundland-1.7461963)**: A subsea cable connecting Nova Scotia and Newfoundland was deliberately cut in December 2023 and again in December 2024. Evidence showed an "angle grinder cut" through the steel-wrapped cable, confirming sabotage, though the perpetrator and motive remain unknown.

These examples represent only a fraction of such incidents, which have escalated in frequency and impact in recent years.

#### Internet Exchange Points (IXPs)
The circles on the map represent cities with [Internet Exchange Points](https://www.cloudflare.com/learning/cdn/glossary/internet-exchange-point-ixp/). If submarine cables are the interstate highways of the internet, then IXPs are the bustling, hyper-connected metropolitan areas where all the traffic is headed.

An IXP is a physical data center, or a set of connected data centers, where many different networks can physically plug into each other to exchange traffic directly. This process is called "peering."

So why do hundreds of networks choose to gather in the same buildings in cities like Frankfurt or Amsterdam? The answer is a powerful network effect that you could call digital gravity. The value of an IXP is determined by the networks present there. Once a major network joins, it becomes exponentially more attractive for others to join as well.

The most powerful sources of this gravity are large content providers like Google, Meta, Apple, and Netflix. These companies have an "open peering policy." In essence, they are saying to any Internet Service Provider (ISP): "Connect with us directly here at the IXP, and we will give your customers a faster, better path to our services."

This creates a powerful win-win scenario:

- The ISP wins because their customers get lightning-fast, low-latency access to YouTube, Google Drive, or Apple's App Store. This makes the ISP's own service more valuable and competitive.
- Google and Apple win because they get to deliver their content without paying high fees to third-party backbone carriers. Every byte of data they serve over a direct peering connection is money saved.

This economic incentive is the engine that drives the growth you see on the map. ISPs flock to the IXPs where the big content providers are, which in turn attracts more content providers, creating a feedback loop of ever-increasing capacity and value. This is why a handful of cities have become global hubs with staggering traffic volumes, while others remain smaller, regional nodes.

{{< d2 >}}
direction: down

content_providers: "Content & Services" {
  netflix: "Netflix" {
    icon: "images/icons/netflix-icon.svg"
  }
  google: "Google" {
    icon: "images/icons/google-icon.svg"
  }
  aws: "AWS Cloud" {
    icon: "images/icons/amazon_aws-icon.svg"
  }
}

consumers: "Internet Users" {
  icon: "data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z'/%3E%3Cpolyline points='9 22 9 12 15 12 15 22'/%3E%3C/svg%3E"
}

ixp: "Internet Exchange Point\n(e.g., DE-CIX Frankfurt)" {
  shape: cloud
}
content_providers -> ixp: "Direct Peering\n(Often free / cheaper)"
ixp -> consumers

transit: "Paid Transit Provider" {
  icon: "data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' xmlns='http://www.w3.org/2000/svg'%3E%3Cline x1='12' y1='1' x2='12' y2='23'/%3E%3Cpath d='M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6'/%3E%3C/svg%3E"
}
content_providers -> transit: "Expensive Paid Transit\n(Often Slower Path)" {
  style: {
    stroke-dash: 4 # Dashed line for a less direct route
  }
}
transit -> consumers {
  style: {
    stroke-dash: 4 # Dashed line for a less direct route
  }
}
{{< /d2 >}}

### The World in 2025: A Snapshot
The animation and data now extend to 2025, revealing significant ongoing investment. In this year alone, **31 new cables** were added (or are promised very soon), stretching over **144,320 kilometers**, enough to circle the earth *three and a half times*.

Some of the longest and most impactful new cables of 2025 include:

* **[Bifrost](https://www.submarinenetworks.com/en/systems/trans-pacific/bifrost) (19,888 km):** A monumental project directly connecting Singapore to North America via Indonesia, the Philippines, and Guam. It's a joint effort by Meta, Keppel, and Telin to bolster connectivity across the Asia-Pacific region.
* **[Echo](https://www.submarinenetworks.com/en/systems/trans-pacific/echo/echo-cable-system-overview) (17,184 km):** Another critical trans-Pacific cable, built by Google and Meta, that forges a new, resilient path from the U.S. to Singapore, also landing in Guam and Indonesia. This cable deliberately avoids the crowded northern routes to increase network diversity.
* **[Firmina](https://www.submarinenetworks.com/en/systems/brazil-us/firmina) (14,517 km):** A Google-led cable enhancing the North-South America connection. It runs from the U.S. East Coast to Argentina, with landings in Brazil and Uruguay, dramatically improving access to Google services in South America.
* **[TPU](https://www.submarinenetworks.com/en/systems/trans-pacific/tpu) (13,470 km):** A Google-owned cable system connecting the U.S. with Taiwan and the Philippines, bolstering trans-Pacific capacity.
* **[JUNO](https://www.submarinenetworks.com/en/systems/trans-pacific/juno) (11,710 km):** A cable system by Seren Juno Network connecting Japan to the U.S., utilizing advanced technology to offer a high number of fiber pairs and enhance communication resiliency.

### Regional Peering Powerhouses

Looking at the total peering capacity reveals a clear global hierarchy. Europe remains the undisputed leader, with an incredible **1.5 Pbit/s** of capacity. Asia and North America follow with robust networks of their own, while South America shows impressive growth.

```mermaid
pie title Peering Capacity by Region (2025)
    "Europe": 1500
    "Asia": 430
    "North America": 403
    "South America": 313
    "Africa": 67.9
    "Oceania": 67.2
````

This massive regional capacity is concentrated in a few key metropolitan hubs. The list of top peering cities shows just how vital they are to the global network:

- **Amsterdam, NL**: 200 Tbit (+12.7 Tbit)
- **Frankfurt, DE**: 166 Tbit (+8.19 Tbit)
- **São Paulo, BR**: 157 Tbit (+3.16 Tbit)
- **London, GB**: 113 Tbit (+3.21 Tbit)
- **Tokyo, JP**: 90.2 Tbit (+2.18 Tbit)

The growth in a city like São Paulo is remarkable and shows the increasing investment in internet infrastructure in South America, directly supported by new cables like Firmina.

### How It's Made: The New Tech Stack

The transition to an interactive map required a complete overhaul of the technology stack. The previous versions, which relied on generating static SVG images, faced several challenges. It was difficult to dynamically size the lines representing cables and find the right balance of detail for country borders; too much detail slowed the map down, while too little looked simplistic when zoomed in.

The solution was to adopt a tile-based methodology, where map tiles at different levels of detail are fetched dynamically as a user zooms—the same concept used by Google Maps. I was faced with a choice: implement this highly complex tiling logic myself or use a well-supported library. Since the project's focus was on data visualization, I opted for the more direct path by using [Leaflet](https://leafletjs.com/), a powerful library for creating dynamic and interactive maps.

The back-end Go scripts that gather and process the data from sources like [TeleGeography](https://telegeography.com/) and [PeeringDB](https://www.peeringdb.com/) were largely unchanged, only needing a few new fields in the JSON output to power the new front end.

{{< diagram >}}
{{< image src="expand.png" alt="Map of the Internet" >}}
{{< /diagram >}}

Beyond the core technology, I also wanted to incorporate a few small details to improve the user experience and make the map feel more intuitive and personal:

- **Personalized View**: The map automatically geolocates your region and centers the initial view there. My hope is that the map feels familiar and relevant the moment you open it, no matter where you are in the world.
- **Persistent 'About' Section**: The 'About this map' panel remembers its state. If you close it, it stays closed on subsequent visits and refreshes until you decide to open it again.
- **Relative City Sizing**: The size of each city circle is relative to its total peering bandwidth, giving an immediate visual sense of where the major hubs of connectivity are concentrated.
- **Interactive Highlighting**: Hovering over or clicking on any cable or city highlights it and brings it to the forefront. This small detail makes it much easier to focus on and explore individual parts of the network.

### Closing Thoughts

This project continues to be a fascinating exploration of the physical reality of our digital world. By making the map interactive, I hope to provide a more powerful tool for anyone curious about the immense and intricate infrastructure that underpins our daily lives. As global data demand soars, the growth of these subsea cables and peering exchanges will only become more critical. Explore the map, watch the internet grow, and see for yourself how the world gets connected.

This project was a significant undertaking, and I've been thrilled to see it shared in various places online. If you choose to share it, I only ask that you please provide attribution by linking back to the project, just as I give credit to my own data sources, [TeleGeography](https://telegeography.com/) and [PeeringDB](https://www.peeringdb.com/).

Thank you for reading, and please feel free to explore and share the map with others! [map.kmcd.dev](https://map.kmcd.dev)
