---
categories: ["article", "project", "dataviz"]
tags: ["dataviz", "internet-map", "internet", "networking", "bgp", "fiber-optics", "map", "world", "infrastructure", "threejs", "cloudflare", "javascript"]
keywords: ["interactive internet map", "live internet map", "internet infrastructure map", "submarine cable map", "3d globe internet map", "cloudflare radar outages"]
series: ["Internet Map"]
date: "2026-10-18T10:00:00Z"
description: "Updating map.kmcd.dev with an interactive 3D globe, more BGP cities, historical submarine cables back to 1858, and live internet outage alerts."
cover: "cover.png"
images: ["posts/internet-map-2026-part-2/cover.png"]
featured: true
linktitle: ""
title: "Visualizing the Internet (2026) - Part 2"
slug: "internet-map-2026-part-2"
type: "posts"
canonical_url: https://kmcd.dev/posts/internet-map-2026-part-2
---

In my [February post](/posts/internet-map-2026/) about the [Internet Infrastructure Map](https://map.kmcd.dev), I ended with "See You Next Year." I usually only touch the map once a year when I pull new data and write up an annual update. This year I kept tinkering with it anyway, and enough has changed that it didn't feel right waiting until next February. You can check out the current version at **[map.kmcd.dev](https://map.kmcd.dev)**.

{{< diagram >}}
{{< figure src="globe-overview.png" alt="The new 3D globe view on map.kmcd.dev" caption="The 3D globe view @ **[map.kmcd.dev](https://map.kmcd.dev)**" loading="lazy" >}}
{{< /diagram >}}

## 3D Globe

Up until now, the map used [Leaflet](https://leafletjs.com/). The tileset I was using with Leaflet dropped its free tier and started requiring a paid API key. I found an alternative for a while, but I didn't like how things felt. This update is essentially the end result of that saga: I replaced Leaflet with **Three.js** and instead of the usual map tiling, I now use an SVG for the country lines and rasterize it (with a cache) to an offscreen `<canvas>` texture for the visible viewport when zoom settles, so you don't lose fidelity or see super blurry borders. The **3D Globe** is now the default, though you can still switch back to the 2D view (which swaps to an orthographic Web Mercator Three.js scene while keeping your camera position synced).

{{< diagram >}}
{{< compare before="view-globe.png" after="view-2d.png" caption="The 3D Globe and the 2D map." >}}
{{< /diagram >}}

For the cables on the 3D globe, I added two ways to view them. In **Arches** mode, cables arc above the globe between landing stations with animated pulses. In **Surface** mode, the cables stay flat along their estimated undersea routes.

{{< diagram >}}
{{< compare before="cables-surface.png" after="cables-arches.png" caption="Surface cables vs. 3D arches." >}}
{{< /diagram >}}

## More BGP Data

In [Part 1](/posts/internet-map-2026/), I only mapped IPv4 BGP routes and filtered out smaller cities to keep the map from bogging down. This time I added IPv6 routes, kept all of the smaller cities on the map, and pulled historical GeoIP snapshots for each year back to 2010 instead of relying on a single modern database. Zooming in also adds text labels for the largest cities in view.

{{< diagram >}}
{{< figure src="hub-closeup.png" alt="Zoomed-in view of Western Europe showing city labels and BGP cities" caption="Western Europe with individual BGP cities and city labels." loading="lazy" >}}
{{< /diagram >}}

On the map, blue dots are cities with public IXP peering and purple dots are cities that only announce BGP IP space. Once I put all of the smaller cities on the globe, I kept noticing big purple dots sitting right on top of major fiber lines with millions of IPs and no local IXP at all. I'm working on a follow-up post about why that happens and what it does to your traffic if you live in one of those cities.

Rather than manually running the data scripts once a year, the map data now updates automatically on the first of every month from the latest BGP routing tables and PeeringDB snapshots.

## Layers and Filtering

You can toggle individual layers (text labels, outage alerts, cable display mode, peering capacity, IPv4 addresses, and IPv6 subnets) from the layers menu in the top toolbar:

{{< diagram width="300px" >}}
{{< figure src="layers-menu.png" alt="The layers menu in the top toolbar" caption="The layers menu for toggling text, outage alerts, cable display mode, peering, IPv4 space, and IPv6 subnets." loading="lazy" >}}
{{< /diagram >}}

The search bar in the toolbar also filters both the 3D globe and 2D map across cables, cities, and countries at once. For example, filtering for `France` isolates every submarine cable landing in France alongside its peering and BGP cities:

{{< diagram >}}
{{< figure src="search-france.png" alt="Filtering the map for France" caption="Searching for 'France' isolates French cities and every submarine cable landing in the country." loading="lazy" >}}
{{< /diagram >}}

## Performance

Putting all of those cities and cables onto a 3D globe at once initially made the page very slow, especially when zooming in and out.

To keep the map at 60 FPS, I moved the city clustering calculations into a Web Worker and split the city data into smaller files that load as you zoom in. Hovering over a cable also used to rebuild its Three.js geometry on every mouse move; now all the cables are merged into a single mesh and I just update a texture to highlight the selected one. I also turned off raycasting on the background country polygons.

## Historical Data

If you dragged the timeline slider back to the 1990s or early 2000s on the old map, a lot of old cables were either missing or had the wrong activation and retirement years. I went through historical cable records to add the missing systems and fix retirement dates. You can also download the full list of cables (from 1858 to 2026) as a CSV at **[cables.csv](https://data.map.kmcd.dev/cables.csv)**. If you notice any errors in the data, please [reach out](/me/).

{{< diagram >}}
{{< figure src="historical-timeline.png" alt="Decommissioned historical submarine cables view" caption="Filtering for [decommissioned cables (`q=decomm`)](https://map.kmcd.dev/?q=decomm) across the historical dataset." loading="lazy" >}}
{{< /diagram >}}

There is also a new **Statistics** panel that updates as you move the timeline slider or type in a search filter, showing how many cables are active and how much bandwidth each region has in that year:

{{< diagram >}}
{{< figure src="statistics-panel.png" alt="The Statistics panel on map.kmcd.dev" caption="The Statistics panel for the selected year." loading="lazy" >}}
{{< /diagram >}}

## Outage Alerts

I also wanted `map.kmcd.dev` to show active internet disruptions alongside the physical and routing layers. The map now pulls outage and anomaly alerts from **Cloudflare Radar** and **IODA** every 15 minutes and overlays them directly onto the globe.

{{< diagram >}}
{{< figure src="cloudflare-alerts.png" alt="Active outage alert on the map" caption="Active outage alerts on the map, with links to Cloudflare Radar." loading="lazy" >}}
{{< /diagram >}}

Countries with an active outage highlight in red on the globe, and clicking the alert badge links out to the incident on Cloudflare Radar. When I first added this, I forgot to hide the alerts when scrubbing the timeline slider, so for a day or two it looked like Cloudflare was reporting live outages in 1995.

## End

We'll have to see what direction this takes next year. I'm really happy with how the project has evolved, starting as a simple rendering of submarine cables and growing into a map that glues together routing tables, peering databases, historical archives, and live outage feeds into a single view.

You can explore the updated map here: **[map.kmcd.dev](https://map.kmcd.dev)**
