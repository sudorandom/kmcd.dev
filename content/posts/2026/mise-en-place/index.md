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

Dependency management is a perennial headache. Your local environment works perfectly until you set up a new laptop, and suddenly everything breaks due to version differences. This usually hits new hires the hardest since they are setting things up fresh and end up accidentally testing your codebase against brand new tool versions.

We have plenty of tools to solve this. I've been through `asdf`, `pyenv`, `nvm`, `gvm`, `bazelisk`, Homebrew bundles, and Dev Containers.

But they all fall short in ways that **[mise-en-place](https://mise.jdx.dev/)** (usually just called `mise`) doesn't. I've actually reached the point where nearly all my dev tooling is project-scoped. I barely install anything globally anymore.

## Getting rid of siloed version managers

Most of us started with `pyenv`, `nvm`, or `gvm`. They do the job, but they operate in silos. If your repository has a Python backend, a Node frontend, and some Go CLI helpers, you end up juggling three different version managers and configuration formats.

`mise` is polyglot. Every repository gets a single `mise.toml`. You can even scope specific versions to subdirectories. It gives you one file per context, eliminating the need for system-level packages.

```toml
[tools]
node = "20.10"
python = "3.12"
go = "1.23"
```

The moment you `cd` into a directory, `mise` activates the right versions instantly. If a nested folder has its own `mise.toml`, those versions automatically take over. This makes supporting monorepos, mixed stacks, or legacy services incredibly easy.

## Native GitHub releases

This is where `mise` heavily outshines older options like `asdf`. It isn't just a wrapper for language runtimes. It has a native GitHub backend.

If your project needs a CLI binary like `terraform`, `kubectl`, or some obscure linter, you don't have to pray someone wrote a plugin for it.

```toml
[tools]
"github:claudiodangelis/qrcp" = "latest"
"github:sharkdp/fd" = "v8.7.0"
```

It fetches the release artifacts, verifies them, and drops them into your path. It basically turns GitHub Releases into a native package manager.

## Handling NPM and Go binaries

A lot of utilities live in the awkward space of being project-scoped but acting like global commands. `mise` handles this cleanly using `npm:` and `go:` backends:

* **Go tools:** `"go:github.com/fullstorydev/grpcurl/cmd/grpcurl" = "latest"`
* **NPM-based linters:** `"npm:prettier" = "3.0"`

They exist when you are in the project folder and disappear when you leave. Your global path stays clean.

## Dev Containers are overkill for this

Dev containers have their place, particularly for massive multi-service architectures. But for day-to-day development, they are usually way more machinery than necessary. `mise` provides the main benefits with a fraction of the overhead:

* **No networking friction:** You are running on bare metal. No port forwarding and no Docker bridge networks to debug.
* **Native file performance:** Your IDE and compiler interact with the native filesystem. No virtiofs or gRPC-fuse lag.
* **Strict version pinning:** A `mise.lock` file guarantees everyone on the team runs the exact same binary hash. You get reproducibility without the conceptual baggage of containers or the chore of maintaining Docker images.

## The problem with `asdf`

I relied on `asdf` for a while. It was fine until it started randomly breaking. The tool resolution would get confused, and randomly a binary execution would revert to the system version instead of the pinned one. Once that trust is gone, it is hard to keep using a tool. I went looking for alternatives, found `mise`, and haven't had a single issue since.

Every binary runs the exact version you expect, every single time. It is predictable and fast.

## CI matching local

I also drop `mise` into my CI pipelines using a [GitHub Action](https://github.com/jdx/mise-action). It makes the remote tooling setup mirror my local environment perfectly.

## Wrapping up

`mise` is built in Rust and makes juggling multiple languages genuinely painless. I primarily work in Go right now, and I don't even have Go installed globally on my machine anymore.

It takes the friction out of environment setup. You just clone the repo, cd into it, and everything is exactly where it needs to be.

### Resources
* **[mise-en-place](https://mise.jdx.dev/)**
* [Github Action: jdx/mise-action](https://github.com/jdx/mise-action)
* [Mise: The Best Way to Manage Tool Versions](https://www.youtube.com/watch?v=eKJCnc0t8V0)
