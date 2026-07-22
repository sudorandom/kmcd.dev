import { mkdir, readFile, writeFile } from "node:fs/promises";
import { cpus, release, totalmem } from "node:os";
import { chromium } from "@playwright/test";
import type { BenchmarkAdapter } from "../shared/contract.js";

const adapters = ["protobuf-es", "ts-proto", "protobufjs", "protobufjs-reflection", "google-protobuf"] as const;
const iterations = Number.parseInt(process.env.BENCH_ITERATIONS ?? "20000", 10);
const samples = Number.parseInt(process.env.BENCH_SAMPLES ?? "20", 10);
const warmupIterations = Math.max(1000, Math.floor(iterations / 4));
const resultsDirectory = new URL("../results/", import.meta.url);
const packageJson = JSON.parse(await readFile(new URL("../package.json", import.meta.url), "utf8"));
await mkdir(resultsDirectory, { recursive: true });

const browser = await chromium.launch({ headless: true });
const runtimeResults = [];
let userAgent = "";

try {
  for (const adapter of adapters) {
    const page = await browser.newPage();
    await page.setContent("<!doctype html><meta charset=utf-8><title>protobuf benchmark</title>");
    await page.addScriptTag({ path: new URL(`../dist/${adapter}.js`, import.meta.url).pathname });
    const result = await page.evaluate(({ iterations, samples, warmupIterations }) => {
      const bench = (globalThis as typeof globalThis & { protobufBench: BenchmarkAdapter }).protobufBench;
      if (!bench.verify()) throw new Error(`${bench.library} failed round-trip verification`);
      bench.encode(warmupIterations);
      bench.decode(warmupIterations);

      const timings = (operation: "encode" | "decode") => {
        const values: number[] = [];
        let checksum = 0;
        for (let sample = 0; sample < samples; sample++) {
          const start = performance.now();
          checksum += bench[operation](iterations);
          values.push(performance.now() - start);
        }
        if (checksum === 0) throw new Error("Invalid zero checksum");
        return values;
      };

      return {
        library: bench.library,
        wireSize: bench.wireSize,
        encodeMs: timings("encode"),
        decodeMs: timings("decode"),
      };
    }, { iterations, samples, warmupIterations });
    userAgent ||= await page.evaluate(() => navigator.userAgent);
    runtimeResults.push({ adapter, ...result });
    await page.close();
  }
} finally {
  await browser.close();
}

const median = (values: number[]) => {
  const sorted = [...values].sort((left, right) => left - right);
  const middle = Math.floor(sorted.length / 2);
  return sorted.length % 2 === 0 ? (sorted[middle - 1] + sorted[middle]) / 2 : sorted[middle];
};
const opsPerSecond = (values: number[]) => iterations / (median(values) / 1000);
const wireSizes = new Set(runtimeResults.map(({ wireSize }) => wireSize));
if (wireSizes.size !== 1) throw new Error(`Implementations produced different wire sizes: ${[...wireSizes].join(", ")}`);
const summary = runtimeResults.map((result) => ({
  adapter: result.adapter,
  library: result.library,
  wireSize: result.wireSize,
  encodeOpsPerSecond: Math.round(opsPerSecond(result.encodeMs)),
  decodeOpsPerSecond: Math.round(opsPerSecond(result.decodeMs)),
}));
const output = {
  metadata: {
    iterations,
    samples,
    warmupIterations,
    userAgent,
    capturedAt: new Date().toISOString(),
    system: {
      platform: process.platform,
      architecture: process.arch,
      release: release(),
      cpu: cpus()[0]?.model ?? "unknown",
      logicalCpuCount: cpus().length,
      memoryBytes: totalmem(),
      node: process.version,
    },
    packages: {
      "@bufbuild/protobuf": packageJson.dependencies["@bufbuild/protobuf"],
      "ts-proto": packageJson.devDependencies["ts-proto"],
      protobufjs: packageJson.dependencies.protobufjs,
      "google-protobuf": packageJson.dependencies["google-protobuf"],
      esbuild: packageJson.devDependencies.esbuild,
      playwright: packageJson.devDependencies["@playwright/test"],
    },
  },
  summary,
  samples: runtimeResults,
};
await writeFile(new URL("runtime.json", resultsDirectory), `${JSON.stringify(output, null, 2)}\n`);
await writeFile(
  new URL("runtime.md", resultsDirectory),
  [
    "| Implementation | Encode (ops/s) | Decode (ops/s) | Wire size |",
    "| :--- | ---: | ---: | ---: |",
    ...summary.map((row) => `| ${row.library} | ${row.encodeOpsPerSecond.toLocaleString("en-US")} | ${row.decodeOpsPerSecond.toLocaleString("en-US")} | ${row.wireSize} B |`),
    "",
  ].join("\n"),
);

console.table(summary);
console.log(userAgent);
