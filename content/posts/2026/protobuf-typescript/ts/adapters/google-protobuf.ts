// The official generator emits JavaScript rather than TypeScript declarations.
// @ts-expect-error generated CommonJS module has no declaration file
import generated from "../gen/google-protobuf/benchmark_pb.cjs";
import { fixture } from "../shared/fixture.js";

const message = new generated.BenchmarkMessage();
message.setId(fixture.id);
message.setName(fixture.name);
message.setEmail(fixture.email);
message.setTagsList([...fixture.tags]);
for (const [key, value] of Object.entries(fixture.scores)) {
  message.getScoresMap().set(key, value);
}
message.setItemsList(fixture.items.map((value) => {
  const item = new generated.Item();
  item.setSku(value.sku);
  item.setQuantity(value.quantity);
  item.setPrice(value.price);
  item.setSamplesList([...value.samples]);
  return item;
}));
message.setNote(fixture.note);
message.setActive(fixture.active);
const encoded = message.serializeBinary();

export const library = "google-protobuf";
export const wireSize = encoded.byteLength;

export function encode(iterations: number): number {
  let checksum = 0;
  for (let index = 0; index < iterations; index++) {
    const bytes = message.serializeBinary();
    checksum += bytes.byteLength + bytes[0] + bytes[bytes.byteLength >> 1] + bytes[bytes.byteLength - 1];
  }
  return checksum;
}

export function decode(iterations: number): number {
  let checksum = 0;
  for (let index = 0; index < iterations; index++) {
    const decoded = generated.BenchmarkMessage.deserializeBinary(encoded);
    checksum += decoded.getId() + decoded.getName().length + decoded.getTagsList().length + decoded.getItemsList().length + decoded.getNote().length;
  }
  return checksum;
}

export function verify(): boolean {
  const decoded = generated.BenchmarkMessage.deserializeBinary(encoded);
  return decoded.getId() === fixture.id && decoded.getItemsList().length === fixture.items.length && decoded.hasNote();
}
