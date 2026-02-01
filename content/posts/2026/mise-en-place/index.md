---
categories: ["article"]
tags: ["development", "tooling"]
date: "2026-03-19"
description: "Why I default to this tool for every project"
cover: "cover.svg"
images: ["/posts/mise-en-place/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Y'all are Sleeping on Mise-en-Place"
slug: "mise-en-place"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/mise-en-place/
draft: true
---

In the culinary world, **mise-en-place** ("everything in its place") is the sacred ritual of prepping every ingredient and tool before the heat even hits the pan. In software, we’ve historically been terrible at this. I’ve spent countless hours debugging the classic “it works on my machine” problem, thanks to conflicting Python, Node, Go, and a dozen random CLI tools cluttering my digital kitchen.

For a long time, the only “reliable” solution was to grab a heavy sledgehammer: **Dev Containers**. Containers solve isolation, sure, but they often come with networking headaches, sluggish file system performance, and the mental load of managing an entire guest OS just to run a linter.

That’s when I found **[mise-en-place](https://mise.jdx.dev/)**, or just `mise`. Finally, I could get my environment ready without the heavy overhead of containers. It gives me the predictability I need, every time, without slowing down my workflow.

## Beyond Language-Specific Managers

Most of us started with `pyenv`, `nvm`, or `gvm`. They work, but they’re silos. If your project has a Python backend, a Node frontend, and a few Go-based CLI helpers, now you’re juggling three version managers with three different configuration formats.

**`mise` is polyglot.** One `mise.toml` file at your project root handles everything:

```
[tools]
node = "20.10"
python = "3.12"
go = "1.23"
```

When you `cd` into the directory, `mise` activates the specified versions instantly. No more scattered `.nvmrc` or `v-env` scripts. Everything just works, like it should.

## The GitHub Backend

This is where `mise` really pulls ahead of older tools like `asdf`. It’s not just a wrapper for language runtimes—it has a native **GitHub Backend**.

If your project depends on a specific CLI tool—say `terraform`, `kubectl`, or a niche linter—you don’t need to hope there’s a plugin. Point `mise` directly at a GitHub repository:

```
[tools]
"github:claudiodangelis/qrcp" = "latest"
"github:sharkdp/fd" = "v8.7.0"
```

`mise` fetches the release artifacts, verifies them, and adds them to your path. In a way, it turns GitHub Releases into your personal tool pantry.

## Native Support for NPM and Go Dependencies

Many tools live in that awkward space of “project-scoped but globally needed.” `mise` handles this elegantly with backends like `npm:` and `go:`:

* **Go tools:** `"go:github.com/fullstorydev/grpcurl/cmd/grpcurl" = "latest"`
* **NPM-based linters:** `"npm:prettier" = "3.0"`

They’re available when you’re in the project folder and invisible when you’re not. No more polluting your global path.

## Why it Beats Dev Containers (For Most Things)

Dev containers are great for complex, multi-service environments. But for everyday development, they’re overkill. `mise` keeps things simple:

* **Zero Networking Friction:** You’re on bare metal—no ports to forward, no Docker bridge networks to troubleshoot.
* **Native File Performance:** Your IDE and compiler see the same files on the same disk. No virtiofs or gRPC-fuse lag.
* **Version Pinning:** The experimental `mise.lock` file ensures everyone on your team runs the exact same binary hash—reproducibility without container weight.

## `asdf` Nightmares

I’ve used `asdf` before, and honestly, it drove me crazy. Tool resolution would sometimes just… break. I could reproduce it: every 20th execution of a binary would revert to the system version instead of the one I expected. That’s unforgivable for me.

Digging into `asdf`’s resolution logic only made me realize this wasn’t a one-off bug—it was baked into how the tool works. That’s when I started looking for alternatives and discovered `mise`.

With `mise`, every binary runs the version you expect, every time. Predictable, fast, and reliable—enough to make me never look back.

## The Bottom Line

`mise` respects the “Digital Mise-en-Place” philosophy: your ingredients are prepped, your tools are sharp, and you can finally focus on coding. It’s fast, built in Rust, and finally makes juggling multiple languages and dependencies feel almost sane.

Stop hunting for the right Python or Node version. Put everything in its place and start coding.


**Want help migrating your current `tool-versions` or environment scripts into a `mise.toml`?**

[Mise: The Best Way to Manage Tool Versions](https://www.youtube.com/watch?v=eKJCnc0t8V0)
This video walks through how `mise` manages Node, Python, and Go, replacing scattered version managers with a single, reliable configuration.
