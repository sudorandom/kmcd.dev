import protobuf from "protobufjs";
import schemaJson from "../gen/protobufjs/benchmark.json";
import { fixture } from "../shared/fixture.js";

const root = protobuf.Root.fromJSON(schemaJson);
const BenchmarkMessage = root.lookupType("benchmark.BenchmarkMessage");

const message = BenchmarkMessage.create({
  ...fixture,
  tags: [...fixture.tags],
  scores: { ...fixture.scores },
  items: fixture.items.map((item) => ({ ...item, samples: [...item.samples] })),
  note: fixture.note,
});
const encoded = BenchmarkMessage.encode(message).finish();

export const library = "protobuf.js (Reflection)";
export const wireSize = encoded.byteLength;

export function encode(iterations: number): number {
  let checksum = 0;
  for (let index = 0; index < iterations; index++) {
    const bytes = BenchmarkMessage.encode(message).finish();
    checksum += bytes.byteLength + bytes[0] + bytes[bytes.byteLength >> 1] + bytes[bytes.byteLength - 1];
  }
  return checksum;
}

export function decode(iterations: number): number {
  let checksum = 0;
  for (let index = 0; index < iterations; index++) {
    const decoded = BenchmarkMessage.decode(encoded);
    checksum += (decoded as any).id + (decoded as any).name.length + (decoded as any).tags.length + (decoded as any).items.length + (decoded as any).note!.length;
  }
  return checksum;
}

export function verify(): boolean {
  const decoded = BenchmarkMessage.decode(encoded);
  return (decoded as any).id === fixture.id && (decoded as any).items.length === fixture.items.length && (decoded as any).payload === "note";
}
