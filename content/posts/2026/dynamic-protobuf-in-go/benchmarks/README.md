# Protobuf vs JSON Benchmarks

This directory contains the benchmarking suite used to compare the performance and wire-size characteristics of various Protocol Buffers dynamic patterns (including `google.protobuf.Value`, `google.protobuf.Any`, and `hyperpb`) against standard Go structs, maps, and JSON.

## Prerequisites

- **Go**: 1.21 or later
- **Just**: (Optional) Command runner
- **Mise**: (Optional) Runtime executor

If you modify the Protobuf definitions, you will also need:
- **Buf CLI**: For generating Go code from Protobuf schemas
- **VTProto compiler**: For generating optimized `vtproto` code

## Running the Benchmarks

To execute the benchmarks and capture the output:

```bash
# Using Just
just dynamic-protobuf-in-go

# Or using Go directly
go test -bench=. -benchmem ./... | tee results.txt
```

To run a specific benchmark (e.g., small payloads):

```bash
go test -bench=BenchmarkMarshal_Small -benchmem ./...
```

To generate the protobuf definitions after changes:

```bash
./generate.sh
```

## Environment

The benchmarks in the article were executed under the following conditions:
- **Go version**: `go1.21` or later
- **OS/Arch**: `darwin/arm64` (Apple M1 Pro)
- **GOMAXPROCS**: 8
- **Benchmark iterations**: Evaluated using Go's built-in benchmarking system (`-bench=.`) which auto-tunes iterations to achieve statistical significance (generally at least 1,000,000 runs for fast paths, 100+ runs for slower paths).
- **Tooling**: Verified and cleaned using `benchstat`.
