---
categories: ["article"]
tags: ["grpc", "golang", "go", "benchmark", "performance", "protobuf"]
date: "2024-05-21"
description: "Let's so how fast gRPC can go in Go."
cover: "cover.jpg"
images: ["/posts/benchmarking-go-grpc/cover.jpg"]
featuredpath: "date"
title: "Benchmarking gRPC (golang)"
slug: "benchmarking-go-grpc"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/benchmarking-go-grpc
---

Hey everyone, as you know from my previous posts I'm a big fan of gRPC especially when working with Go. It's been my go-to tool for building remote procedure calls (RPCs) for a while now. It has been extremely reliable for providing high-performance RPC systems. Originally, there was only one choice: Google's `grpc-go` library. It came before HTTP/2 support landed in the Go standard library, and being from Google, the creators of gRPC, it seemed like the natural fit. But is it *actually* the best gRPC library for Go?

As time marches on, and with it, some concerns about `grpc-go` have emerged:

* **Experimental Features:** Many important features are labeled `experimental`, meaning they could change or even disappear in future updates. This can cause compatibility issues down the road.
* **Design Quirks:** Some design choices feel a bit odd. Because it doesn't integrate well with any standard HTTP middleware or tooling then it makes for a more fragmented series of libraries that are specific to gRPC. Furthermore, widely used features are often deprecated, moved and removed, breaking compatibility for existing projects. There's also been a few [missteps](https://github.com/grpc/grpc-go/commit/0f6ef0fbe51aa33d05a91d0fa87b28113b83f5a9) around API management.
* **Separate Server Dependency:** `grpc-go` requires a separate server instead of leveraging the standard Go HTTP server. This might not be ideal if you need to serve other functionalities alongside your gRPC endpoints, like OAuth or large file uploads.

## The lineup
Because of these reasons, I decided to explore some alternatives to grpc-go and if I require similar performance, I should make sure that the alternatives are in the same ballpark of performance. This post dives into benchmarks comparing the performance of three different ways to run a gRPC server in Go:

* **[grpc-go](https://github.com/grpc/grpc-go):** The official library from Google. See my implementation [here](https://github.com/sudorandom/go-grpc-bench/blob/v0.0.1/cmd/grpc-go/main.go).
* **[grpc-go-servehttp](https://pkg.go.dev/google.golang.org/grpc#Server.ServeHTTP):** A variant of `grpc-go` that allows using the standard Go HTTP server. This is useful when you want to add extra HTTP routes alongside the gRPC ones without having to manage another HTTP server instance on a different port. See my implementation [here](https://github.com/sudorandom/go-grpc-bench/blob/v0.0.1/cmd/grpc-go-servehttp/main.go).
* **[connectrpc](https://github.com/connectrpc/connect-go):** A third-party gRPC library that supports the Go HTTP server [(and gRPC-Web and the Connect protocol)](https://connectrpc.com/docs/multi-protocol/). I wrote about this [previously](/posts/connectrpc). See my implementation [here](https://github.com/sudorandom/go-grpc-bench/blob/v0.0.1/cmd/connectrpc/main.go).

Before diving into writing my own benchmarks, I wanted to see what exists today:
- The gRPC authors have published their own **[benchmark results](https://grpc.io/docs/guides/benchmarking/)**. In these results, it looks like grpc-go is among the top performers.
- **[LesnyRumcajs/grpc_bench](https://github.com/LesnyRumcajs/grpc_bench)** has a very good suite of gRPC frameworks in many different languages. The latest "official" results are [here](https://github.com/LesnyRumcajs/grpc_bench/discussions/441). This paints a significantly different picture where grpc-go is in the "middle of the pack".

I wanted to reproduce what happens in grpc_bench but with Go's CPU profiling enabled so that I can dig in and see what parts of the code are taking longer. After trying to bootstrap it to grpc_bench for a while I ended up just making my own repo with benchmark tests just for Go libraries.

*Aside:* Don't worry, before I did this I [contributed back](https://github.com/LesnyRumcajs/grpc_bench/pull/459) a change that updates all of the relevant libraries and the version of Go being used. I guessed that this might be some of the reason for the low ConnectRPC numbers because it was using a super old version from before they moved to `connectrpc.com/connect` as the canonical import.

In my new benchmark repo, I used [ghz](https://ghz.sh/) to run the benchmarks. I added the ability to capture pprof profiles while running the benchmark.


## Test One: The Empty Message Test
This scenario uses an empty message to gauge the overall performance of handling requests, without any of the overhead of protobuf parsing.
```json
{
    "proto": "proto/flex.proto",
    "call": "flex.FlexService/NormalRPC",
    "total": 101000,
    "skipFirst": 1000,
    "concurrency": 50,
    "connections": 20,
    "data": {},
    "max-duration": "300s",
    "host": "0.0.0.0:6660",
    "insecure": true
}
```
[(source)](https://github.com/sudorandom/go-grpc-bench/blob/v0.0.1/scenarios/empty.json)

### Results
The results are measured in Requests Per Second (RPS) which means that higher is better. This was run on an Intel NUC:
- Intel(R) Core(TM) i7-8705G CPU @ 3.10GHz
- 8GB of DDR4 RAM

This is a pretty outdated machine so if you run these benchmarks locally, you should see better performance.

{{< chart >}}
{
    type: 'bar',
    data: {
        labels: ['grpc-go', 'grpc-go-servehttp', 'connectrpc'],
        datasets: [{
            label: 'rps',
            data: [20303, 14216, 16272],
            backgroundColor: [
                'rgba(255, 99, 132, 0.5)',
                'rgba(54, 162, 235, 0.5)',
                'rgba(255, 206, 86, 0.5)'
            ],
            borderColor: [
                'rgba(255, 99, 132, 1)',
                'rgba(54, 162, 235, 1)',
                'rgba(255, 206, 86, 1)'
            ],
            borderWidth: 1
        }]
    },
    options: {
        indexAxis: 'y',
        plugins: {
            legend: {
                display: false
            },
            title: {
                display: true,
                text: 'Empty Protobuf Message (higher is better)'
            }
        },
        scaleShowValues: true,
        scales: {
            yAxes: [{
                ticks: {
                    autoSkip: false
                }
            }],
            xAxes: [{
                ticks: {
                    autoSkip: false
                }
            }]
        }
    }
}
{{< /chart >}}

This benchmark measured the performance of handling requests with an empty message. This means no actual data is being exchanged, so it focuses on the overhead of handling gRPC requests themselves.

 - **grpc-go was the clear winner**, processing over 20,000 requests per second.
 - **connectrpc followed closely behind** at around 16,000 requests per second.
 - **grpc-go (with ServeHTTP) lagged behind** at roughly 14,000 requests per second.

## Test Two: Something slightly more realistic
This scenario uses a larger message with most of the protobuf types to exercise all of the different code paths when marshaling and unmarshalling protobuf messages.
```json
{
    "proto": "proto/flex.proto",
    "call": "flex.FlexService/NormalRPC",
    "total": 101000,
    "skipFirst": 1000,
    "concurrency": 50,
    "connections": 20,
    "data": {
        "msg": {
            "doubleField": 123.4567,
            "floatField": 123.4567,
            "int32Field": 1234,
            "int64Field": 1234567,
            "uint32Field": 1234,
            "uint64Field": 1234567,
            "sint32Field": 1234,
            "sint64Field": 1234567,
            "fixed32Field": 1234,
            "fixed64Field": 1234567,
            "sfixed32Field": 1234,
            "sfixed64Field": 1234567,
            "boolField": true,
            "stringField": "hello world",
            "msgField": {},
            "repeatedMsgField": [{}, {}],
            "optionalMsgField": {}
        }
    },
    "max-duration": "300s",
    "host": "0.0.0.0:6660",
    "insecure": true
}
```
[(source)](https://github.com/sudorandom/go-grpc-bench/blob/v0.0.1/scenarios/complex.json)

In this scenario, I also tested with and without an alternative marshal/unmarshal implementation provided by [vtprotobuf](https://github.com/planetscale/vtprotobuf), which can further optimize performance. This was omitted from the last test because there shouldn't really be much performance difference when you're parsing an empty message.

### Results
Now here are the results of a test that uses a relatively complex message so we can see how protobuf parsing factors into the benchmarks:
{{< chart >}}
{
    type: 'bar',
    data: {
        labels: ['grpc-go', 'grpc-go (vtprotobuf)', 'grpc-go-servehttp', 'grpc-go-servehttp (vtprotobuf)', 'connectrpc', 'connectrpc (vtprotobuf)'],
        datasets: [{
            label: 'rps',
            data: [16836, 17450, 12403, 12836, 13963, 14079],
            backgroundColor: [
                'rgba(255, 99, 132, 0.5)',
                'rgba(255, 99, 132, 0.5)',
                'rgba(54, 162, 235, 0.5)',
                'rgba(54, 162, 235, 0.5)',
                'rgba(255, 206, 86, 0.5)',
                'rgba(255, 206, 86, 0.5)'
            ],
            borderColor: [
                'rgba(255, 99, 132, 1)',
                'rgba(255, 99, 132, 1)',
                'rgba(54, 162, 235, 1)',
                'rgba(54, 162, 235, 1)',
                'rgba(255, 206, 86, 1)',
                'rgba(255, 206, 86, 1)'
            ],
            borderWidth: 1
        }]
    },
    options: {
        indexAxis: 'y',
        plugins: {
            legend: {
                display: false
            },
            title: {
                display: true,
                text: 'More Complex Protobuf Message (higher is better)'
            }
        },
        scaleShowValues: true,
        scales: {
            yAxes: [{
                ticks: {
                    autoSkip: false
                }
            }],
            xAxes: [{
                ticks: {
                    autoSkip: false
                }
            }]
        }
    }
}
{{< /chart >}}

The goal of this benchmark was to introduce a larger message with various data types to better simulate real-world gRPC communication. It examines how well each library handles message marshaling and unmarshaling on top of the request processing.

 - Again, **grpc-go came out on top**, processing over 16,000 requests per second.
 - Even with a complex message, **connectrpc stayed competitive** at around 14,000 requests per second.
 - **grpc-go (with ServeHTTP) stayed in last place** with nearly 13,000.
 - Note that **vtprotobuf does help out a decent amount**. vtprotobuf is being used in a mode that has zero extra hand-written code. The plugin also allows another optimization using pools which does require code changes. I may add that setup to these benchmarks at some point.

## The Takeaway
If performance is the most important aspect of your application, stick with grpc-go. But if you want to add extra HTTP endpoints, add [gRPC-Web support](https://connectrpc.com/docs/multi-protocol) without an extra proxy, support for HTTP/1.1 or be able to [just use curl your gRPC endpoints](https://connectrpc.com/docs/curl-and-other-clients/) you might want to look into ConnectRPC.

I wouldn't recommend using [grpc-go with ServeHTTP](https://pkg.go.dev/google.golang.org/grpc#Server.ServeHTTP) because it's slower than ConnectRPC while offering fewer features. Maybe only if you're really entrenched into the grpc-go ecosystem should you consider using it.

## Open Questions and Next Steps
I am curious about exactly why supporting the `ServeHTTP` interface causes such a drastic difference in performance for grpc-go. There must be some cost to supporting the `ServeHTTP` interface that you don't incur when implementing the HTTP/2 server or some kind of optimization that you can't do in a general case that grpc-go can do. If you have any ideas about this, I'm happy to hear them.

I do have to give the standard disclaimer about benchmarks. This is an artificial benchmark that was run on my underpowered, hobbyist Intel NUC and while grpc-go performs well, the difference between it and the other methods is probably negligible for use cases that don't require peak performance.

The approach that ConnectRPC has with using ServeHTTP with a normal `http.Server` provided by the standard library means that it's likely to "just work" with http/3 [when it lands in the Go standard library](https://github.com/golang/go/issues/32204). This is exciting for these artificial benchmarks (and some real use cases) where creating new connections is a significant part of the overhead.

See the full benchmark source here at [github.com/sudorandom/go-grpc-bench](https://github.com/sudorandom/go-grpc-bench/tree/v0.0.1). The repo does contain CPU profile captures alongside the results which I was attempting to use to narrow down the performance differences, but I haven't found anything interesting there yet. If I succeed at finding anything interesting, I may post another update here!
