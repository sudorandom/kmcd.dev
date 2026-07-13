import { BenchmarkMessageSchema } from "../gen/protobuf-es/benchmark_pb.js";

console.log("BenchmarkMessageSchema keys:", Object.keys(BenchmarkMessageSchema));
console.log("Fields:", (BenchmarkMessageSchema as any).fields);
if ((BenchmarkMessageSchema as any).fields) {
  for (const f of (BenchmarkMessageSchema as any).fields) {
    console.log("Field:", f.no, f.name, f.kind, f.type, f.repeated);
  }
}
