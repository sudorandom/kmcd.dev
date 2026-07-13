import { writeFileSync, mkdirSync, readFileSync } from "node:fs";
import { execSync } from "node:child_process";
import { gzipSync } from "node:zlib";
import { build } from "esbuild";

const messageCounts = [1, 5, 10, 20, 50, 100];
const baseDir = new URL("..", import.meta.url).pathname;

interface SizeResult {
  messages: number;
  protobufEsMin: number;
  protobufEsGzip: number;
  tsProtoMin: number;
  tsProtoGzip: number;
}

const results: SizeResult[] = [];

console.log("Starting scale tests...");

for (const N of messageCounts) {
  console.log(`Generating files for N = ${N} messages...`);

  // 1. Generate proto/scale.proto
  let protoContent = `syntax = "proto3";\npackage scale;\n\nmessage SubMessage {\n  string sku = 1;\n  uint32 quantity = 2;\n  double price = 3;\n}\n\n`;
  for (let i = 1; i <= N; i++) {
    protoContent += `message ScaleMessage${i} {\n  uint32 id = 1;\n  string name = 2;\n  string email = 3;\n  repeated string tags = 4;\n  repeated SubMessage items = 5;\n  bool active = 6;\n}\n\n`;
  }
  writeFileSync(`${baseDir}/proto/scale.proto`, protoContent);

  // 2. Run buf generate
  execSync("pnpm exec buf generate", { cwd: baseDir });

  // 3. Write adapter/scale-protobuf-es.ts
  let esImports = `import { create } from "@bufbuild/protobuf";\n`;
  let esSchemaNames: string[] = [];
  for (let i = 1; i <= N; i++) {
    esImports += `import { ScaleMessage${i}Schema } from "../gen/protobuf-es/scale_pb.js";\n`;
    esSchemaNames.push(`ScaleMessage${i}Schema`);
  }
  const esAdapterContent = `${esImports}\nexport function run() {\n  return [\n    ${esSchemaNames.map(s => `create(${s})`).join(",\n    ")}\n  ];\n}\n`;
  writeFileSync(`${baseDir}/adapters/scale-protobuf-es.ts`, esAdapterContent);

  // 4. Write adapter/scale-ts-proto.ts
  let tsImports = ``;
  let tsNames: string[] = [];
  for (let i = 1; i <= N; i++) {
    tsImports += `import { ScaleMessage${i} } from "../gen/ts-proto/scale.js";\n`;
    tsNames.push(`ScaleMessage${i}`);
  }
  const tsAdapterContent = `${tsImports}\nexport function run() {\n  return [\n    ${tsNames.map(t => `${t}.create()`).join(",\n    ")}\n  ];\n}\n`;
  writeFileSync(`${baseDir}/adapters/scale-ts-proto.ts`, tsAdapterContent);

  // 5. Bundle with esbuild
  const bundle = async (adapterName: string) => {
    const outfile = `${baseDir}/dist/scale-${adapterName}.js`;
    await build({
      entryPoints: [`${baseDir}/adapters/scale-${adapterName}.ts`],
      outfile,
      bundle: true,
      minify: true,
      treeShaking: true,
      platform: "browser",
      target: ["chrome120"],
      format: "iife",
      globalName: `protobufScaleBench`,
      legalComments: "none",
    });
    const bytes = readFileSync(outfile);
    const gzipBytes = gzipSync(bytes, { level: 9 });
    return {
      min: bytes.byteLength,
      gzip: gzipBytes.byteLength
    };
  };

  const esSize = await bundle("protobuf-es");
  const tsSize = await bundle("ts-proto");

  results.push({
    messages: N,
    protobufEsMin: esSize.min,
    protobufEsGzip: esSize.gzip,
    tsProtoMin: tsSize.min,
    tsProtoGzip: tsSize.gzip
  });
}

// Format and print the results as Markdown
console.log("\nScale Test Results:\n");
console.log("| Messages | Protobuf-ES Minified | Protobuf-ES Min+Gzip | ts-proto Minified | ts-proto Min+Gzip |");
console.log("| :--- | ---: | ---: | ---: | ---: |");
for (const r of results) {
  console.log(`| ${r.messages} | ${(r.protobufEsMin / 1024).toFixed(2)} KiB | ${(r.protobufEsGzip / 1024).toFixed(2)} KiB | ${(r.tsProtoMin / 1024).toFixed(2)} KiB | ${(r.tsProtoGzip / 1024).toFixed(2)} KiB |`);
}

// Write the results to results/scale.json and results/scale.md
const resultsMD = [
  "| Messages | Protobuf-ES Minified | Protobuf-ES Min+Gzip | ts-proto Minified | ts-proto Min+Gzip |",
  "| :--- | ---: | ---: | ---: | ---: |",
  ...results.map(r => `| ${r.messages} | ${(r.protobufEsMin / 1024).toFixed(2)} KiB | ${(r.protobufEsGzip / 1024).toFixed(2)} KiB | ${(r.tsProtoMin / 1024).toFixed(2)} KiB | ${(r.tsProtoGzip / 1024).toFixed(2)} KiB |`),
  ""
].join("\n");

writeFileSync(`${baseDir}/results/scale.md`, resultsMD);
writeFileSync(`${baseDir}/results/scale.json`, JSON.stringify(results, null, 2));

console.log("\nScale test results saved to results/scale.md and results/scale.json\n");
