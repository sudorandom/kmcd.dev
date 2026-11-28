---
title: "{{ replace .File.ContentBaseName "-" " " | title }}"
date: "{{ (time.AsTime .Date).UTC.Format "2006-01-02T15:04:05Z" }}"
categories: ["article"]
tags: []
keywords: []
description: ""
cover: "cover.svg"
images: ["/posts/{{ .File.ContentBaseName }}/cover.svg"]
slug: "{{ .File.ContentBaseName }}"
type: "posts"
canonical_url: https://kmcd.dev/posts/{{ .File.ContentBaseName }}/
devtoSkip: true
draft: true
---

