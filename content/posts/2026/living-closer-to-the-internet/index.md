---
categories: ["article", "analysis", "dataviz"]
tags: ["dataviz", "internet-map", "internet", "networking", "bgp", "ixp", "peering", "latency", "fiber-optics", "infrastructure"]
keywords: ["internet exchange point latency", "trombone routing", "hairpin routing", "peering vs transit", "ixp map", "local peering latency", "private network interconnect"]
series: ["Internet Map"]
date: "2026-10-24T10:00:00Z"
description: "Why gigabit fiber in your apartment still comes with a latency penalty if your city lacks an active Internet Exchange Point."
cover: "cover.png"
images: ["posts/living-closer-to-the-internet/cover.png"]
linktitle: ""
title: "Living Closer to the Internet"
slug: "living-closer-to-the-internet"
type: "posts"
canonical_url: https://kmcd.dev/posts/living-closer-to-the-internet
---

In my [recent update to the Internet Map](/posts/internet-map-2026-part-2/), I expanded the BGP layer on **[map.kmcd.dev](https://map.kmcd.dev)** to show all attributed IP address space globally instead of just the largest hubs.

On the map, blue dots are cities with public [Internet Exchange Points (IXPs)](https://www.internetsociety.org/issues/ixps/) and purple dots are cities that only announce BGP IP space. Once I put all of the smaller cities on the globe, I noticed a lot of big purple dots sitting right on top of major fiber lines with millions of IPs and no local IXP.

{{< diagram >}}
{{< figure src="ixp-vs-bgp-cities.png" alt="Blue IXP cities and purple BGP-only cities on map.kmcd.dev" caption="Blue IXP cities and purple BGP-only cities on **[map.kmcd.dev](https://map.kmcd.dev)**." loading="lazy" >}}
{{< /diagram >}}

Even if you have gigabit fiber to your apartment, living in a city without an IXP affects how your traffic is routed.

## Trombone Routing

If you and your neighbor across the street are on two different ISPs, your fiber cables probably run through the same conduit under the road. If those two ISPs don't have a place to connect in your city, your packets have to leave town to reach your neighbor, go to another city where both ISPs peer, and come back. This is called **[trombone routing](https://resources.telegeography.com/what-is-the-trombone-effect)**:

{{< diagram >}}
{{< figure src="trombone-routing-map.svg" alt="OpenStreetMap map of Denmark with arrows from Odense to Copenhagen on ISP A and back to Odense on ISP B." caption="Without a local exchange, traffic between two Odense networks takes a detour to Copenhagen and back." loading="lazy" >}}
{{< /diagram >}}

Take two users in Odense whose ISPs only peer in Copenhagen. For their packets to cross the street, they have to leave the island of Funen, cross over to Zealand, hit an exchange in Copenhagen, and turn right back around.

With a local IXP in Odense, both ISPs can exchange local routes directly in town:

{{< diagram >}}
{{< figure src="local-peering-map.svg" alt="The same map of Odense and Copenhagen, with a green route staying in Odense through hypothetical local peering." caption="With a local IXP in Odense, the traffic stays in town." loading="lazy" >}}
{{< /diagram >}}

## Same Fiber, Different Paths

You can see this in latency measurements from [RIPE Atlas](https://atlas.ripe.net/) and [M-Lab](https://www.measurementlab.net/tests/ndt/) if you compare cities in the same region that use the same ISPs:

| Cities | With IXP | Without IXP | Detour |
| :--- | :--- | :--- | :--- |
| **[Copenhagen](https://www.peeringdb.com/ix/78) / Odense** | 1.5–3 ms | 8.5–13 ms | Copenhagen |
| **[Columbus](https://www.peeringdb.com/ix/869) / Cincinnati** | 2.5–4.5 ms | 18–26 ms | Chicago / [Ashburn](/posts/ashburn/) |
| **[Calgary](https://www.peeringdb.com/ix/639) / Saskatoon** | 2.5–5 ms | 22–34 ms | Seattle / Winnipeg |
| **[Stuttgart](https://www.peeringdb.com/ix/212) / Karlsruhe** | 1.8–3 ms | 11–15 ms | Frankfurt |
| **[Lyon](https://www.peeringdb.com/ix/69) / Clermont-Ferrand** | 2–3.5 ms | 14–19 ms | Paris |

I [live in Copenhagen](/posts/texans-in-denmark/), where the major Danish ISPs peer with each other at [DIX](https://www.peeringdb.com/ix/78) and [Netnod](https://www.peeringdb.com/ix/193). In [Odense](https://map.kmcd.dev/?lat=55.55&lng=11.45&z=7.00&year=2026) (165 km west of Copenhagen), those same ISPs carry their regional traffic back to Copenhagen to peer, so packets between two ISPs in Odense cross the Great Belt bridge to Copenhagen and back. You can see the same thing in [Ohio](https://map.kmcd.dev/?lat=39.96&lng=-83.50&z=6.20&year=2026), where Columbus has a local exchange ([Ohio IX](https://www.peeringdb.com/ix/869)) and Cincinnati routes cross-ISP traffic through Chicago or [Ashburn](/posts/ashburn/). And in [Calgary](https://map.kmcd.dev/?lat=51.50&lng=-110.50&z=5.50&year=2026), local traffic used to go through Seattle, Chicago, or Toronto at ~30 ms until the city set up [YYCIX](https://www.peeringdb.com/ix/639) and brought local pings under 5 ms.

## Empty IXPs and Private Cables

The blue and purple dots on the map don't show everything, since [PeeringDB](https://www.peeringdb.com/) only lists public exchanges.

Some cities have a blue IXP dot on the map, but the big residential cable and phone companies in town don't connect to it. Large providers [often require](https://drpeering.net/white-papers/Peering-Tactics/Top-Ten-Reasons-Not-To-Peer.html) a network to meet minimum traffic levels and connect with them in several major cities before they will peer for free, especially if that smaller network would otherwise have to [pay them for transit](https://www.kentik.com/kentipedia/ip-transit-vs-peering/). Often the local exchange is only used by a university and a few small regional ISPs, and the big national providers still route all of their home customers through Chicago, Ashburn, or Frankfurt.

Other cities only show up in purple on the map because the big networks there [connect privately](https://www.kentik.com/kentipedia/what-is-internet-peering/). When an ISP and a CDN have routers in the same building, they can plug a fiber cable directly between them using a **[Private Network Interconnect (PNI)](https://en.wikipedia.org/wiki/Peering#Private_peering)**. Since PNIs are private agreements and cables between two companies, they aren't in PeeringDB and don't show up on my map.

## Check Your City on the Map

The cable layer on the map shows where the physical fiber goes, and the IXP and BGP layers show where networks actually connect to each other. You can check your own city on **[map.kmcd.dev](https://map.kmcd.dev)** to see whether it shows up in blue or purple.

**[Explore the Map »](https://map.kmcd.dev)**
