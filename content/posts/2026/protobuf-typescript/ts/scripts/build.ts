import { gzipSync } from "node:zlib";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { build } from "esbuild";

const adapters = ["protobuf-es", "ts-proto", "protobufjs", "google-protobuf"] as const;
const outputDirectory = new URL("../dist/", import.meta.url);
const resultsDirectory = new URL("../results/", import.meta.url);

await Promise.all([mkdir(outputDirectory, { recursive: true }), mkdir(resultsDirectory, { recursive: true })]);

const results = [];
for (const adapter of adapters) {
  const outfile = new URL(`${adapter}.js`, outputDirectory).pathname;
  await build({
    entryPoints: [new URL(`../adapters/${adapter}.ts`, import.meta.url).pathname],
    outfile,
    bundle: true,
    minify: true,
    treeShaking: true,
    platform: "browser",
    target: ["chrome120"],
    format: "iife",
    globalName: "protobufBench",
    legalComments: "none",
  });
  const bundle = await readFile(outfile);
  results.push({ adapter, bytes: bundle.byteLength, gzipBytes: gzipSync(bundle, { level: 9 }).byteLength });
}

await writeFile(new URL("bundle-size.json", resultsDirectory), `${JSON.stringify(results, null, 2)}\n`);
await writeFile(
  new URL("bundle-size.md", resultsDirectory),
  [
    "| Implementation | Minified | Minified + gzip |",
    "| :--- | ---: | ---: |",
    ...results.map(({ adapter, bytes, gzipBytes }) => `| ${adapter} | ${(bytes / 1024).toFixed(2)} KiB | ${(gzipBytes / 1024).toFixed(2)} KiB |`),
    "",
  ].join("\n"),
);

console.table(results);
