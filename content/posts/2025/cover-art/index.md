---
categories: ["article"]
tags: ["art", "generative-art", "philosophy", "creativity", "ai"]
date: "2025-10-15T10:00:00Z"
description: "Why I felt guilty using AI art, and chose to embrace a more personal, code-driven creative process."
cover: "cover.svg"
images: ["/posts/cover-art/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "On Creating My Own Cover Art"
slug: "cover-art"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/cover-art/
draft: true
---

I felt guilty using AI-generated art for my blog.

When AI tools first appeared, they felt like magic. A simple text prompt could conjure a unique, often beautiful image to accompany my writing. It offered a powerful new way to communicate, especially for me, who is not artistically inclined at all. But over time, the process felt hollow. Beyond the feeling of it being a shortcut, I found that AI often struggled to fully understand and capture the vision I had in mind for my cover art. It felt like the cover art was the result of me complaining at a machine until it produced something that was kind-of what I wanted and had the fewest obvious flaws.

I was doing this as a shortcut but it began to feel like a chore. So I'm changing how the covers are generated for my posts now.

I've always loved generative art: art created from code, algorithms, and a touch of randomness. It just felt right to spend time generating my own cover art with code, rather than asking an AI. For a blog about making things with code, it just makes sense that code would generate the cover art as well.

So, I built a small pipeline. It's a two-step process: [a Go program](https://github.com/sudorandom/kmcd.dev/blob/main/cmd/cover-art-generator/main.go) first generates a detailed, chaotic raster image, and then utilizes a Go library called [`primitive`](https://github.com/fogleman/primitive) to reinterpret that raster image into a stylized vector piece.

Here's the raw, detailed output from the Go script. This script generates random shapes of random sizes and using one of some color pallettes that I made. It also randomly generates connections to other shapes, because I like the "graph diagram" feeling that it gives. What is left is complex but it very much feels like a random smattering of shapes and lines.

{{< figure src="example-0.png" width="700px" >}}

And here's the final result after [`primitive`](https://github.com/fogleman/primitive) has done its work. `primitive` takes the source image and creates an abstract impression of the source image. There are many options as which shapes are 'available' to use, which provides makes the end results dynamic and varied. The result is a very cool looking abstract artwork:

{{< figure src="example-0.svg" width="700px" >}}

It is also worth noting that the resulting images from `primitive` are all SVGs, which can be much smaller than png, jpg or webp formats because these images are vector based.

This new process feels right, now. With this setup, I may decide to use primative on different kinds of source images if the post is about something visual, but for now I'm pretty happy with how this has turned out.

Here's a gallery of the art this process has helped create.

{{< gallery "example-*.svg" >}}
