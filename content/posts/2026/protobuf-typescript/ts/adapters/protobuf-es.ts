import { create, fromBinary, toBinary } from "@bufbuild/protobuf";
import { BenchmarkMessageSchema } from "../gen/protobuf-es/benchmark_pb.js";
import { fixture } from "../shared/fixture.js";

const message = create(BenchmarkMessageSchema, {
  ...fixture,
  tags: [...fixture.tags],
  scores: { ...fixture.scores },
  items: fixture.items.map((item) => ({ ...item, samples: [...item.samples] })),
  payload: { case: "note", value: fixture.note },
});
const encoded = toBinary(BenchmarkMessageSchema, message);

export const library = "Protobuf-ES";
export const wireSize = encoded.byteLength;

export function encode(iterations: number): number {
  let checksum = 0;
  for (let index = 0; index < iterations; index++) {
    const bytes = toBinary(BenchmarkMessageSchema, message);
    checksum += bytes.byteLength + bytes[0] + bytes[bytes.byteLength >> 1] + bytes[bytes.byteLength - 1];
  }
  return checksum;
}

export function decode(iterations: number): number {
  let checksum = 0;
  for (let index = 0; index < iterations; index++) {
    const decoded = fromBinary(BenchmarkMessageSchema, encoded);
    checksum += decoded.id + decoded.name.length + decoded.tags.length + decoded.items.length + decoded.payload.value!.length;
  }
  return checksum;
}

export function verify(): boolean {
  const decoded = fromBinary(BenchmarkMessageSchema, encoded);
  return decoded.id === fixture.id && decoded.items.length === fixture.items.length && decoded.payload.case === "note";
}
