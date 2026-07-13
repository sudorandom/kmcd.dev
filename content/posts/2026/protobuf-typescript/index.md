---
categories: ["article"]
tags: ["typescript", "protobuf", "javascript"]
date: "2026-09-09T10:00:00Z"
title: "Choosing the Right TypeScript Protocol Buffers Implementation"
description: "An in-depth technical comparison of the TypeScript Protobuf ecosystem. Evaluate Protobuf-ES (@bufbuild/protobuf), ts-proto, protobuf.js, and google-protobuf across code generation, module compliance, and modern Protobuf feature support like Editions and Oneofs."
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

Protocol Buffers offer a robust framework for schema-first API design, but the TypeScript ecosystem has historically been fragmented. Choosing the right generator and runtime library significantly impacts your application bundle size, developer experience, and runtime performance.

This article compares the major TypeScript implementations of Protocol Buffers available today to help you select the best fit for your stack.

## The Contenders

### 1. Protobuf-ES (`@bufbuild/protobuf`)
[`@bufbuild/protobuf`](https://github.com/bufbuild/protobuf-es) is the modern implementation developed by Buf and designed from the ground up for modern JavaScript and TypeScript environments. It is my default recommendation for greenfield projects.
* **API & Ergonomics:** It generates clean, plain JavaScript objects that conform to native TypeScript interfaces. Instead of heavy class constructors or getter/setter methods, you can work with fields directly using standard property access. Oneofs are natively supported as type-safe discriminated unions, which fits idiomatic TypeScript perfectly.
* **Module Compliance:** Features full native ECMAScript Module (ESM) compliance by default, making it easily tree-shakable.
* **Codegen Flow:** Integrates directly with the standard `protoc` compiler or Buf plugins (`protoc-gen-es`), requiring minimal build steps.
* **Proto2 Extensions:** Fully supports typed proto2 extensions and registry APIs.
* **Tooling Friction:** Low. A single generator plugin and runtime package provide everything you need without separate compilation wrappers.

### 2. `ts-proto`
[`ts-proto`](https://github.com/stephenh/ts-proto) is a popular community utility that functions as a plugin for the standard `protoc` compiler. It focuses on generating clean, raw TypeScript interfaces rather than classes, making it a favorite for frontend developers who prefer functional patterns.
* **API & Ergonomics:** Generates pure, clean TypeScript interfaces with structural typing. However, by default, `ts-proto` maps `oneof` fields to flat, optional properties (e.g. `note?: string; image?: Image;`), which allows developers to write type-unsafe states (like setting both fields simultaneously). To get type-safe discriminated unions similar to Protobuf-ES, you must pass the `oneof=unions-value` configuration flag. Furthermore, because it avoids traditional class wrappers, operations like serialization and deserialization are handled through external helper methods (`ScaleMessage.encode`, `ScaleMessage.decode`) rather than methods bound to the objects.
* **Module Compliance:** Offers solid support for both ESM and CommonJS.
* **Codegen Flow:** Integrates as a `protoc` plugin (`protoc-gen-ts_proto`).
* **Proto2 Extensions:** Has basic support, but lacks complete extension registry coverage.
* **Tooling Friction:** High. It relies on a sprawling, brittle matrix of configuration flags. Getting the desired imports, typing defaults, and environment setups require stringing together a long, complex list of command-line arguments.

### 3. `protobuf.js`
[`protobuf.js`](https://github.com/protobufjs/protobuf.js) is the historical workhorse of the JavaScript ecosystem. While highly flexible, it represents a legacy era of JavaScript tooling.
* **API & Ergonomics:** Its generated code leans heavily toward legacy JavaScript patterns, constructing ES5-style functions and nested namespaces. Working with TypeScript requires a secondary step to generate definitions, and oneofs are represented as virtual property string mappings rather than type-safe discriminated unions.
* **Module Compliance:** Limited. The runtime supports CommonJS, AMD, and global scripts, but does not provide modern ESM structures natively.
* **Codegen Flow:** Requires a custom, multi-step pipeline using its own command-line utilities `pbjs` (to compile schemas to JS) and `pbts` (to compile the JS into TypeScript definition files).
* **Proto2 Extensions:** Poor support. The static generator breaks when compiling grouped proto2 extensions.
* **Tooling Friction:** High. The multi-step build flow, custom wrappers, and separate declaration files add considerable build-step complexity.

### 4. `google-protobuf`
[`google-protobuf`](https://github.com/protocolbuffers/protobuf-javascript) is the official implementation maintained by Google. It is primarily built to align with Java and C++ conventions rather than idiomatic JavaScript.
* **API & Ergonomics:** Ergonomics are notoriously poor for TypeScript developers. Working with messages requires calling Java-style getter and setter methods (e.g., `message.setName()`, `message.getName()`) instead of accessing properties directly. Oneofs are managed via a complex helper maze and `*Case()` enums.
* **Module Compliance:** Lacks native ESM support, export statements are CommonJS-only, and ES6 imports are not natively implemented.
* **Codegen Flow:** Uses the standard `protoc` compiler with Google's JS plugin.
* **Proto2 Extensions:** Supported, but relies on older, clunky extension APIs.
* **Tooling Friction:** High. Requires community-maintained typings (`@types/google-protobuf`) to work with TypeScript, and its class-heavy, CommonJS format makes it difficult to tree-shake or bundle.

## Architectural Philosophies

The contenders in this comparison fall into two main design camps:

* **Runtime-Driven Engines (Protobuf-ES and `google-protobuf`):** These libraries ship a single, centralized runtime engine. Instead of generating complete, duplicated serialization loops for every single message type, they generate lightweight schema metadata and let the runtime handle the heavy lifting. Protobuf-ES implements this with modern, native ESM modules and clean TypeScript interfaces, allowing bundlers like Vite, Rollup, or Webpack to easily tree-shake unused message definitions. By contrast, `google-protobuf` relies on a legacy, Java-like class getter/setter wrapping pattern and CommonJS, which is highly resistant to modern tree-shaking.
* **Direct Code Generators (`ts-proto` and `protobuf.js`):** These tools generate self-contained, hardcoded codec loops for each message. Because they output standalone functions or customized constructor classes for every message type, they bypass the need for a runtime interpreter. This gives them a very low starting bundle floor and high runtime throughput, but they pay for it with linear bundle growth at scale and major specification gaps.

## Conformance First: Why Correctness is Non-Negotiable

When choosing a Protocol Buffers implementation, **specification conformance is the most critical metric**. Protobuf is not just a serialization format; it is a strict, cross-language communication contract. Relying on a non-conforming library introduces silent, dangerous failure modes in production.

If your library does not strictly adhere to the specification, you risk:
* **Interoperability Failures:** A Go or Java backend might encode maps, unknown fields, or extensions that your TypeScript frontend silently ignores, corrupts, or fails to parse entirely.
* **Security & Validation Bugs:** Conformance tests verify bound checks, UTF-8 validation, and invalid inputs. Non-conforming parsers can crash or exhibit undefined behavior when exposed to malicious or malformed network payloads.
* **ProtoJSON Incompatibilities:** The mapping between JSON and Protobuf is strictly specified (e.g., camelCase conversion, 64-bit integer strings, enum representations). A non-conforming library can write JSON that other conforming services reject.

Buf runs and publishes a reproducible conformance test score using Google's official conformance runner. Here is how the contenders stack up:

| Implementation | Highest Edition Tested | Required Failures | Recommended Failures |
| :--- | :---: | ---: | ---: |
| **Protobuf-ES** | 2024 | **0** | **12** |
| **protobuf-ts** | proto3 | **6** | **7** |
| **google-protobuf** | 2023 | **1,169** | **389** |
| **ts-proto** | proto3 | **751** | **613** |
| **protobuf.js** | 2024 | **1,847** | **579** |

Protobuf-ES is the only runtime in this list that passes **100% of the required conformance tests** for the latest Edition 2024. While libraries like `ts-proto` and `protobuf.js` perform well in isolated tests, they do so by cutting corners on specification coverage (JSON serialization quirks, presence checks, and oneof edge cases).

### ProtoJSON, Reflection, and the Network Tab DX

While binary Protobuf is optimal for backend-to-backend communication, frontend developers heavily favor **ProtoJSON** in web applications. Utilizing JSON over the wire—such as ConnectRPC's JSON transport—allows developers to inspect API requests and responses directly within the browser's native **Network Tab** without installing custom decoding tools or browser extensions.

However, ProtoJSON is deceptively difficult to implement correctly. It relies on **runtime reflection** to translate between schema descriptors and JSON keys (handling camelCase conversion, serializing 64-bit integers as strings to prevent JavaScript float precision loss, and decoding dynamic wrappers like `google.protobuf.Struct` or `google.protobuf.Value`).

This is where the architectural difference between runtimes becomes a critical developer experience factor:
* **Protobuf-ES** retains schema descriptors and reflection metadata. This allows its runtime engine to handle ProtoJSON dynamically and pass 100% of the specification checks out of the box.
* **ts-proto** completely strips out reflection descriptors to optimize for initial bundle size. As a result, its ProtoJSON capability is heavily compromised, rendering it incapable of parsing dynamic fields or conforming to strict ProtoJSON specs.

---

## Bundle Size & Scaling Behavior

In frontend web development, single-schema measurements do not tell the whole story. The architectural approach of each library determines how it scales as your application grows.

Below is the scaling behavior of these architectures as the schema footprint expands (note the logarithmic scale on the y-axis, which is required to display both the low-end crossover and the multi-megabyte overhead of other engines on the same plot):

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
        "data": [49.59, 69.78, 95.00, 145.51, 297.04, 549.58, 2573.80, 5104.09],
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
        "text": "Bundle Size Scaling Behavior up to 1000 Message Types: lower is better",
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

<details>
<summary><b>Show complete scaling data table</b></summary>

| Message Types | Protobuf-ES | `ts-proto` | `protobuf.js` | `google-protobuf` |
| :--- | ---: | ---: | ---: | ---: |
| **1** | **63.05 KiB** | 11.86 KiB | 49.59 KiB | 241.99 KiB |
| **5** | **63.75 KiB** | 18.52 KiB | 69.78 KiB | 257.62 KiB |
| **10** | **64.63 KiB** | 26.86 KiB | 95.00 KiB | 277.19 KiB |
| **20** | **66.39 KiB** | 43.54 KiB | 145.51 KiB | 316.61 KiB |
| **50** | **71.70 KiB** | 93.61 KiB | 297.04 KiB | 434.85 KiB |
| **100** | **80.54 KiB** | 177.06 KiB | 549.58 KiB | 631.95 KiB |
| **500** | **152.15 KiB** | 844.64 KiB | 2,573.80 KiB | 2,221.80 KiB |
| **1000** | **241.67 KiB** | 1,679.11 KiB | 5,104.09 KiB | 4,209.13 KiB |

</details>

The scaling behavior reveals a fundamental architectural divide in how these libraries are constructed.

Protobuf-ES utilizes a shared-runtime approach where the encoding and decoding engine is bundled inside the library itself, leaving generated files as lightweight schema descriptors. This results in an incredibly flat scaling curve, adding only **~0.18 KiB** per message type. Even when scaling up to 1,000 message types, the entire bundled output remains a lean **241.67 KiB**.

By contrast, `ts-proto`, `google-protobuf`, and `protobuf.js` generate complete, self-contained control loops, ES5 constructors, helper validation routines, and object converters (`toObject`/`fromObject`) for every single message. Because this logic is duplicated for each type rather than shared in a runtime, their bundle sizes scale linearly:
* **`ts-proto`** adds **~1.67 KiB** per message type, starting as the lightest choice but swelling to **1.64 MiB** at 1,000 types. The crossover point where Protobuf-ES becomes lighter than `ts-proto` occurs around **30 message types**.
* **`google-protobuf`** suffers from verbose class and getter/setter boilerplate, adding **~3.97 KiB** per message and ballooning to **4.11 MiB** at scale. Protobuf-ES becomes more compact than `google-protobuf` at any footprint beyond just **5 message types**.
* **`protobuf.js`** exhibits the steepest growth at **~5.05 KiB** per message, culminating in a **4.98 MiB** bundle for 1,000 message types.

For large microservices or monorepos, selecting an engine with inline code generation can quietly introduce megabytes of JavaScript to your client bundle.

---

## Browser Runtime Performance

I evaluated the performance of these libraries in a headless Chromium page (Playwright) using 20 samples of 20,000 operations (following a 5,000-iteration warmup). Each operation processes a 294-byte message containing scalar types, nested messages, repeated fields, maps, and oneofs.

Here is how the Contenders compare in encode and decode throughput:

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
      "google-protobuf"
    ],
    "datasets": [
      {
        "label": "Encode Throughput (ops/s)",
        "data": [
          49517,
          54720,
          980392,
          73760
        ],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(135, 206, 250, 1)",
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
      "google-protobuf"
    ],
    "datasets": [
      {
        "label": "Decode Throughput (ops/s)",
        "data": [
          210526,
          496278,
          900901,
          303260
        ],
        "backgroundColor": [
          "rgba(0, 191, 255, 0.75)",
          "rgba(255, 165, 0, 0.75)",
          "rgba(135, 206, 250, 0.75)",
          "rgba(186, 85, 211, 0.75)"
        ],
        "borderColor": [
          "rgba(0, 191, 255, 1)",
          "rgba(255, 165, 0, 1)",
          "rgba(135, 206, 250, 1)",
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

| Implementation | Encode (ops/s) | Decode (ops/s) | Wire size |
| :--- | ---: | ---: | ---: |
| **Protobuf-ES** | 49,517 | 210,526 | 294 B |
| **ts-proto** | 54,720 | 496,278 | 294 B |
| **protobuf.js** | 980,392 | 900,901 | 294 B |
| **google-protobuf** | 73,760 | 303,260 | 294 B |

</details>

The performance numbers reflect a direct correlation with the underlying runtime designs.

The standout throughput leader is `protobuf.js`, which is engineered specifically for raw speed. It generates custom, highly optimized codecs at runtime using dynamic JIT compilation (`new Function`) and employs a zero-allocation buffer pooling strategy. While this makes it incredibly fast, the use of `eval`-based JIT compilation prevents it from running in strict browser environments with Content Security Policies (CSPs) that ban `unsafe-eval`.

For the other contenders, decoding performance highlights the cost of Protobuf-ES's reflection layer. Because `ts-proto` compiles direct property assignments into static files, V8 can optimize the object properties instantly, yielding a 2.3x speed advantage in decode throughput over Protobuf-ES's descriptor-driven parser.

On the encoding side, however, both Protobuf-ES and `ts-proto` share the same core `fork().join()` writer architecture under the hood. For every nested sub-message, map, or packed repeated field, the writer spins up a temporary buffer, serializes the sub-component, and then merges it back into the parent array. This process creates high garbage collection (GC) pressure from continuous array allocations, which is why their encoding speeds are comparable and substantially slower than the pooled approach in `protobuf.js`.



## Verdict

For modern web applications and TypeScript projects, **Protobuf-ES is the clear, definitive recommendation**.

While it does not win runtime throughput microbenchmarks, it wins where it matters most: **correctness, specification compliance, and real-world bundle size efficiency**.

* **Stellar Conformance:** Protobuf-ES is the only engine tested that achieves 100% correctness (0 required failures) on the latest specifications. A fast serialization library that quietly drops fields, encodes oneofs incorrectly, or fails to parse standard formats is a liability in production.
* **Massive Scaling Advantages:** Although libraries like `ts-proto` offer a smaller initial foot-in-the-door size for a single message, they scale linearly. For large applications, microservice architectures, or monorepos with hundreds of message types, `ts-proto`, `protobuf.js`, and `google-protobuf` swell into megabytes of compiled JavaScript code. Protobuf-ES's shared-runtime architecture scales flatly, saving your users from downloading massive JavaScript bundles.

If you are building a small project and bundle size is your single absolute bottleneck, you might consider **ts-proto**, provided you are willing to audit its conformance limitations. Reach for **protobuf.js** if your stack requires dynamic runtime schema parsing, and treat **google-protobuf** as a legacy fallback only. But for any scale-oriented production environment, Protobuf-ES represents the most correct, maintainable, and client-efficient choice.
