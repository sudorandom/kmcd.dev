---
categories: ["article", "project"]
tags: ["dataviz", "spectrum", "sun", "light", "python", "svg", "data", "science"]
date: "2022-02-19"
description: "Visualizing the visual spectrum of the sun. Spectrum analysis, wavelengths of light, data visualization"
featured: "thumbnail.png"
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Visualizing the spectrum of the sun"
images: ["/posts/sun-spectra-image.svg/thumbnail.png"]
slug: "sun-spectra-image.svg"
type: "posts"
devtoSkip: true
aliases: [
    "portfolio/sun-spectra-v1/",
]
mastodonID: "112277307770471020"
---

## Basic Details

We all know the light that we get from the sun is white, meaning it contains about the same amount of every single color. If it had more green than other colors then the sun would give off green light. That's awesome. Except... it's not really true. There are certain colors that the sun does NOT give us. There are essentially holes in every rainbow. And the images below show exactly where those holes are.

[Github](https://github.com/sudorandom/sun-fingerprint) | [All output images](https://github.com/sudorandom/sun-fingerprint/tree/main/output)

-------
## The sun's spectra (visible light)
![Spectra of the sun in visible spectrum]({{< permalink "visible.svg" >}} "The Sun")
[Click here to see the full resolution]({{< permalink "visible.svg" >}})

## The sun's spectra (full spectrum)
I also made a version that shows the NON visible spectrum. I can't use pretty colors for this because we literally don't have colors to map this part of the spectrum to. So I used greyscale (a gradient from black to white) to denote the intensity of this kind of light emitted from the sun. The image looks blurry but that's an artifact of how precise the data is.

![Spectra of the sun in all spectrums]({{< permalink "non-visible.svg" >}} "The Sun")
[Click here to see the full resolution]({{< permalink "non-visible.svg" >}})


## The sun's spectra (annotated)

I also made a version that has text that describes what you're seeing.

![Spectra of the sun, annotated]({{< permalink "annotated.svg" >}} "The Sun")
[Click here to see the full resolution]({{< permalink "annotated.svg" >}})
