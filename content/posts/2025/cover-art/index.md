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

So, I built a small pipeline. It's a two-step process: [a Go program](https://github.com/sudorandom/kmcd.dev/blob/main/cmd/cover-art-generator/main.go) first generates a raster image, and then a Go library called [`primitive`](https://github.com/fogleman/primitive) reinterprets it into a stylized vector piece.

My Go script generates random shapes of different sizes, pulling from a few color palettes I like. It also randomly draws lines connecting things, because I like the "graph diagram" feel it gives. It's all pretty simple stuff. For example, here's the little chunk of code that draws a star—nothing fancy, just a bit of trigonometry to connect the dots.

```go
func drawStar(dc *gg.Context, points, centerX, centerY, outerRadius float64) {
	innerRadius := outerRadius * 0.4
	dc.NewSubPath()
	for i := 0.0; i < points*2; i++ {
		r := outerRadius
		if int(i)%2 != 0 {
			r = innerRadius
		}
		angle := (math.Pi*2/(points*2))*i - math.Pi/2
		x := centerX + r*math.Cos(angle)
		y := centerY + r*math.Sin(angle)
		dc.LineTo(x, y)
	}
	dc.ClosePath()
}
```
*A snippet from the generator showing how a star shape is drawn.*

The output is a chaotic but structured smattering of shapes and lines.

{{< figure src="example-0.png" width="700px" >}}

To transform this into something with a more dynamic flare, I turned to `primitive`. It takes the source image and creates an abstract impression of it by layering simple shapes. With options to control what kinds of shapes are available, the end results are wonderfully varied. The result is a cool, abstract artwork with a clean, vector-based aesthetic.

{{< figure src="example-0.svg" width="700px" >}}

It's also worth noting that the resulting images from `primitive` are SVGs. Because they're vector-based, they are often much smaller than their raster counterparts (PNG, JPG) and scale perfectly to any size.

This new process feels right. With this setup, I may decide to use `primitive` on different kinds of source images if a post is about something visual, but for now I'm pretty happy with how this has turned out. It's more personal, and there's a fun element of surprise in seeing what the code comes up with each time. It feels less like a shortcut and more like my own little art machine.

Here's a gallery of the art this process has helped create.

{{< gallery "example-*.svg" >}}

[Click here for the full source code](https://github.com/sudorandom/kmcd.dev/blob/main/cmd/cover-art-generator/main.go).
