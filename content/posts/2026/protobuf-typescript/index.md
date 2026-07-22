---
categories: ["article"]
tags: ["typescript", "protobuf", "javascript"]
date: "2026-09-09T10:00:00Z"
title: "Comparing TypeScript Protobuf Libraries in 2026"
description: "Benchmarking Protobuf-ES, ts-proto, protobuf.js, and google-protobuf in the browser reveals sharp trade-offs between speed, bundle size, and spec compliance."
cover: "cover.png"
images: ["/posts/protobuf-typescript/cover.png"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
slug: "protobuf-typescript"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/protobuf-typescript/
draft: true
---

I expected the newer TypeScript Protobuf libraries to perform similarly once bundled. Instead, the results split sharply between runtime speed, bundle-size scaling, and specification support. `protobuf.js` was dramatically faster than everything else, while Protobuf-ES had the strongest conformance results and the slowest bundle growth as schemas expanded.

The results ultimately pushed me toward Protobuf-ES as the best default for new browser applications, although the other libraries still win in specific situations.

---

## Generated API Comparison

Before looking at benchmarks, it is worth comparing the code each generator expects you to write.

### Implementations at a Glance

| Library / Mode | API Style | Module Format | Oneof Representation | Runtime Descriptors |
| :--- | :--- | :--- | :--- | :--- |
| **Protobuf-ES** | Plain objects (`create`) + functions | Native ESM | Type-safe discriminated union (`case`/`value`) | Yes |
| **`ts-proto`** | Interfaces + static methods (`User.create`) | ESM / CommonJS | Flat optional fields (or discriminated union via option) | No |
| **`protobuf.js` (Static)** | Plain objects + static methods | ESM (via `--wrap`) / CommonJS | Property assignment + active case string property | No |
| **`google-protobuf`** | Getter / Setter class instances (`new User()`) | CommonJS only | Enums and individual getters (`hasNote()`, `getNote()`) | Limited / Legacy |

### Code Examples

#### Protobuf-ES (`@bufbuild/protobuf` v2.12.1)

```typescript
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { UserSchema } from "./gen/protobuf-es/user_pb.js";

// Message construction via helper
const user = create(UserSchema, {
  id: 42,
  name: "Alice",
  payload: { case: "note", value: "hello" }
});

const bytes = toBinary(UserSchema, user);
const decoded = fromBinary(UserSchema, bytes);

// Discriminated union narrowing
if (decoded.payload.case === "note") {
  console.log(decoded.payload.value.toUpperCase());
}
```

Protobuf-ES v2 dropped class wrappers in favor of plain JavaScript objects and standalone functions. The discriminated union representation is especially nice for `oneof` fields. The catch is that operations such as `create`, `toBinary`, and `fromBinary` also need the message descriptor (`UserSchema`).

#### `ts-proto` (v2.12.0)

```typescript
import { User } from "./gen/ts-proto/user.js";

// Message construction
const user = User.create({
  id: 42,
  name: "Alice",
  payload: { $case: "note", value: "hello" }
});

const bytes = User.encode(user).finish();
const decoded = User.decode(bytes);

if (decoded.payload?.$case === "note") {
  console.log(decoded.payload.value.toUpperCase());
}
```

By default, `ts-proto` emits separate optional properties for `oneof` fields, allowing several alternatives to be populated at once. The example above uses `oneof=unions-value`, which generates a proper discriminated union instead. This is a good API, but you have to know to enable it.

#### `protobuf.js` (Static Mode v8.7.1)

```typescript
import pkg from "./gen/protobufjs/benchmark.cjs";
const { User } = pkg.benchmark;

const user = User.create({
  id: 42,
  name: "Alice",
  note: "hello"
});

const bytes = User.encode(user).finish();
const decoded = User.decode(bytes);

// Virtual property holds the active field name
if (decoded.payload === "note") {
  console.log(decoded.note.toUpperCase());
}
```

Static `protobuf.js` generates JavaScript codecs alongside TypeScript declarations rather than native TypeScript source. Its message API is straightforward, but `oneof` fields are represented indirectly: the value lives in a normal property while a separate string property identifies which variant is active.

*(Note: While static mode is used for our primary comparison, `protobuf.js` also offers a reflection mode that parses raw `.proto` definitions at runtime. Reflection mode is included in our browser performance benchmark because it uses a distinct JIT codec strategy).*

```typescript
import { User } from "./gen/google-protobuf/user_pb.cjs";

const user = new User();
user.setId(42);
user.setName("Alice");

const bytes = user.serializeBinary();
const decoded = User.deserializeBinary(bytes);
```

`google-protobuf` replicates Java and C++ getter/setter patterns in JavaScript. It lacks native ESM exports, handles `oneof` checks via numeric case enums, and feels out of place in a modern TypeScript codebase.

---

## Conformance and Spec Support

The upstream Protobuf conformance suite (v28.2) evaluates how strictly each library enforces binary and ProtoJSON specifications.

| Tested Library | Syntax / Edition | Binary Failures | ProtoJSON Failures | Total Required Failures |
| :--- | :---: | :---: | :---: | :---: |
| **Protobuf-ES** | Edition 2024 | 0 | 0 | **0** |
| **protobuf.js** (Static) | Edition 2024 | 6 | 38 | **44** |
| **google-protobuf** | Edition 2023 | 14 | 32 | **46** |
| **`ts-proto`** | proto3 | 515 | 849 | **1,364** |

Each implementation was tested against the newest syntax it claims to support, so the totals also reflect differences in edition support rather than only bugs in shared proto3 behavior.

### Where the Failures Come From

The largest divide is not binary encoding performance but access to schema metadata. Protobuf-ES retains descriptors at runtime, which gives it enough information to handle `Any`, ProtoJSON field-name mapping, validation annotations (via `@bufbuild/protovalidate`), and newer edition behavior. It passed every required test across both binary and ProtoJSON suites.

`ts-proto` deliberately removes that metadata. That produces lightweight TypeScript interfaces without an accompanying reflection model, but it also makes some specification behavior impossible to implement generically at runtime. Its high failure count (515 binary, 849 ProtoJSON) is therefore less a collection of isolated bugs than a consequence of the library’s architecture.

`protobuf.js` and `google-protobuf` land between those extremes. Both handle the core binary format well (recording 44 and 46 total failures respectively), but their failures cluster around newer editions and ProtoJSON edge cases.

---

## Bundle-Size Scaling Behavior

For both benchmark sections, I generated the same schema across all four toolchains. Every implementation produced the same 294-byte wire payload, and browser decoding includes complete JavaScript object materialization rather than lazy field access.

The bundle-size results mostly come down to whether serialization logic lives in a shared runtime or is generated separately for every message type.

{{< tabs >}}
  {{< tab name="Minified Size" >}}
{{< chart >}}
{
  "type": "line",
  "data": {
    "labels": ["1", "5", "10", "20", "50", "100", "500", "1000"],
    "datasets": [
      {
        "label": "Protobuf-ES (Minified KiB)",
        "data": [63.05, 63.75, 64.63, 66.39, 71.70, 80.54, 152.15, 241.67],
        "borderColor": "rgba(0, 191, 255, 1)",
        "backgroundColor": "rgba(0, 191, 255, 0.05)",
        "tension": 0.1,
        "fill": false
      },
      {
        "label": "ts-proto (Minified KiB)",
        "data": [11.86, 18.52, 26.86, 43.54, 93.61, 177.06, 844.64, 1679.11],
        "borderColor": "rgba(255, 165, 0, 1)",
        "backgroundColor": "rgba(255, 165, 0, 0.05)",
        "tension": 0.1,
        "fill": false
      },
      {
        "label": "protobuf.js (Minified KiB)",
        "data": [49.60, 69.81, 95.06, 145.63, 297.33, 550.17, 2576.73, 5109.95],
        "borderColor": "rgba(135, 206, 250, 1)",
        "backgroundColor": "rgba(135, 206, 250, 0.05)",
        "tension": 0.1,
        "fill": false
      },
      {
        "label": "google-protobuf (Minified KiB)",
        "data": [241.99, 257.62, 277.19, 316.61, 434.85, 631.95, 2221.80, 4209.13],
        "borderColor": "rgba(186, 85, 211, 1)",
        "backgroundColor": "rgba(186, 85, 211, 0.05)",
        "tension": 0.1,
        "fill": false
      }
    ]
  },
  "options": {
    "plugins": {
      "title": {
        "display": true,
        "text": "Bundle Size Scaling Behavior (Minified Size): lower is better",
        "color": "#fff"
      }
    },
    "scales": {
      "x": {
        "ticks": { "color": "#fff" },
        "title": { "display": true, "text": "Number of Message Types", "color": "#fff" }
      },
      "y": {
        "type": "logarithmic",
        "min": 10,
        "ticks": { "color": "#fff" },
        "title": { "display": true, "text": "Bundle Size (KiB, Log Scale)", "color": "#fff" }
      }
    }
  }
}
{{< /chart >}}
  {{< /tab >}}
  {{< tab name="Gzipped Size" >}}
{{< chart >}}
{
  "type": "line",
  "data": {
    "labels": ["1", "5", "10", "20", "50", "100", "500", "1000"],
    "datasets": [
      {
        "label": "Protobuf-ES (Gzip KiB)",
        "data": [16.71, 16.98, 17.06, 17.17, 17.52, 18.10, 22.33, 26.19],
        "borderColor": "rgba(0, 191, 255, 1)",
        "backgroundColor": "rgba(0, 191, 255, 0.05)",
        "tension": 0.1,
        "fill": false
      },
      {
        "label": "ts-proto (Gzip KiB)",
        "data": [3.91, 4.07, 4.25, 4.57, 5.47, 6.84, 17.07, 29.10],
        "borderColor": "rgba(255, 165, 0, 1)",
        "backgroundColor": "rgba(255, 165, 0, 0.05)",
        "tension": 0.1,
        "fill": false
      },
      {
        "label": "protobuf.js (Gzip KiB)",
        "data": [13.79, 14.16, 14.58, 15.33, 17.50, 20.82, 43.61, 71.42],
        "borderColor": "rgba(135, 206, 250, 1)",
        "backgroundColor": "rgba(135, 206, 250, 0.05)",
        "tension": 0.1,
        "fill": false
      },
      {
        "label": "google-protobuf (Gzip KiB)",
        "data": [41.55, 42.13, 42.89, 44.28, 48.24, 54.09, 98.32, 152.84],
        "borderColor": "rgba(186, 85, 211, 1)",
        "backgroundColor": "rgba(186, 85, 211, 0.05)",
        "tension": 0.1,
        "fill": false
      }
    ]
  },
  "options": {
    "plugins": {
      "title": {
        "display": true,
        "text": "Bundle Size Scaling Behavior (Gzipped Size): lower is better",
        "color": "#fff"
      },
      "legend": {
        "display": false
      }
    },
    "scales": {
      "x": {
        "ticks": { "color": "#fff" },
        "title": { "display": true, "text": "Number of Message Types", "color": "#fff" }
      },
      "y": {
        "type": "logarithmic",
        "min": 1,
        "ticks": { "color": "#fff" },
        "title": { "display": true, "text": "Bundle Size (KiB, Log Scale)", "color": "#fff" }
      }
    }
  }
}
{{< /chart >}}
  {{< /tab >}}
{{< /tabs >}}

<details>
<summary><b>Show complete scaling data table</b></summary>

| Message Types | Protobuf-ES (Min / Gzip) | `ts-proto` (Min / Gzip) | `protobuf.js` (Min / Gzip) | `google-protobuf` (Min / Gzip) |
| :--- | :---: | :---: | :---: | :---: |
| **1** | 63.05 KiB / 16.71 KiB | **11.86 KiB / 3.91 KiB** | 49.60 KiB / 13.79 KiB | 241.99 KiB / 41.55 KiB |
| **5** | 63.75 KiB / 16.98 KiB | **18.52 KiB / 4.07 KiB** | 69.81 KiB / 14.16 KiB | 257.62 KiB / 42.13 KiB |
| **10** | 64.63 KiB / 17.06 KiB | **26.86 KiB / 4.25 KiB** | 95.06 KiB / 14.58 KiB | 277.19 KiB / 42.89 KiB |
| **20** | 66.39 KiB / 17.17 KiB | **43.54 KiB / 4.57 KiB** | 145.63 KiB / 15.33 KiB | 316.61 KiB / 44.28 KiB |
| **50** | **71.70 KiB** / 17.52 KiB | 93.61 KiB / **5.47 KiB** | 297.33 KiB / 17.50 KiB | 434.85 KiB / 48.24 KiB |
| **100** | **80.54 KiB** / 18.10 KiB | 177.06 KiB / **6.84 KiB** | 550.17 KiB / 20.82 KiB | 631.95 KiB / 54.09 KiB |
| **500** | **152.15 KiB** / 22.33 KiB | 844.64 KiB / **17.07 KiB** | 2,576.73 KiB / 43.61 KiB | 2,221.80 KiB / 98.32 KiB |
| **1000** | **241.67 KiB / 26.19 KiB** | 1,679.11 KiB / 29.10 KiB | 5,109.95 KiB / 71.42 KiB | 4,209.13 KiB / 152.84 KiB |

</details>

Protobuf-ES begins with a relatively large runtime (~63 KiB minified), but adding message types is cheap because generated files contain mostly descriptors rather than serialization logic. It adds only ~0.18 KiB per message type.

The other three libraries put substantially more codec logic into every generated message.

`ts-proto` starts at only 11.86 KiB, making it the clear winner for tiny schemas. That advantage shrinks quickly because each message adds roughly 1.67 KiB of generated codec logic. It crosses Protobuf-ES at about 35 message types and reaches 1.64 MiB at 1,000.

`protobuf.js` and `google-protobuf` scale even more aggressively. At 1,000 types, their minified bundles reach roughly 4.99 MiB and 4.11 MiB respectively.

The gzip results initially make `ts-proto` look much closer to Protobuf-ES than the minified output suggests (~29.10 KiB vs ~26.19 KiB at 1,000 messages). Because `ts-proto` generates repetitive codec structures, gzip compresses the text effectively for network transport. However, network transfer is only part of the cost; browser engines still have to parse, compile, and evaluate the full 1.64 MiB of uncompressed JavaScript, consuming memory and main-thread time.

---

## Browser Runtime Performance

Bundle size is only half of the comparison. The implementations also take very different approaches to encoding and decoding at runtime.

{{< tabs >}}
  {{< tab name="Encode Throughput" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Protobuf-ES",
      "ts-proto",
      "protobuf.js",
      "protobuf.js (Reflection)",
      "google-protobuf"
    ],
    "datasets": [
      {
        "label": "Encode Throughput (ops/s)",
        "data": [
          50176,
          52569,
          863931,
          851064,
          72886
        ],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(30, 144, 255, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(30, 144, 255, 1)",
          "rgba(186, 85, 211, 1)"
        ],
        "borderWidth": 1
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Encode Throughput (ops/s): higher is better",
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
  {{< tab name="Decode Throughput" >}}
{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Protobuf-ES",
      "ts-proto",
      "protobuf.js",
      "protobuf.js (Reflection)",
      "google-protobuf"
    ],
    "datasets": [
      {
        "label": "Decode Throughput (ops/s)",
        "data": [
          209534,
          498132,
          858369,
          911162,
          273224
        ],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(30, 144, 255, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(135, 206, 250, 1)",
          "rgba(30, 144, 255, 1)",
          "rgba(186, 85, 211, 1)"
        ],
        "borderWidth": 1
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Decode Throughput (ops/s): higher is better",
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

<details>
<summary><b>Show complete performance data table</b></summary>

| Implementation | Encode Throughput Median (Min - Max ops/s) | Decode Throughput Median (Min - Max ops/s) | Wire Size |
| :--- | :---: | :---: | :---: |
| **Protobuf-ES** | 50,176 (47,416 - 54,540) | 209,534 (163,532 - 212,993) | 294 B |
| **ts-proto** | 52,569 (47,015 - 66,823) | 498,132 (373,134 - 519,481) | 294 B |
| **protobuf.js** (Static) | 863,931 (823,045 - 877,193) | 858,369 (833,333 - 873,362) | 294 B |
| **protobuf.js** (Reflection) | 851,064 (806,452 - 877,193) | 911,162 (888,889 - 934,579) | 294 B |
| **google-protobuf** | 72,886 (63,032 - 93,897) | 273,224 (145,560 - 280,505) | 294 B |

</details>

In a typical request-driven web application, protobuf serialization is unlikely to be the bottleneck. Even Protobuf-ES, the slowest encoder in the test, executes over 50,000 serialization operations per second on a single thread—far more throughput than most browser applications consume while waiting on network round-trips or rendering UI updates.

However, when serialization really is on the hot path, the implementation details matter a lot:

The surprising result is `protobuf.js`. Both its static and reflection modes sustain roughly 850,000 operations per second, more than an order of magnitude above Protobuf-ES and `ts-proto` when encoding. Reflection mode generates specialized codecs with `new Function`, while static mode moves that work to build time and achieves nearly identical performance without requiring CSP `unsafe-eval`.

`ts-proto` is much more competitive when decoding, reaching roughly 498,000 operations per second by assigning fields directly into plain objects. Its encoding result, however, is almost identical to Protobuf-ES.

`google-protobuf` requires a caveat: its normal lazy-decoding behavior does not produce the same fully materialized object shape as the other libraries. Calling `.toObject()` makes the comparison fairer, but also reduces its measured throughput.

---

## Benchmark Setup and Reproducibility

I generated the same schema across all four toolchains, bundled each output with esbuild, and ran the resulting code in Chromium.

<details>
<summary><b>Show hardware, environment, and verification details</b></summary>

* **Hardware & OS:** Apple M1 Pro (8-core CPU, 16 GB RAM), macOS 15.1.
* **Execution Environments:** Node.js 22.x and headless Chromium 120.0.6099.28 (Playwright 1.61.1).
* **Library Versions:**
  * **Protobuf-ES** (`@bufbuild/protobuf`): `2.12.1`
  * **`ts-proto`**: `2.12.0`
  * **`protobuf.js`**: `8.7.1` (compiled in static-module mode)
  * **`google-protobuf`**: `4.0.2`
* **Bundling & Optimization:** `esbuild 0.28.1` targeting `chrome120` in production mode.
* **Benchmark Harness:** Warmup of 5,000 iterations followed by 20 samples of 20,000 operations per sample. Encoding measures total materialization (`toBinary()` or `.finish()`), while decoding measures complete JS object instantiation.
* **Correctness Verification:**
  1. Serialized payload verified at exactly 294 bytes across all generators.
  2. Cross-decoding verified across all contender outputs.
  3. Field content equality checked across nested structs, maps, oneofs, and repeated fields.

</details>

Benchmark adapter definitions are located in [`ts/adapters/`](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2026/protobuf-typescript/ts/adapters/), and the runner script is available at [`ts/scripts/benchmark.ts`](https://github.com/sudorandom/kmcd.dev/blob/main/content/posts/2026/protobuf-typescript/ts/scripts/benchmark.ts).

---

## Final Recommendations

After running these tests, Protobuf-ES is the library I would choose for a new browser application. It is not the fastest implementation, and it carries more initial runtime weight than `ts-proto`, but it has the best overall combination of API design, conformance, module support, and bundle-size scaling.

There are two meaningful exceptions. For a small, self-contained schema, `ts-proto` can save a noticeable amount of initial bundle weight. For applications processing protobuf messages continuously, such as high-frequency WebSocket streams or browser-side data pipelines, the throughput advantage of static `protobuf.js` is large enough to matter.

I would not start a new TypeScript application with `google-protobuf`. Its CommonJS output and getter/setter API are difficult to justify unless compatibility with an existing codebase requires it.
