import generated from "../gen/protobufjs/benchmark.cjs";
import { fixture } from "../shared/fixture.js";

const message = generated.benchmark.BenchmarkMessage.create({
  ...fixture,
  tags: [...fixture.tags],
  scores: { ...fixture.scores },
  items: fixture.items.map((item) => ({ ...item, samples: [...item.samples] })),
  note: fixture.note,
});
const encoded = generated.benchmark.BenchmarkMessage.encode(message).finish();

export const library = "protobuf.js";
export const wireSize = encoded.byteLength;

export function encode(iterations: number): number {
  let checksum = 0;
  for (let index = 0; index < iterations; index++) {
    const bytes = generated.benchmark.BenchmarkMessage.encode(message).finish();
    checksum += bytes.byteLength + bytes[0] + bytes[bytes.byteLength >> 1] + bytes[bytes.byteLength - 1];
  }
  return checksum;
}

export function decode(iterations: number): number {
  let checksum = 0;
  for (let index = 0; index < iterations; index++) {
    const decoded = generated.benchmark.BenchmarkMessage.decode(encoded);
    checksum += decoded.id + decoded.name.length + decoded.tags.length + decoded.items.length + decoded.note!.length;
  }
  return checksum;
}

export function verify(): boolean {
  const decoded = generated.benchmark.BenchmarkMessage.decode(encoded);
  return decoded.id === fixture.id && decoded.items.length === fixture.items.length && decoded.payload === "note";
}
