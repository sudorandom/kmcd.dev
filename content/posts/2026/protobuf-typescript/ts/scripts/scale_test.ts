import { writeFileSync, mkdirSync, readFileSync } from "node:fs";
import { execSync } from "node:child_process";
import { gzipSync } from "node:zlib";
import { build } from "esbuild";

const messageCounts = [1, 5, 10, 20, 50, 100, 500, 1000];
const baseDir = new URL("..", import.meta.url).pathname;

interface SizeResult {
  messages: number;
  protobufEsMin: number;
  protobufEsGzip: number;
  tsProtoMin: number;
  tsProtoGzip: number;
  protobufjsMin: number;
  protobufjsGzip: number;
  googleProtobufMin: number;
  googleProtobufGzip: number;
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

  // 3. Rename Google Protobuf if it exists
  try {
    execSync("mv gen/google-protobuf/scale_pb.js gen/google-protobuf/scale_pb.cjs", { cwd: baseDir, stdio: "ignore" });
  } catch (e) {
    // Ignore error if it's already renamed or not generated
  }

  // 4. Generate protobufjs static module
  mkdirSync(`${baseDir}/gen/protobufjs`, { recursive: true });
  execSync("pnpm exec pbjs -t static-module -w commonjs -o gen/protobufjs/scale.cjs proto/scale.proto", { cwd: baseDir });
  execSync("pnpm exec pbts -o gen/protobufjs/scale.d.cts gen/protobufjs/scale.cjs", { cwd: baseDir });

  // 5. Generate adapters
  // Protobuf-ES
  let esImports = `import { create } from "@bufbuild/protobuf";\n`;
  let esSchemaNames: string[] = [];
  for (let i = 1; i <= N; i++) {
    esImports += `import { ScaleMessage${i}Schema } from "../gen/protobuf-es/scale_pb.js";\n`;
    esSchemaNames.push(`ScaleMessage${i}Schema`);
  }
  const esAdapterContent = `${esImports}\nexport function run() {\n  return [\n    ${esSchemaNames.map(s => `create(${s})`).join(",\n    ")}\n  ];\n}\n`;
  writeFileSync(`${baseDir}/adapters/scale-protobuf-es.ts`, esAdapterContent);

  // ts-proto
  let tsImports = ``;
  let tsNames: string[] = [];
  for (let i = 1; i <= N; i++) {
    tsImports += `import { ScaleMessage${i} } from "../gen/ts-proto/scale.js";\n`;
    tsNames.push(`ScaleMessage${i}`);
  }
  const tsAdapterContent = `${tsImports}\nexport function run() {\n  return [\n    ${tsNames.map(t => `${t}.create()`).join(",\n    ")}\n  ];\n}\n`;
  writeFileSync(`${baseDir}/adapters/scale-ts-proto.ts`, tsAdapterContent);

  // protobufjs
  const pbjsAdapterContent = `import generated from "../gen/protobufjs/scale.cjs";\nexport function run() {\n  return [\n    ${Array.from({ length: N }, (_, i) => `generated.scale.ScaleMessage${i + 1}.create()`).join(",\n    ")}\n  ];\n}\n`;
  writeFileSync(`${baseDir}/adapters/scale-protobufjs.ts`, pbjsAdapterContent);

  // google-protobuf
  const gpAdapterContent = `import scale from "../gen/google-protobuf/scale_pb.cjs";\nexport function run() {\n  return [\n    ${Array.from({ length: N }, (_, i) => `new scale.ScaleMessage${i + 1}()`).join(",\n    ")}\n  ];\n}\n`;
  writeFileSync(`${baseDir}/adapters/scale-google-protobuf.ts`, gpAdapterContent);

  // 6. Bundle with esbuild
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
  const pbSize = await bundle("protobufjs");
  const gpSize = await bundle("google-protobuf");

  results.push({
    messages: N,
    protobufEsMin: esSize.min,
    protobufEsGzip: esSize.gzip,
    tsProtoMin: tsSize.min,
    tsProtoGzip: tsSize.gzip,
    protobufjsMin: pbSize.min,
    protobufjsGzip: pbSize.gzip,
    googleProtobufMin: gpSize.min,
    googleProtobufGzip: gpSize.gzip
  });
}

console.log("\nScale Test Results:\n");
console.log(JSON.stringify(results, null, 2));

// Write back to results/scale.json
writeFileSync(`${baseDir}/results/scale.json`, JSON.stringify(results, null, 2));

// Write back to results/scale.md
const scaleMD = [
  "| Messages | Protobuf-ES Minified | ts-proto Minified | protobuf.js Minified | google-protobuf Minified |",
  "| :--- | ---: | ---: | ---: | ---: |",
  ...results.map(r => `| ${r.messages} | ${(r.protobufEsMin / 1024).toFixed(2)} KiB | ${(r.tsProtoMin / 1024).toFixed(2)} KiB | ${(r.protobufjsMin / 1024).toFixed(2)} KiB | ${(r.googleProtobufMin / 1024).toFixed(2)} KiB |`),
  ""
].join("\n");
writeFileSync(`${baseDir}/results/scale.md`, scaleMD);

console.log("\nScale test results saved to results/scale.json and results/scale.md\n");
