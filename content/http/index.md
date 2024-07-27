---
description: "What version of HTTP are you connecting with?"
linktitle: ""
title: "What version of HTTP are you connecting with?"
titleIcon: "fa-globe"
cover: "cover.jpg"
subtitle: ""
devtoSkip: false
layout: http
---

Below you will find descriptions of each version with a list of significant features added. I also wrote a post about this page. Feel free to [read it here](/posts/http-tool/).

You can get this same information with the command line like so:

```shell
curl --http1.0 https://kmcd.dev/http/ -Is | grep x-kmcd
```

Here's an example using curl's HTTP/3 support (requires a [specific build of curl](https://curl.se/docs/http3.html))
```shell
$ curl --http3 https://kmcd.dev/http/ -Is | grep x-kmcd-http-request-version
x-kmcd-http-request-version: HTTP/3
```
