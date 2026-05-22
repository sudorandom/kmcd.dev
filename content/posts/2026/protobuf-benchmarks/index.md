---
title: "google.protobuf.Value considered harmful?"
date: "2026-06-15T10:00:00Z"
categories: ["article"]
tags: ["protobuf", "go", "performance", "json", "software-architecture"]
description: "The hidden performance cost of dynamic Protobuf in Go."
cover: "cover.svg"
images: ["/posts/protobuf-benchmarks/cover.svg"]
featuredalt: ""
featuredpath: "date"
slug: "protobuf-benchmarks"
type: "posts"
devtoSkip: true
draft: false
---

Migrating legacy JSON APIs to gRPC frequently stumbles over a common anti-pattern: unstructured, dynamic JSON fields (such as `metadata` or `extra_properties`) mapped directly into Protobuf using `google.protobuf.Value` or `google.protobuf.Struct`. 

While this looks like a convenient shortcut for representing arbitrary nested maps, wrapping JSON in `google.protobuf.Struct` degrades the performance benefits of binary serialization. 

Benchmarks show that using `google.protobuf.Value` or `Struct` for arbitrary JSON-like payloads can produce payloads larger than compact JSON and significantly slower to process.

---

## 1. The Benchmark Setup

We built a Go benchmark comparing standard JSON against various Protobuf strategies across three payload sizes:
* **Small:** A flat object with 4 fields (string ID, status boolean, age integer, score float).
* **Medium:** A nested user signup event containing an actor object, string tags, and a metadata map.
* **Large:** An array repeating the Medium object 100 times.

To handle arbitrary data, the dynamic Protobuf configurations rely on standard `structpb` definitions:

```protobuf
syntax = "proto3";
package event;

import "google/protobuf/struct.proto";

message EventEnvelope {
  string id = 1;
  int64 timestamp = 2;
  google.protobuf.Value payload = 3; // Dynamic payload field
}
```

**Note on methodology:** These benchmarks isolate the marshaling and unmarshaling steps using prebuilt structures. They do not include network transit times or the initial conversion cost of translating native Go types into `structpb.NewStruct()`, which would add even more overhead to the dynamic Protobuf numbers.

---

## 2. Benchmark Results

The benchmarks were executed under Go 1.26 on an Apple M1 Pro.

### Wire Size
While static binary structures are compact, dynamic Protobuf schemas drop the schema optimization completely. When representing arbitrary object structures, `google.protobuf.Struct` must serialize map entries, UTF-8 field names, and dynamically typed wrapper values on the wire. Furthermore, `google.protobuf.Value` stores numeric values using the JSON-oriented `number_value` representation (double precision floating point), preventing protobuf from using compact integer Varint encodings. As a result, payloads end up larger than compact JSON.

{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "Concrete (proto)",
      "google.protobuf.Value (proto)",
      "google.protobuf.Any (proto)",
      "Concrete (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)",
      "google.protobuf.Any (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Small Payload (Bytes)",
        "data": [25, 74, 74, 55, 55, 55, 111],
        "backgroundColor": "rgba(0, 224, 180, 0.75)",
        "borderColor": "rgba(0, 224, 180, 1)",
        "borderWidth": 1
      },
      {
        "label": "Medium Payload (Bytes)",
        "data": [162, 328, 212, 291, 293, 291, 349],
        "backgroundColor": "rgba(0, 191, 255, 0.75)",
        "borderColor": "rgba(0, 191, 255, 1)",
        "borderWidth": 1
      },
      {
        "label": "Large Payload (Bytes)",
        "data": [16500, 33104, 21200, 29201, 29412, 29201, 34900],
        "backgroundColor": "rgba(255, 107, 107, 0.75)",
        "borderColor": "rgba(255, 107, 107, 1)",
        "borderWidth": 1
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Serialized Data Size Comparison: lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" }
      }
    },
    "scales": {
      "x": {
        "type": "linear",
        "min": 0,
        "ticks": { "color": "#fff" }
      },
      "y": {
        "ticks": { "color": "#fff" }
      }
    }
  }
}
{{< /chart >}}

| Format / Config (Medium Payload) | Serialized Size | % of JSON (lower is better) |
| :--- | :---: | :---: |
| **Concrete (JSON)** | 291 B | 100.0% (Baseline) |
| **Concrete (proto)** / **VTProto** | 162 B | **55.7%** |
| **google.protobuf.Any (proto)** | 212 B | 72.9% |
| **google.protobuf.Value (proto)** | 328 B | 112.7% |
| **google.protobuf.Any (JSONProto)** | 349 B | 119.9% |

### Processing Throughput
Building and parsing schema-less Protobuf trees involves significant pointer-wrapping overhead, resulting in higher CPU usage and frequent heap allocations. Standard concrete Protobuf marshals almost instantly, and PlanetScale's reflection-free generator `VTProto` is the absolute fastest.

{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "VTProto",
      "Concrete (proto)",
      "google.protobuf.Any (proto)",
      "google.protobuf.Value (proto)",
      "Concrete (JSON)",
      "Struct (JSONv2)",
      "Map (JSONv2)",
      "Map (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Small Payload (ns/op)",
        "data": [23.4, 78.8, 222.2, 1686, 159.4, 266.5, 753.1, 616.0, 579.3, 2332],
        "backgroundColor": "rgba(0, 224, 180, 0.75)",
        "borderColor": "rgba(0, 224, 180, 1)",
        "borderWidth": 1
      },
      {
        "label": "Medium Payload (ns/op)",
        "data": [100.7, 278.0, 456.2, 5392, 496.2, 774.9, 1407, 1849, 2113, 7783],
        "backgroundColor": "rgba(0, 191, 255, 0.75)",
        "borderColor": "rgba(0, 191, 255, 1)",
        "borderWidth": 1
      },
      {
        "label": "Large Payload (ns/op)",
        "data": [7185, 23800, 48004, 523403, 40214, 61623, 82711, 179229, 223203, 763128],
        "backgroundColor": "rgba(255, 107, 107, 0.75)",
        "borderColor": "rgba(255, 107, 107, 1)",
        "borderWidth": 1
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Marshalling Performance: lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" }
      }
    },
    "scales": {
      "x": {
        "type": "linear",
        "min": 0,
        "ticks": { "color": "#fff" }
      },
      "y": {
        "ticks": { "color": "#fff" }
      }
    }
  }
}
{{< /chart >}}

| Benchmark (Medium Payload) | ns/op | Memory (B/op) | Allocations/op |
| :--- | :---: | :---: | :---: |
| **VTProto (Static Generated)** | **100.7 ns** | **176 B** | **1** |
| Concrete (proto) | 278.0 ns | 176 B | 1 |
| Concrete (JSON) | 496.2 ns | 464 B | 2 |
| google.protobuf.Any (proto) | 456.2 ns | 528 B | 4 |
| google.protobuf.Value (proto) | 5,392.0 ns | 2,959 B | 68 |
| google.protobuf.Value (JSONProto) | 7,783.0 ns | 4,977 B | 113 |

For a medium payload, standard static Protobuf is 19x faster than dynamic binary `Value` serialization. When evaluating unmarshalling, the gap widens further:

{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": [
      "VTProto",
      "Concrete (proto)",
      "google.protobuf.Any (proto)",
      "google.protobuf.Value (proto)",
      "Struct (JSONv2)",
      "Map (JSONv2)",
      "Concrete (JSON)",
      "Map (JSON)",
      "Concrete (JSONProto)",
      "google.protobuf.Value (JSONProto)"
    ],
    "datasets": [
      {
        "label": "Small Payload (ns/op)",
        "data": [25.1, 124.7, 251.7, 1363, 328.6, 768.3, 733.9, 1049, 889.1, 2579],
        "backgroundColor": "rgba(0, 224, 180, 0.75)",
        "borderColor": "rgba(0, 224, 180, 1)",
        "borderWidth": 1
      },
      {
        "label": "Medium Payload (ns/op)",
        "data": [305.5, 550.1, 696.1, 4571, 1079, 2084, 2862, 3285, 3613, 8355],
        "backgroundColor": "rgba(0, 191, 255, 0.75)",
        "borderColor": "rgba(0, 191, 255, 1)",
        "borderWidth": 1
      },
      {
        "label": "Large Payload (ns/op)",
        "data": [34241, 51941, 70174, 465091, 105672, 174445, 275634, 270555, 361495, 877593],
        "backgroundColor": "rgba(255, 107, 107, 0.75)",
        "borderColor": "rgba(255, 107, 107, 1)",
        "borderWidth": 1
      }
    ]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Unmarshalling Performance: lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" }
      }
    },
    "scales": {
      "x": {
        "type": "linear",
        "min": 0,
        "ticks": { "color": "#fff" }
      },
      "y": {
        "ticks": { "color": "#fff" }
      }
    }
  }
}
{{< /chart >}}

Dynamic binary parsing takes **4,571.0 ns** and requires **90 allocations**, compared to just **550.1 ns** and **15 allocations** for standard static Protobuf.

---

## 3. The Root Cause: Wire Overhead and Heap Allocations

The performance drop comes down to two specific architectural factors:

1. **Wire Format Overhead:** Statically compiled Protobuf omits field names entirely, sending only numeric field tags. A dynamic `Value` field has no static schema. To represent a simple pair like `{"age": 30}`, Protobuf must serialize a `MapEntry` message containing the string key `"age"` (5 bytes), a type wrapper, and an 8-byte double precision float. This brings the wire footprint up to **18 bytes**, compared to just **8 bytes** in compact JSON.
2. **Heap Allocations in Go:** In Go, representing dynamic, polymorphic variants requires nested pointers and interfaces. Every map item inside a `structpb.Struct` maps to a distinct `*structpb.Value` pointer containing an interface value. Parsing a large payload into this structural tree demands **over 6,700 individual heap allocations**, introducing substantial garbage collection pressure.

---

## 4. High-Performance Alternatives

Importantly, this problem is not inherent to runtime protobuf parsing itself. The real bottleneck is schema-less JSON-style polymorphism layered onto protobuf through `Struct` and `Value`.

If your system requires runtime schema flexibility, avoid `google.protobuf.Struct` for high-throughput paths and leverage these specific optimizations depending on your runtime requirements:

### Polymorphism: Use `google.protobuf.Any`
When data conforms to a known set of pre-compiled schemas, wrap the fields in an `Any` message. It records a clean `type_url` string alongside raw compiled binary bytes.
* **Pros:** Highly compact (212 bytes for a medium payload) and fast. Processing is roughly 11x faster than using generic values.
* **Cons:** Requires compile-time schema awareness for all incoming types.

### Runtime Schema Discovery: Use Buf's `hyperpb`
For pipelines that handle dynamic descriptors entirely at runtime (like schema registries or event gateways), Go's native `dynamicpb` is notoriously slow. Buf's `hyperpb` fixes this by compiling a message descriptor into dedicated Table-Driven Parser bytecode at application startup.

By combining this bytecode engine with a thread-local `hyperpb.Shared` arena pool, you can eliminate request-time heap churn:

```go
shared := new(hyperpb.Shared) // Instantiated once per goroutine

for _, payload := range incoming {
    msg := shared.NewMessage(mType) // Reuses the underlying memory arena
    _ = proto.Unmarshal(payload, msg)
    
    route(msg) // Read-only access pipeline
    shared.Free() // Recycles the arena back to the pool
}
```

On a large payload, `hyperpb + Shared` processes requests in **21,869 ns** with exactly **1 heap allocation**, outperforming even build-time generated static Protobuf code (**66,369 ns**, **1,509 allocations**).

{{< chart >}}
{
  "type": "bar",
  "data": {
    "labels": ["dynamicpb", "Concrete (proto)", "VTProto", "hyperpb", "hyperpb + Shared"],
    "datasets": [{
      "label": "ns/op",
      "data": [297153, 66369, 42462, 28902, 21869],
      "backgroundColor": "rgba(0, 191, 255, 0.75)",
      "borderColor": "rgba(0, 191, 255, 1)",
      "borderWidth": 1
    }]
  },
  "options": {
    "indexAxis": "y",
    "plugins": {
      "title": {
        "display": true,
        "text": "Dynamic Parsing Performance (Large Payload): lower is better",
        "color": "#fff"
      },
      "legend": {
        "labels": { "color": "#fff" }
      }
    },
    "scales": {
      "x": {
        "min": 0,
        "ticks": { "color": "#fff" }
      },
      "y": {
        "ticks": { "color": "#fff" }
      }
    }
  }
}
{{< /chart >}}

---

## 5. When google.protobuf.Struct Is Still Reasonable

`Struct` remains useful and entirely appropriate for:

* low-throughput administrative APIs
* debugging endpoints
* rapidly evolving schemas
* plugin metadata
* cross-language extensibility layers

The performance problems only emerge when these dynamic trees sit directly on hot-path production traffic.

---

## 6. Recommendations

1. **Stable Schemas:** Commit to first-class, statically typed fields whenever possible. It's worth it.
2. **Flat Attributes:** If your metadata is strictly flat key-value strings (like HTTP headers or tags), use a native `map<string, string>`. It converts cleanly to a native Go map without pointer wrapping.
3. **Opaque JSON Packaging:** If the payload is complex, nested, and truly arbitrary, bypass the Protobuf wrapper completely. Store the raw data as an opaque `string` or `bytes` field directly in the message template:

```protobuf
message UserEvent {
  string event_id = 1;
  int64 timestamp = 2;
  string raw_metadata_json = 3; // Avoids structural parsing overhead during transit
}
```

This lets your edge nodes route the packet instantly without parsing overhead. Downstream consumer services can then extract and decode the payload cleanly into native Go structures using optimized JSON parsers only when necessary.

<script>
(function() {
    function enhanceChart(chart) {
        if (chart.__enhanced) return;
        
        const datasets = chart.data.datasets;
        if (!datasets || datasets.length <= 1) return;
        
        chart.__enhanced = true;

        const originalLabels = [...chart.data.labels];
        const originalDatasets = chart.data.datasets.map(ds => ({
            ...ds,
            data: [...ds.data]
        }));

        function applyFilter(filterIndex) {
            if (filterIndex === null) {
                // Restore original labels and datasets
                chart.data.labels = [...originalLabels];
                chart.data.datasets.forEach((ds, dsIndex) => {
                    ds.data = [...originalDatasets[dsIndex].data];
                    if (typeof chart.setDatasetVisibility === 'function') {
                        chart.setDatasetVisibility(dsIndex, true);
                    } else {
                        ds.hidden = false;
                    }
                });
            } else {
                // Zipping labels with all values
                const zipped = originalLabels.map((label, itemIndex) => {
                    return {
                        label: label,
                        values: originalDatasets.map(ds => ds.data[itemIndex])
                    };
                });

                // Sort ascending (lower is better)
                zipped.sort((a, b) => {
                    const valA = a.values[filterIndex];
                    const valB = b.values[filterIndex];
                    if (valA === undefined || valA === null) return 1;
                    if (valB === undefined || valB === null) return -1;
                    return valA - valB;
                });

                // Update labels
                chart.data.labels = zipped.map(item => item.label);

                // Update datasets data and visibility
                chart.data.datasets.forEach((ds, dsIndex) => {
                    ds.data = zipped.map(item => item.values[dsIndex]);
                    if (typeof chart.setDatasetVisibility === 'function') {
                        chart.setDatasetVisibility(dsIndex, dsIndex === filterIndex);
                    } else {
                        ds.hidden = (dsIndex !== filterIndex);
                    }
                });
            }
            chart.update();
        }

        // Find wrapper and insert controls
        const canvas = chart.canvas;
        const wrapper = canvas.closest('.chart-wrapper');
        if (wrapper) {
            const controlsDiv = document.createElement('div');
            controlsDiv.className = 'chart-controls';
            controlsDiv.style.display = 'flex';
            wrapper.insertBefore(controlsDiv, wrapper.firstChild);

            // 1. All Payloads Button
            const btnAll = document.createElement('button');
            btnAll.textContent = 'All Payloads';
            btnAll.className = 'active';
            btnAll.onclick = () => {
                setActiveButton(btnAll);
                applyFilter(null);
            };
            controlsDiv.appendChild(btnAll);

            // 2. Buttons for individual datasets
            datasets.forEach((ds, index) => {
                const btn = document.createElement('button');
                let label = ds.label || `Dataset ${index + 1}`;
                label = label.replace(/\s*\(.*\)/, '');
                btn.textContent = label;
                btn.onclick = () => {
                    setActiveButton(btn);
                    applyFilter(index);
                };
                controlsDiv.appendChild(btn);
            });

            function setActiveButton(activeBtn) {
                controlsDiv.querySelectorAll('button').forEach(btn => {
                    btn.classList.remove('active');
                });
                activeBtn.classList.add('active');
            }
        }
    }

    // Enhance any already-loaded charts
    if (window.activeCharts) {
        window.activeCharts.forEach(enhanceChart);
    }

    // Register callback for future loaded charts
    const originalOnChartInit = window.onChartInit;
    window.onChartInit = function(chart) {
        if (originalOnChartInit) {
            originalOnChartInit(chart);
        }
        enhanceChart(chart);
    };
})();
</script>
