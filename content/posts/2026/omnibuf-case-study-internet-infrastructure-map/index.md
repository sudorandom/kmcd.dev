---
categories: ["article", "project", "dataviz"]
tags: ["protobuf", "omnibuf", "performance", "javascript", "dataviz", "internet-map", "cloudflare"]
keywords: ["omnibuf", "binary serialization", "protobuf alternative", "submarine cable map", "geojson optimization", "browser deserialization"]
series: ["Internet Map"]
date: "2026-12-08T10:00:00Z"
description: "Switching map.kmcd.dev from GeoJSON to OmniBuf significantly reduced memory and processing time."
cover: "cover.svg"
images: ["/posts/omnibuf-case-study-internet-infrastructure-map/cover.svg"]
linktitle: ""
title: "OmniBuf Case Study: Internet Infrastructure Map"
slug: "omnibuf-case-study-internet-infrastructure-map"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/omnibuf-case-study-internet-infrastructure-map/
---

If you've been following along, I've been improving the **[Internet infrastructure map](https://map.kmcd.dev)** quite a lot lately. I've added significantly more data to it and the loading times increased significantly. I was eager to improve these times because it was impacting search indexing and I knew that there was some portion of users that would immediately close a tab if it takes too long to load. Also, the required memory also shot up to over 11MB. That's a lot to ask for a single browser tab. So I set out to fix the problem, reaching for a technology that I knew well, Protobuf. But Protobuf wasn't the hero in this story.

## Protobuf Wasn't Faster

My first thought was just to use Protobuf. The binary `.pb` files came out significantly smaller than JSON, which significantly improved download speeds... but it actually didn't fix anything else. In fact, load times got *worse*. V8's `JSON.parse` is C++ and really fast, and trying to decode huge nested coordinate arrays in JavaScript with Protobuf just couldn't beat it. Memory usage also went up too because of all the intermediate objects it had to allocate. And when you consider the extra libraries and generated code that I added to the bundle, the payload size savings were mostly a wash, too. This was an utter failure. This type of data was just not good with Protobuf encoding.

{{< image src="pivot.png" width="400px" class="center" >}}

I was considering other options and sitting on the problem for a bit. I happened to be building **[OmniBuf](https://omnibuf.com)** around the same time. It's a binary wire format with explicit wire types and packed primitive arrays, and the whole goal was fast single-pass decoding that beats Protobuf for all environments, including Typescript on the browser.

{{< github-repo repo="omnibuf/omnibuf-es" description="TypeScript and JavaScript runtime for OmniBuf." >}}

So I figured I'd try it on the cable map data. After moving both files to `.obuf`, the downloads got a lot smaller and decoding in the browser was almost 4x faster than `JSON.parse`. I also was able to adjust some things to lower the memory usage, which also significantly improved things.

This is when I knew I was on to something with OmniBuf.

## Payload Size

Let's look at the file sizes first. I tested `.json` against `.obuf` raw, with `gzip -9`, with Cloudflare's edge Brotli (`q6`), and with static max-effort Brotli (`q11`):

| Dataset | Raw | Gzip (`-9`) | Brotli (`q6`) | Brotli (`q11`) |
| :--- | ---: | ---: | ---: | ---: |
| `all_cables.json` | 2,894.8 KiB | 868.3 KiB | 737.4 KiB | 510.1 KiB |
| `all_cables.obuf` | **1,089.9 KiB** | **469.0 KiB** | **435.4 KiB** | **383.3 KiB** |
| `globe-data.json` | 1,484.9 KiB | 277.5 KiB | 187.4 KiB | 154.4 KiB |
| `globe-data.obuf` | **223.1 KiB** | **104.5 KiB** | **93.9 KiB** | **84.2 KiB** |
| **Total `.json`** | 4,379.7 KiB | 1,145.8 KiB | 924.8 KiB | 664.5 KiB |
| **Total `.obuf`** | **1,313.0 KiB** | **573.5 KiB** | **529.3 KiB** | **467.5 KiB** |
| **Saved** | **-70.0%** | **-50.0%** | **-42.8%** | **-29.6%** |

{{< tabs >}}
  {{< tab name="Cloudflare Edge Brotli (q6)" >}}
{{< chart sort="false" unit="KiB" >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "all_cables (.json)",
      "all_cables (.obuf)",
      "globe-data (.json)",
      "globe-data (.obuf)"
    ],
    "datasets": [
      {
        "label": "Transfer Size (KiB)",
        "data": [
          737.4,
          435.4,
          187.4,
          93.9
        ],
        "backgroundColor": [
          "rgba(255, 159, 64, 0.75)",
          "rgba(0, 191, 255, 0.85)",
          "rgba(255, 159, 64, 0.75)",
          "rgba(0, 191, 255, 0.85)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Cloudflare Edge Brotli (q6) Payload Size: lower is better",
        "color": "#fff"
      },
      "legend": {
        "display": false
      }
    },
    "scales": {
      "x": {
        "type": "linear",
        "min": 0,
        "ticks": {
          "color": "#fff"
        }
      },
      "y": {
        "ticks": {
          "color": "#fff"
        }
      }
    }
  }
}
{{< /chart >}}
  {{< /tab >}}
  {{< tab name="Uncompressed" >}}
{{< chart sort="false" unit="KiB" >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "all_cables (.json)",
      "all_cables (.obuf)",
      "globe-data (.json)",
      "globe-data (.obuf)"
    ],
    "datasets": [
      {
        "label": "Uncompressed Size (KiB)",
        "data": [
          2894.8,
          1089.9,
          1484.9,
          223.1
        ],
        "backgroundColor": [
          "rgba(255, 159, 64, 0.75)",
          "rgba(0, 191, 255, 0.85)",
          "rgba(255, 159, 64, 0.75)",
          "rgba(0, 191, 255, 0.85)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Uncompressed Payload Size: lower is better",
        "color": "#fff"
      },
      "legend": {
        "display": false
      }
    },
    "scales": {
      "x": {
        "type": "linear",
        "min": 0,
        "ticks": {
          "color": "#fff"
        }
      },
      "y": {
        "ticks": {
          "color": "#fff"
        }
      }
    }
  }
}
{{< /chart >}}
  {{< /tab >}}
{{< /tabs >}}

Uncompressed, the payload is 70% smaller overall (and 85% smaller for `globe-data`).

## Processing Speed

Next I benchmarked decode speed and came up with these results:

| Dataset | JSON.parse | OmniBuf | Speedup |
| :--- | ---: | ---: | ---: |
| `all_cables` | 13.39 ms | **3.29 ms** | **4.07x** |
| `globe-data` | 4.01 ms | **1.24 ms** | **3.23x** |
| **Combined** | 17.40 ms | **4.53 ms** | **3.84x** |

{{< chart sort="false" unit="ms" >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "all_cables (JSON.parse)",
      "all_cables (OmniBuf)",
      "globe-data (JSON.parse)",
      "globe-data (OmniBuf)"
    ],
    "datasets": [
      {
        "label": "Decode Time (ms)",
        "data": [
          13.39,
          3.29,
          4.01,
          1.24
        ],
        "backgroundColor": [
          "rgba(255, 159, 64, 0.75)",
          "rgba(0, 191, 255, 0.85)",
          "rgba(255, 159, 64, 0.75)",
          "rgba(0, 191, 255, 0.85)"
        ],
        "borderWidth": 0
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Main-Thread Deserialization Time (ms): lower is better",
        "color": "#fff"
      },
      "legend": {
        "display": false
      }
    },
    "scales": {
      "x": {
        "type": "linear",
        "min": 0,
        "ticks": {
          "color": "#fff"
        }
      },
      "y": {
        "ticks": {
          "color": "#fff"
        }
      }
    }
  }
}
{{< /chart >}}

So we went from 17.4ms of processing time down to 4.5ms on my laptop. This sounds small, but this matters a lot more on mobile devices, so I typically throttle the CPU by 4x in Chrome DevTools to see how lower-powered devices might handle the changes. This, by the way, is probably what we need to do all the time. Anyway, here's the flamegraph comparison:

{{< image src="devtools-flamegraph.svg" width="840px" class="center" >}}

| Metric | GeoJSON | OmniBuf | Difference |
| :--- | ---: | ---: | ---: |
| Main-thread decode | 69.6 ms | **18.1 ms** | <span style="color: #16a34a; font-weight: 700;">-74%</span> |
| Long Task (> 50 ms) | +19.6 ms | **None** | <span style="color: #16a34a; font-weight: 700;">-51.5 ms</span> |
| Peak JS heap | 11.4 MB | **4.7 MB** | <span style="color: #16a34a; font-weight: 700;">-59%</span> |
| Retained JS heap | 10.8 MB | **4.7 MB** | <span style="color: #16a34a; font-weight: 700;">-56%</span> |
| Minor GC pauses | 1 | **0** | <span style="color: #16a34a; font-weight: 700;">Eliminated</span> |

So the parsing speed is not only improved with OmniBuf, but the memory usage significantly dropped as well. Also, we avoided a mid-load GC pause completely because of this.

Overall, this experiment was an incredibly success for OmniBuf. To me, it justified continuing to work on it.

Go check out the live map at **[map.kmcd.dev](https://map.kmcd.dev)** and the OmniBuf docs at **[omnibuf.com](https://omnibuf.com)**.
