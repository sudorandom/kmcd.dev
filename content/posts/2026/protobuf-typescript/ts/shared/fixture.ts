export const fixture = {
  id: 42,
  name: "Ada Lovelace",
  email: "ada@example.com",
  tags: ["typescript", "protobuf", "benchmark", "browser"],
  scores: { correctness: 100, ergonomics: 87, performance: 93 },
  items: [
    { sku: "compiler", quantity: 2, price: 129.95, samples: [1, 2, 3, 5, 8, 13] },
    { sku: "runtime", quantity: 4, price: 49.5, samples: [21, 34, 55, 89] },
    { sku: "types", quantity: 1, price: 79.0, samples: [144, 233, 377] },
  ],
  note: "A representative message with scalar, repeated, map, nested, and oneof fields.",
  active: true,
} as const;
