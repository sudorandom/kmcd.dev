---
categories: ["article", "project", "dataviz"]
tags: ["dataviz", "internet-map", "internet", "networking", "bgp", "fiber-optics", "radio", "telegraph", "history", "map", "world", "infrastructure"]
keywords: ["physical internet", "how the internet works", "submarine cable history", "undersea telegraph cable", "vlf radio stations", "shortwave radio trunks", "coaxial submarine cable", "tat-1", "tat-8", "undersea fiber optic cables", "edfa amplifier", "ixp peering", "bgp routing", "decommissioned submarine cables"]
series: ["Internet Map"]
date: "2026-11-03T10:00:00Z"
description: "Adding a visual history of global communications at map.kmcd.dev/learn/, backfilling decommissioned telegraph and coaxial cables, and mapping intercontinental radio trunks."
cover: "cover.webp"
images: ["posts/from-copper-to-glass/cover.webp"]
featured: true
linktitle: ""
title: "The Physical Internet: From Copper to Glass"
slug: "from-copper-to-glass"
type: "posts"
canonical_url: https://kmcd.dev/posts/from-copper-to-glass
draft: true
---

I originally built the [Internet Infrastructure Map](https://map.kmcd.dev) because I thought undersea fiber-optic cables were interesting. Since then, I've been slowly piecing the history of the global Internet together. After adding historical fiber cables, I started backfilling old telegraph and coaxial cables. Then I realized the picture still wasn't complete: if you scrubbed through the early-to-mid 1900s on the map, it didn't make any sense. Major routes were being decommissioned with no alternative replacing them.

The missing piece was point-to-point radio systems that took over intercontinental traffic during that era. Once I added radio to the map, I also built a new page at **[map.kmcd.dev/learn/](https://map.kmcd.dev/learn/)** to walk through the technological history that led to our modern internet.

{{< diagram >}}
{{< figure src="learn-telegraph.png" alt="The new /learn/ page on map.kmcd.dev showing the Copper Pulses telegraph section" caption="The new page at **[map.kmcd.dev/learn/](https://map.kmcd.dev/learn/)**." loading="lazy" >}}
{{< /diagram >}}

## Backfilling Decommissioned Cables

Modern submarine cable databases focus almost exclusively on active fiber systems. When I first added the timeline slider down to 1858, a lot of historical cables were either missing or had the wrong activation and retirement years. If a cable's retirement year was missing, it either stayed on the map forever or never showed up at all.

I went back through historical records to backfill dozens of decommissioned systems and fix their retirement dates across three generations of undersea cables:

- **Telegraph Cables (1858–1900s):** From the short-lived **1858 Transatlantic Cable** and the **1866 Cable** laid by the *SS Great Eastern* to the **1870 UK–Australia** chain, the **German Atlantic (1900)** cable, the British **All Red Line (1902)**, and the **First Transpacific (1903)** cable.
- **Coaxial Telephone Cables (1920s–1980s):** Starting with **Key West–Havana (1921)** and **TAT-1 (1956)** through **TAT-7**, **CANTAT-1/2**, **COMPAC**, **TPC-1/2**, and **SAT-1**.
- **Retired First-Generation Fiber (1988–2000s):** Early optical systems from the late 1980s and 1990s that have already been retired, like **TAT-8**, **HAW-4**, **TPC-3**, and **SEA-ME-WE 1 & 2**.

I also backfilled the historical BGP routing layer on the map from 2010 back to **1997** using archived Cisco `show ip bgp` text dumps from [RouteViews](https://archive.routeviews.org/oix-route-views/) and early MRT snapshots from [RIPE RIS](https://data.ris.ripe.net/rrc00/).

{{< diagram >}}
{{< figure src="map-coax-1965.png" alt="The map in 1965 showing coaxial submarine cables and decommissioned radio links" caption="The map in **[1965](https://map.kmcd.dev/?year=1965)** as coaxial telephone cables were replacing older telegraph and radio links." loading="lazy" >}}
{{< /diagram >}}

## Why the Map Needed Radio

Fixing the decommission dates on the old telegraph cables created a new problem. When I dragged the timeline slider through the early-to-mid 1900s, major routes were turning red and disappearing with nothing taking their place. Early transatlantic telegraph cables were retiring in the 1910s and 1920s, the **German Atlantic** cable was cut in 1914 with no replacement cable for decades, and the **1903 First Transpacific** cable retired in 1951—years before the first transpacific coaxial cables (**HAW-1** and **TPC-1**) were laid.

Looking at a cables-only map, it looked like the world was disconnecting itself in the first half of the 1900s. In reality, those routes moved into the air. Before undersea vacuum-tube repeaters made transoceanic telephone cables possible in 1956, point-to-point longwave and shortwave radio networks took over long-distance telegraphy and carried the first intercontinental phone calls.

{{< diagram >}}
{{< figure src="map-radio-1929.png" alt="The map in 1929 showing wavy point-to-point radio links across the globe" caption="Intercontinental radio trunks (wavy purple lines) alongside submarine telegraph cables in **[1929](https://map.kmcd.dev/?year=1929)**." loading="lazy" >}}
{{< /diagram >}}

To fill in that gap, I added a **Radio** layer to **[map.kmcd.dev](https://map.kmcd.dev)** (rendered as wavy purple lines) covering major intercontinental radio links from 1907 to the 1970s—including Marconi's early transatlantic stations, the British Imperial Wireless Chain, the Dutch *"Hallo Bandoeng"* link to Java, AT&T and RCA's transatlantic and transpacific trunks, Denmark's Skamlebæk Radio across the North Atlantic, and high-power VLF stations like Grimeton (SAQ) and Rugby (GBR).

With radio on the map, the timeline in the 1900s finally makes sense. You can see radio links spin up in the 1910s and 1920s as old telegraph cables age out, and then watch each radio trunk retire in the 1950s and 1960s the exact year a coaxial cable like **TAT-1 (1956)**, **CANTAT-1 (1961)**, **COMPAC (1963)**, or **TPC-1 (1964)** lands on the same corridor.

## `map.kmcd.dev/learn/`: From Copper to Glass

The map shows *where* all of these systems were built, but there are a lot of cool engineering stories behind how they worked and why each generation replaced the last. I put together **[map.kmcd.dev/learn/](https://map.kmcd.dev/learn/)** as an interactive companion guide to the map.

{{< diagram >}}
{{< figure src="learn-radio.png" alt="The Longwave Ether section on map.kmcd.dev/learn showing ground-wave diffraction" caption="Inside **[map.kmcd.dev/learn/](https://map.kmcd.dev/learn/)**." loading="lazy" >}}
{{< /diagram >}}

It walks through five eras with custom SVG diagrams and direct links to jump to each era on the map:

1. **[Copper Pulses (1850s to 1920s)](https://map.kmcd.dev/learn/#era-telegraph):** Why the first transatlantic cable burned out after three weeks, how 4,000 km of ocean turns a wire into a giant capacitor, and the clever signaling trick that made undersea telegraphy work.
2. **[Longwave Ether (1910s to 1940s)](https://map.kmcd.dev/learn/#era-radio):** How wartime censorship led to a giant antenna strung across a volcanic gorge in Java, and how amateur radio experiments upended early megawatt longwave stations.
3. **[Coaxial Waves (1950s to 1980s)](https://map.kmcd.dev/learn/#era-coax):** Why solar storms pushed intercontinental phone calls back to the ocean floor, and how engineers powered vacuum-tube repeaters every 70 km across the Atlantic.
4. **[Photonic Glass (1988 to Present)](https://map.kmcd.dev/learn/#era-fiber):** Inside an 8-layer modern submarine fiber cable, how undersea repeaters amplify light directly inside the glass without converting it back into electricity, and some of my favorite geographic quirks to spot on the map.
5. **[Network Peering (Present)](https://map.kmcd.dev/learn/#era-bgp):** What happens once the light leaves the ocean—how Internet Exchange Points, private cross-connects, and BGP tie the physical cables together.

## Check It Out

Head over to **[map.kmcd.dev/learn/](https://map.kmcd.dev/learn/)** to read the full guide, or scrub through the updated historical timeline yourself on **[map.kmcd.dev](https://map.kmcd.dev)**.
