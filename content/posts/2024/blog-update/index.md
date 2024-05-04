---
categories: ["article"]
tags: ["blog", "update"]
date: "2024-05-14"
description: ""
cover: "cover.jpg"
images: ["/posts/blog-update/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Blog Update"
slug: "blog-update"
type: "posts"
devtoSkip: true
canonical_url: https://sudorandom.dev/posts/blog-update
mastodonID: ""
---

Hello! This will be a lighter update for this blog. I'm going to recap some of the changes I've made to it over the last couple of months, the technologies that power it, and my plans for the future.

## Schedule
I've been keeping up a one-post-a-week cadence. This has started pretty well because I was in a hyper-productive mode and built up a fairly sizable backlog but I'm now in a less productive phase. I'm uncertain that I can keep up the weekly post schedule but I will try. Some weeks in the future may be "softer" than others and I may resort to link posts, commenting about recent events, and other lower-effort ways of getting more words onto the website. This post is one of the lower-effort posts since it doesn't require any more research!

## Importing mastodon posts
Mastodon posts are automatically downloaded and re-hosted on this website, [here](https://sudorandom.dev/updates/). This serves a few purposes:

- At the bottom of every blog post, the corresponding mastodon post is referenced. I hope this will be a good way to point readers who want to make a comment on the article or to give me a thumbs up.
- Allows me to reference Mastodon posts inside of blog posts (without relying on an external service or iframes). I haven't used this yet, but it may come in handy.
- It also just acts as my personal mastodon archive. Because I have this history, I don't need some other mechanism to back up my public Mastodon posts. I strongly believe in content ownership, which is the whole reason I don't use many of the existing blog publishing platforms except for syndication.

## Prompt of the day
I've started posting [a daily writing prompt](https://sudorandom.dev/prompts/) every day on my Mastodon account. This is automatically posted from a combination of future-dated pages in Hugo, scheduled Github Actions and Echofeed. I feel like these prompts help me write SOMETHING every day. I typically just respond to the mastodon post. There's been an encouraging amount of participation from others with this. It was fun to come up with the first month's worth of questions.

## RSS
I've made some improvements to how RSS feeds are generated so that links and images use absolute paths in all cases, which allows you to see these images in your RSS feed if your reader supports that. It's strange how awful the "out of the box" experience is with Hugo and RSS. I feel like at least some of my issues should be handled automatically.

## Small Refinements
I originally used the [Hello Friend 4s3ti](https://github.com/coolapso/hugo-theme-hello-4s3ti) theme for my blog. I've slowly made small adjustments, moving more and more layouts away from the theme into the main blog section. At some point, I may copy over the rest of the resources and use a completely custom theme which will allow me to delete features I don't ever plan on using and more easily customize. But anyway, I like the style that I've settled on. It fits me.

### Projects Page
I updated the projects page to be more... clear. The image gallery format just wasn't working to show what I worked on. I'm not an artist and I should come to terms with that!

### Links Page
I have become a fan of the "small web" and I feel like link pages bring some of that small web feeling back.

# Technologies
## Cloudflare
As you may be able to tell from DNS, I use Cloudflare as a CDN, DNS, and firewall. Cloudflare has an incredibly generous free tier, which I use extensively.

## GitHub Pages
The actual website is hosted on Github Pages. Github also has a generous free tier here, but because of Cloudflare, I'm not sure it actually gets a lot of non-cached traffic. I think I could easily switch to using [Cloudflare R2](https://www.cloudflare.com/developer-platform/r2/) if I needed to. The killer feature for Github pages is having easy integration with Github Actions, which I use to build and deploy the website. I simply push to my main branch and the process is handled from there. I also rely on Github Actions to publish scheduled content for me and to import mastodon post updates for me.

## Hugo
I use [Hugo](https://gohugo.io/) for my static website generator. I prefer it due to the Go templating style that I was already very familiar with along with the ease of use. With Hugo, I don't need to download 100 different npm packages to build my website. It does have its quirks, though and some things I do get frustrated about.

## EchoFeed
[EchoFeed](https://echofeed.app/) is a new but welcome addition to my tech stack. This service consumes my RSS/Atom/jsonfeed feeds and makes corresponding posts to... somewhere. Currently, it supports Mastodon, bluesky, discord, Github, webmentions, webhooks, and a few others. Here's how I use the service:

- updating mastodon when there is a new post
- updating mastodon with a new prompt of the day
- updating the github repo with new mastodon posts from myself
