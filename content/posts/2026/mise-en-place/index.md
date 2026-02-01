---
categories: ["article"]
tags: ["development", "tooling", "programming", "devops", "productivity"]
date: "2026-03-19T10:00:00Z"
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
---

Managing dependencies for a software project can be frustrating. Your local setup works perfectly fine until you update a dev dependency and suddenly things are seriously broken. Or worse, they're subtly broken.

I've tried `asdf`, `pyenv`, `nvm`, `gvm`, and `bazelisk`. I've tried Homebrew bundles. I've tried Dev Containers.

They all fail in ways that **[mise-en-place](https://mise.jdx.dev/)** (or just `mise`) doesn't. I've gotten to the point where almost all of my dev tooling is strictly project-scoped and never installed system-wide.

## Ditching Multiple Version Managers

Most of us started with `pyenv`, `nvm`, or `gvm`. They work, but they're silos. If your project has a Python backend, a Node frontend, and a few Go-based CLI helpers, now you're juggling three version managers with three different configuration formats.

**`mise` is polyglot.** Each repo gets its own `mise.toml`, and you can even scope different versions per subdirectory if you need to. One file per context, no need for "system" packages or global installs.

```toml
[tools]
node = "20.10"
python = "3.12"
go = "1.23"
```

When you `cd` into the directory, `mise` activates the specified versions instantly. If a subdirectory contains its own `mise.toml`, those versions automatically take precedence when you enter it. This makes it trivial to support monorepos, mixed stacks, or legacy code without hacks.

## Pull Tools Straight from GitHub

This is where `mise` really pulls ahead of older tools like `asdf`. It's not just a wrapper for language runtimes; it actually features a native **GitHub Backend**.

If your project depends on a CLI like `terraform`, `kubectl`, or a niche linter, you don't need to hope there's a plugin.

```toml
[tools]
"github:claudiodangelis/qrcp" = "latest"
"github:sharkdp/fd" = "v8.7.0"
```

`mise` fetches the release artifacts, verifies them, and adds them to your path. It effectively turns GitHub Releases into your personal tool pantry.

## One File to Rule NPM and Go Tools

Many tools live in that awkward space of “project-scoped but globally needed.” `mise` handles this elegantly with backends like `npm:` and `go:`:

* **Go tools:** `"go:github.com/fullstorydev/grpcurl/cmd/grpcurl" = "latest"`
* **NPM-based linters:** `"npm:prettier" = "3.0"`

They're available when you're in the project folder and invisible when you're not. No more polluting your global path.

## Why I Ditched Dev Containers

Dev containers absolutely have their place, especially for complex, multi-service setups. For everyday development, though, they tend to be more machinery than I actually need. `mise` gives me most of the benefits with far less overhead and tooling complexity:

* **Zero Networking Friction:** You're on bare metal. This means there are no ports to forward, no Docker bridge networks to troubleshoot.
* **Native File Performance:** Your IDE and compiler see the same files on the same disk. No virtiofs or gRPC-fuse lag.
* **Version Pinning:** The `mise.lock` file ensures everyone on your team runs the exact same binary hash, providing reproducibility without the conceptual cost of containers or the overhead of managing and updating another Docker image.

## `asdf` Nightmares

I've used `asdf` before, and honestly, it was fine until it wasn't. Tool resolution would sometimes just break. It somehow got into a state where every 20th-ish execution of a binary would revert to the system version instead of the one I expected. After that, I just couldn't trust it anymore. I started looking for alternatives and discovered `mise`, which hasn't betrayed me like this.

With `mise`, every binary runs the version you expect, every time. Predictable, fast, and reliable, which is enough to make me never look back.

## Putting Everything in Its Place

`mise` respects the “Digital Mise-en-Place” philosophy: your ingredients are prepped, your tools are sharp, and you can finally focus on coding. It's fast, built in Rust, and finally makes juggling multiple languages and dependencies feel almost sane.

Stop hunting for the right Python or Node version. Put everything in its place and start coding.

### Resources
 - **[mise-en-place](https://mise.jdx.dev/)**
 - [Github Action: jdx/mise-action](https://github.com/jdx/mise-action)
 - [Mise: The Best Way to Manage Tool Versions](https://www.youtube.com/watch?v=eKJCnc0t8V0)
