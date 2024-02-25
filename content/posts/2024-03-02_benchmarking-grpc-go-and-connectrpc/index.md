+++
categories = ["article"]
tags = ["networking", "grpc", "http", "tutorial", "protobuf", "connectrpc"]
date = "2024-03-02"
description = ""
cover = "cover.jpg"
images = ["/posts/benchmarking-grpc-go-vs-connectrpc/social.jpg"]
featured = ""
featuredalt = ""
featuredpath = "date"
linktitle = ""
title = "Benchmarking grpc-go vs ConnectRPC"
slug = "benchmarking-grpc-go-vs-connectrpc"
type = "posts"
draft = true
+++

If you've been reading the last few posts you might have realized that I'm a big fan of [ConnectRPC](https://connectrpc.com/) and how it extends the use-case of gRPC even further than [gRPC-Web](https://github.com/grpc/grpc-web). However, you do need to consider that [grpc-go](https://github.com/grpc/grpc-go) has existed for a long time and, when Google was making it, primarily focused on performance. So how does the up-and-coming gRPC implementation compare to the battle-tested veteran? That's what I'm going to find out.

## Speed
When we talk about "speed" for any kind of API we are referring to two separate topics: latency and throughput. Latency is how fast a single request can be handled. Throughput is how many requests can be handled.

### Test Setup

### Results

## CPU
### Test Setup

### Results

## Memory
### Test Setup

### Results


## Thoughts


Write out pprof bencharks on exit: CPU/memory
Render results as pretty SVGs or other visual (or interactive) formats
Note that I'm not (currently) trying to test protobuf encoding/decoding performance


## Testing gRPC server performance in Go:

Here's a strategy for testing two different Go implementations of gRPC servers in performance across latency, throughput, memory, and CPU:

**Preparation:**

1. **Define benchmarks:** Clearly outline the specific operations you want to benchmark. Focus on realistic use cases and represent typical server workload.
2. **Choose tools:**
    * **Load testing:** ghz ([https://ghz.sh/](https://ghz.sh/)) is a popular, lightweight HTTP/1 and gRPC load testing tool.
    * **Profiling:** Utilize tools like pprof (built-in to Go) or go tool pprof to capture detailed CPU profiles.
3. **Environment setup:**
    * Ensure consistent testing environment, hardware, OS, and Golang version.
    * Set up the infrastructure to run both servers and the load testing tool.
    * Prepare scripts or automation to manage test runs and data collection.

**Test Execution:**

1. **Warm-up phase:** Run each server with low load for a short period to stabilize performance.
2. **Latency measurement:**
    * Configure benchmarks focused on specific operations (e.g., RPC call latency).
    * Use ghz to send multiple concurrent requests and measure average response time.
    * Analyze the distribution of response times (percentiles) for insights into outliers.
3. **Throughput measurement:**
    * Configure benchmarks to simulate high concurrent load.
    * Use ghz to measure requests served per second (RPS) or bytes transferred per second (MBps).
    * Monitor server resource utilization during the test.
4. **Memory and CPU profiling:**
    * While under load, capture memory and CPU profiles for each server using appropriate tools.
    * Analyze profiles to identify memory allocations and hot spots in CPU usage.

**Data Analysis and Comparison:**

1. **Aggregate and visualize results:** Use tools like Grafana to visualize latency distributions, throughput trends, and resource utilization.
2. **Statistical analysis:** Compare metrics across implementations statistically, considering confidence intervals and potential outliers.
3. **Qualitative analysis:** Analyze profiling data to understand where each implementation allocates memory and utilizes CPU, looking for potential optimizations.

**Additional considerations:**

* **Test different client workloads:** Consider variations in request size, frequency, and concurrency to reflect real-world scenarios.
* **Isolate potential bottlenecks:** Look for bottlenecks in network, database access, or application logic, not just gRPC server implementation.
* **Repeat and validate:** Repeat tests multiple times for consistency and consider different deployment configurations.
* **Open-source your approach:** Consider contributing your testing methodology and results to the Go community for wider benefit.

Remember, the best strategy depends on your specific goals and priorities. Tailor the approach to accurately measure and compare the performance of your chosen gRPC server implementations.
