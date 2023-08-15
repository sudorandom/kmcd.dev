+++
categories = ["project", "data-visualization"]
tags = ["dataviz", "internet", "networking", "fiber optics", "map", "world", "infrastructure", "javascript", "svg"]
date = "2022-02-26"
description = "I drew a pretty map that shows the underwater cables that carry our data around the world; fiber optic cables, submarine cables"
images = ["/posts/internet-map-v1/thumbnail.png"]
featured = "thumbnail.png"
featuredalt = ""
featuredpath = "date"
linktitle = ""
title = "Visualizing the Internet (2022)"
slug = "internet-map-v1"
type = "posts"
aliases = [
  "/posts/submarine-cable-map.svg",
  "/portfolio/submarine-cable-map",
]
+++

> There is an updated version of this map detailed [in this updated post](/posts/internet-map-v2/).

### Basic Details
I used data from the [submarinecablemap.com](https://submarinecablemap.com) website to create my own visualization of Submarine Cables that live under our oceans and carry the majority of trans-continental internet traffic. Mostly, I wanted a 'dark mode' version of the map but I also plan on adding some interesting annotations from different sources and computing some metrics... Like there is enough fiber optic cable under the oceans to wrap the earth over 103 times! These SVGs were made with javascript, [d3](https://d3js.org). I also used this experience to look at different map projections, which is neat.


[Github](https://github.com/sudorandom/submarine-cable-map) | [All output images](https://github.com/sudorandom/tree/main/output)

-------

Here are the resulting images.

### geo-mercator.svg
![Cable map using the mercator projection](geo-mercator.svg "geo-mercator.svg")
[click here for full resolution](geo-mercator.svg)

### geo-conic-conformal.svg

![Cable map using the conic coformal projection](geo-conic-conformal.svg "geo-conic-conformal.svg")
[click here for full resolution](geo-conic-conformal.svg)

### geo-conic-equal-area.svg
![Cable map using the Geo Conic Equal Area projection](geo-conic-equal-area.svg "geo-conic-equal-area.svg")
[click here for full resolution](geo-conic-equal-area.svg)

### geo-natural-earth-1.svg
![Cable map using the Geo Natural Earth projection](geo-natural-earth-1.svg "geo-natural-earth-1")
[click here for full resolution](geo-natural-earth-1.svg)
