---
categories: ["article", "project", "dataviz"]
tags: ["dataviz", "internet-map", "internet", "networking", "bgp", "fiber-optics", "radio", "satellites", "telegraph", "history", "map", "world", "infrastructure"]
keywords: ["physical internet", "how the internet works", "submarine cable history", "undersea telegraph cable", "vlf radio stations", "shortwave radio trunks", "coaxial submarine cable", "tat-1", "tat-8", "communications satellites", "satnet", "starlink vs fiber optic bandwidth", "undersea fiber optic cables", "terrestrial backbone corridors", "edfa amplifier", "roadm", "ixp peering", "bgp routing", "decommissioned submarine cables"]
date: "2026-11-05T10:00:00Z"
description: "Adding a visual history of global communications at map.kmcd.dev/learn/, backfilling decommissioned telegraph and coaxial cables, mapping intercontinental radio trunks, and adding terrestrial backbone corridors."
cover: "cover.webp"
images: ["posts/from-copper-to-glass/cover.webp"]
linktitle: ""
title: "The Physical Internet: From Copper to Glass"
slug: "from-copper-to-glass"
type: "posts"
canonical_url: https://kmcd.dev/posts/from-copper-to-glass
---

I originally built the [Internet Infrastructure Map](https://map.kmcd.dev) because I thought undersea fiber-optic cables were interesting. Since then, I've been slowly piecing the history of the global Internet together. After adding historical fiber cables, I started backfilling old telegraph and coaxial cables. Then I finally added radio systems to the map, stopping short of the monumental task of mapping satellites. I also built a new page at **[map.kmcd.dev/learn/](https://map.kmcd.dev/learn/)** to walk through the technological history that led to our modern internet.

{{< diagram >}}
{{< figure src="learn-telegraph.png" alt="The new /learn/ page on map.kmcd.dev showing the Copper Pulses telegraph section" caption="The new page at **[map.kmcd.dev/learn/](https://map.kmcd.dev/learn/)**." loading="lazy" >}}
{{< /diagram >}}

## Backfilling Decommissioned Cables

Modern submarine cable databases focus almost exclusively on active fiber systems. The historical data is pretty messy. I did a lot of tedious work trying to patch the full history together. There is a new button on the map if you want to download the whole 1858-to-present cable list as a CSV. I suspect it will be useful to some others as well.

While I was digging through old archives, I also pushed the map's historical BGP routing layer back from 2010 to **1997** using [RouteViews](https://archive.routeviews.org/oix-route-views/) and [RIPE RIS](https://data.ris.ripe.net/rrc00/).

{{< diagram >}}
{{< figure src="map-coax-1965.png" alt="The map in 1965 showing coaxial submarine cables and decommissioned telegraph cables" caption="The map in **[1965](https://map.kmcd.dev/?year=1965)** as coaxial telephone cables were replacing older telegraph cables." loading="lazy" >}}
{{< /diagram >}}

## Why the Map Needed Radio

Fixing the decommission dates on the old telegraph cables created a new problem. When I dragged the timeline slider through the early-to-mid 1900s, major routes were turning red and disappearing with nothing taking their place. Out in the Pacific, the **1903 First Transpacific** cable shut down in 1951, years before the first coaxial telephone cables (**HAW-1** and **TPC-1**) were laid. Looking at a cables-only map, it looked like the world was disconnecting itself in the first half of the 1900s.

In reality, those routes moved into the air.

{{< diagram >}}
{{< figure src="map-radio-1929.png" alt="The map in 1929 showing wavy point-to-point radio links across the globe" caption="Intercontinental radio trunks (wavy purple lines) alongside submarine telegraph cables in **[1929](https://map.kmcd.dev/?year=1929)**." loading="lazy" >}}
{{< /diagram >}}

To fill that gap, I added a **Radio** layer to **[map.kmcd.dev](https://map.kmcd.dev)** (rendered as wavy purple lines) covering major intercontinental radio trunks from 1907 into the 1970s. High-power VLF stations also show up on the map as big coverage rings. With radio on the map, the timeline in the 1900s finally makes sense.

## Terrestrial Backbones

You might have noticed that submarine cables are only shown in the water. One reason is the definition of "submarine" but yes, there is a ton of buried fiberoptic cable along the sides of major highways and rail lines all across every continent in the world. This hasn't been represented very well, so that map had huge empty gaps across the middle of continents even with inland IXPs from PeeringDB on the map. To help fix this, I added a new **Backbone** layer (dashed orange lines) to connect major inland IXPs to each other and to the coastal cable landings.

{{< diagram >}}
{{< figure src="map-backbone.png" alt="The map showing terrestrial backbone corridors connecting inland metro hubs across North America, Europe, South America, and India" caption="Terrestrial backbone corridors (dashed orange lines) alongside submarine cables in **[2026](https://map.kmcd.dev)**." loading="lazy" >}}
{{< /diagram >}}

## From Copper to Glass

A map is great for showing where all of these lines went, but I also wanted a place to explain how the hardware worked and why each technology replaced the one before it. That's what **[map.kmcd.dev/learn/](https://map.kmcd.dev/learn/)** is for. It also has a full section on **[satellites](https://map.kmcd.dev/learn/#era-satellite)**, since trying to draw satellite orbits on the main map was a complete mess.

{{< diagram >}}
{{< figure src="learn-radio.png" alt="The Longwave Ether section on map.kmcd.dev/learn" caption="Inside **[map.kmcd.dev/learn/](https://map.kmcd.dev/learn/)**." loading="lazy" >}}
{{< /diagram >}}

{{< diagram >}}
{{< figure src="learn-satellites.png" alt="The Satellites (Orbital Relays) section on map.kmcd.dev/learn" caption="The new **[Satellites](https://map.kmcd.dev/learn/#era-satellite)** section on **[map.kmcd.dev/learn/](https://map.kmcd.dev/learn/)**." loading="lazy" >}}
{{< /diagram >}}

Take a look at **[map.kmcd.dev/learn/](https://map.kmcd.dev/learn/)** and **[map.kmcd.dev](https://map.kmcd.dev)**, and let me know what you think.
