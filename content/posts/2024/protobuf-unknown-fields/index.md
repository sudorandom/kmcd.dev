---
categories: ["article"]
tags: ["protobuf", "grpc", "api", "microservices", "datapipelines", "connectrpc", "go", "typescript", "architecture"]
keywords: ["protobuf unknown fields", "schema evolution", "api gateway", "data preservation", "forward compatibility", "proto3"]
date: "2026-04-16"
description: "How Protobuf unknown fields enable seamless schema evolution and robust middleware."
cover: "cover.svg"
images: ["/posts/protobuf-unknown-fields/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Unknown Fields in Protobuf"
slug: "protobuf-unknown-fields"
type: "posts"
devtoId: 1797652
devtoPublished: false
devtoSkip: false
canonical_url: https://kmcd.dev/posts/protobuf-unknown-fields/
mastodonID: "112277337082054030"
---

{{< disclaimer >}}
This article was originally published in March 2024. It was republished in April 2026 after some significant editing and modernization.
{{< /disclaimer >}}

[Protobuf](https://protobuf.dev/programming-guides/proto3/) includes a feature known as [**unknown fields**](https://protobuf.dev/programming-guides/proto3/#unknowns). They act as a safety net when systems encounter data they weren't explicitly built to handle. Here is a breakdown of what they are and why they matter.

## What are Protobuf Unknown Fields?

Your `.proto` file defines the expected structure, fields, and data types. But what happens when you parse a message and it contains fields that aren't in your current `.proto` definition?

These extra pieces of data are called **unknown fields**.

At a lower level, unknown fields are **field numbers and wire types that exist in the serialized message but are not defined in the current schema**.

This mechanism is what enables **forward compatibility**: an older version of your software can safely read, process, and forward data produced by a newer version of the schema without crashing or losing the new data.

> **Key idea:** Unknown fields enable forward compatibility by default.

---

## Preserving Unknown Data

A key aspect of unknown fields is how they behave during message manipulation.

If you receive a message with unknown fields and forward it to another system, Protobuf defaults to **forwarding the unknown fields alongside the known ones**. This ensures the receiving system gets the complete payload.

If this didn't happen, you could accidentally clear field values set by another part of the system.

This forwarding capability also applies when **persisting messages**, as long as they remain in **binary Protobuf format**. If you store and later reload the binary payload, the unknown fields are preserved.

> **Key idea:** Binary Protobuf preserves unknown fields end-to-end.

> **Historical Note:** Protobuf v3 initially tried to simplify the specification by removing several proto2 features, but real-world usage forced them to walk the biggest ones back. Early versions of proto3 dropped unknown fields entirely, but this was reversed in v3.5. Similarly, proto3 initially removed the `optional` keyword, but brought it back in v3.15 after developers struggled to distinguish between a field being unset and a field just having a zero value, which is [a classic programming mistake](https://en.wikipedia.org/wiki/Null_Island).

---

## JSON Comparison (What Actually Breaks)

Consider a scenario where a new field, `email`, is added to a user object. The backend is updated, but the frontend is not.

The issue in JSON systems is not JSON itself, but rather **typed deserialization**.

```json
{
  "user": {
    "id": "0edc0903-9e31-47be-adad-1dfc434ca2d3",
    "name": "Bob",
    "email": "bob@example.com"
  }
}
```

If the frontend maps this into a typed structure:

```typescript
class User {
  id: string;
  name: string;
}
```

The unknown field (`email`) is dropped during deserialization. When the object is sent back:

```json
{
  "user": {
    "id": "0edc0903-9e31-47be-adad-1dfc434ca2d3",
    "name": "Bob"
  }
}
```

The `email` field is lost.

> **Key idea:** Typed JSON pipelines often drop unknown fields during reserialization.

---

## Protobuf Behavior

With Protobuf, the same scenario behaves differently.

Even if the frontend does not know about the `email` field, it is preserved internally:

```typescript
Symbol(@bufbuild/protobuf/unknown-fields): [
  {0: {no:3, wire_type:2, data: Uint8Array(14)}}
]
```

*(Note: This specific `Symbol` representation is how the `@bufbuild/protobuf` implementation manages it under the hood. Other JS/TS generators might expose this data slightly differently, but the underlying concept remains the same.)*

- `no: 3` → field number (email)
- `wire_type: 2` → length-delimited (used for strings)
- `data` → raw encoded value

When the message is re-encoded:

```text
1:LEN {"0edc0903-9e31-47be-adad-1dfc434ca2d3"}
2:LEN {"Bob"}
3:LEN {"bob@example.com"}
```

The unknown field survives the round trip.

> **Key idea:** Unknown fields are preserved even when not understood.

---

## The Middleware Advantage

Unknown fields shine in internal, middleware-heavy architectures.

Example:

- API Gateway reads `id` for routing
- Logging service reads `trace_id`
- Downstream service understands full schema including new fields

Intermediate services can safely:
1. Unmarshal using an older schema
2. Read known fields
3. Forward the message unchanged

No coordination is required when new fields are added upstream.

> **Key idea:** Internal middleware can stay stable while schemas evolve.

---

## Observability: A Signal for Upgrades

Beyond just forwarding data safely, unknown fields provide a highly valuable observability metric.

When an API gateway or a downstream service detects unknown fields in incoming payloads, it serves as a clear telemetry signal: a client or upstream service is sending extra information because it is using a newer schema.

Instead of crashing or silently dropping the data, the service can log the presence of these unknown fields. You can use this data to trigger alerts, track the rollout progress of new features across your architecture, and pinpoint exactly which legacy services are lagging behind and due for an upgrade.

---

## A Note on JSON Serialization and Object Re-use

There are a couple of important exceptions where unknown fields get lost.

First, unknown field preservation applies **only to binary Protobuf serialization**. If you convert from binary to JSON (e.g., using `protojson` in Go or `toJson` in TypeScript), unknown fields are **dropped** during the encoding process. Conversely, when unmarshaling JSON back into Protobuf, many libraries are strictly configured by default. For instance, Go's `protojson.Unmarshal` will throw a hard error if it encounters unknown fields in the JSON payload unless you explicitly bypass it by passing `DiscardUnknown: true`. JSON simply isn't designed to carry this extra payload without a strict schema map.

Second, preserving these fields during binary serialization requires that you re-use the exact same object for re-serialization. If you read a message, pull out the known fields, and map them into a freshly created object to send downstream, the unknown fields tied to the original object will be left behind.

> **Key idea:** Binary preserves and JSON drops. Always re-use the original object if you want to keep unknown fields intact.

---

## Databases and Security

The theoretical elegance of unknown fields often collides with the messy reality of databases and security perimeters. In practice, relying on unknown fields breaks down entirely in a few critical scenarios.

First, consider database persistence. If clients are trying to store extra data, and a backend service parses a Protobuf message to map it to standard relational database columns, those unknown fields are absolutely gone. There is no magic column for data your database schema does not know about.

The only way to achieve true end-to-end preservation is to store the entire serialized Protobuf message directly in the database as a BLOB. Some teams do this, but blindly storing data you haven't validated and don't even recognize is highly dangerous.

Allowing unknown fields to propagate unchecked from external sources is a significant security risk. While they are a powerful tool inside clearly defined, trusted internal pipelines, accepting them from the open web opens your system up to data smuggling. It allows malicious actors to sneak unvalidated payloads into unknown fields to bypass validation layers that only inspect known schema structures. If your systems blindly unmarshal, store, and forward this data, older services act as unwitting mules for malicious input.

Because of these exact risks, the standard security posture is to aggressively filter at the edge. API Gateways and ingress proxies should explicitly discard unknown fields before the data ever reaches internal microservices.

---

## Conclusion

Unknown fields provide a powerful mechanism for **forward compatibility** in distributed systems. They allow internal systems to evolve independently, act as a clear signal for required upgrades, reduce coordination overhead, and simplify middleware design.

However, they are not a substitute for validation, schema discipline, or proper security boundaries. Use them intentionally in trusted internal pipelines, but never trust them at the edge.
