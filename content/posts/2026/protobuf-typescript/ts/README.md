# TypeScript Protobuf benchmarks

This directory reproduces the bundle-size and browser runtime measurements used by the article. All four implementations are generated from `proto/benchmark.proto` and exercise the same populated message.

## Run

```sh
mise install
pnpm install
pnpm exec playwright install chromium
pnpm bench
```

`pnpm generate` regenerates all implementations, `pnpm build` creates minified browser bundles and records raw/gzip sizes, and `pnpm benchmark` runs encode/decode samples in headless Chromium. Results are written to `results/` as both JSON (including raw samples and environment metadata) and Markdown tables.

You can adjust runtime duration with `BENCH_ITERATIONS` and `BENCH_SAMPLES`. Defaults are 20,000 operations per sample and 20 samples. Each implementation receives a warmup before measurements, and the reported throughput is calculated from the median sample.

Generated sources and bundles are ignored. The captured `results/` are committed so the article's tables have inspectable raw samples; rerunning the benchmark replaces them. Run it on an otherwise idle machine, and identify the CPU, OS, browser version, package versions, and benchmark settings when publishing new numbers.
