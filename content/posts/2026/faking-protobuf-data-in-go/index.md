---
categories: ["article"]
tags: ["protobuf", "grpc", "testing"]
date: "2026-03-12"
description: "Stop hand-writing test fixtures"
cover: "cover.svg"
images: ["/posts/faking-protobuf-data-in-go/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Faking protobuf data in Go"
slug: "faking-protobuf-data-in-go"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/faking-protobuf-data-in-go/
---

If you build Go services that use protobuf over gRPC, you often need realistic-looking data. Sometimes you need a lot of it and your creative juices and manual data entry only goes so far. Tests, local demos, and stubs all benefit from messages that look plausible without carrying meaning.

FauxRPC addresses this. It’s a small Go library that fills protobuf messages with plausible data. The focus is not perfect realism; it’s speed, convenience, and reducing boilerplate.

This article covers two main use cases: populating generated protobuf types and working with dynamic descriptors. Along the way, you’ll see how to generate data and control its behavior for repeatable results.

## Core entry points

FauxRPC provides two primary functions:

* [`fauxrpc.SetDataOnMessage`](https://pkg.go.dev/github.com/sudorandom/fauxrpc#SetDataOnMessage): Populates a concrete Go struct generated from `.proto` files.
* [`fauxrpc.NewMessage`](https://pkg.go.dev/github.com/sudorandom/fauxrpc#NewMessage): Creates a message from a `protoreflect.MessageDescriptor`, useful when types aren’t known at compile time.

Both use the same underlying logic; the difference is whether you start with a Go struct or a descriptor.

## Filling a generated message

For a simple case, consider the Eliza demo service and its `SayResponse` message.

{{< details-md open="true" summary="Eliza Service SayResponse Example" github_file="go/example1_say_response.go" >}}
{{% render-code file="go/example1_say_response.go" language="go" %}}
{{< /details-md >}}

`SetDataOnMessage` mutates the message in place. Every field receives a value compatible with its type.

Marshaling to JSON produces something like:

```json
{
  "sentence": "Jean shorts."
}
```

Values change each run. That randomness exposes assumptions in tests and ensures your code doesn’t rely on fixed values.

## Populating a nested message

Most protobufs are more complex than a single field. For example, an `ownerv1.Owner` message with nested messages, timestamps, enums, and strings:

{{< details-md open="true" summary="Owner Service Owner Example" github_file="go/example2_owner.go" >}}
{{% render-code file="go/example2_owner.go" language="go" %}}
{{< /details-md >}}

`SetDataOnMessage` recursively populates nested messages, assigns valid enum values, and generates RFC 3339 timestamps. Marshaled with `protojson`:

```json
{
  "organization": {
    "id": "a4cf6166453b49d9811cfdc169c36354",
    "createTime": "1912-06-01T16:45:13.510830647Z",
    "updateTime": "2009-04-03T13:11:31.216229245Z",
    "name": "xm0r3",
    "description": "Godard selvage.",
    "url": "https://www.dynamicreintermediate.name/robust/seize/metrics/b2c",
    "verificationStatus": "ORGANIZATION_VERIFICATION_STATUS_OFFICIAL"
  }
}
```

This works well for local servers, demos, and tests that focus on shape rather than meaning.

## Using dynamic messages

For proxies, gateways, or tools without generated Go types, `dynamicpb` allows message creation from descriptors. FauxRPC supports this:

{{< details-md open="true" summary="Dynamic Protobuf Example" github_file="go/example3_dynamic.go" >}}
{{% render-code file="go/example3_dynamic.go" language="go" %}}
{{< /details-md >}}

FauxRPC fills dynamic messages with the same rules as concrete types, making them usable like any other protobuf message.

## Respecting protovalidate constraints

FauxRPC reads `protovalidate` annotations to generate data that respects field constraints. Length limits, numeric ranges, regex patterns, and formats are applied when generating values.

For example, a `User` message with a username, email, and age:

{{< details-md open="true" summary="Protovalidate-aware Data Generation" github_file="go/example5_protovalidate.go" >}}
{{% render-code file="go/example5_protovalidate.go" language="go" %}}
{{< /details-md >}}

Sample output:

```json
{
  "username": "Ezequiel",
  "email": "kiarasantos@grant.ne",
  "age": 46
}
```

FauxRPC ensures string lengths, numeric ranges, and formats match the annotations. The result is still fake data but plausible enough to exercise validators and gateways realistically.

## Controlling randomness with GenOptions

By default, FauxRPC uses a global random generator. `GenOptions` allows supplying a seeded `gofakeit.Faker` for repeatable output:

{{< details-md open="true" summary="Customizing Data Generation with GenOptions" github_file="go/example4_genoptions.go" >}}
{{% render-code file="go/example4_genoptions.go" language="go" %}}
{{< /details-md >}}

Repeatable generation helps make tests predictable without hard-coded fixtures.

## FauxRPC in handlers

At the edge of a service, handlers can generate fake responses quickly.

ConnectRPC example:

{{< details-md open="true" summary="ConnectRPC Handler" github_file="go/snippet1_connect_handler.go" >}}
{{% render-code file="go/snippet1_connect_handler.go" language="go" %}}
{{< /details-md >}}

Request example:

```shell
$ buf curl --schema=buf.build/connectrpc/eliza \
         -d '{"sentence": "Hello world!"}' \
         http://127.0.0.1:6660/connectrpc.eliza.v1.ElizaService/Say
{
  "sentence": "Microdosing."
}
```

gRPC-go example:

{{< details-md open="true" summary="gRPC-go Handler" github_file="go/snippet2_grpc_handler.go" >}}
{{% render-code file="go/snippet2_grpc_handler.go" language="go" %}}
{{< /details-md >}}

The behavior is identical and works out-of-the-box.

## FauxRPC CLI

The CLI can generate data without compiling Go code:

```shell
$ fauxrpc generate --schema=. --target=example.v1.User
{"username":"Birdie", "email":"pariskozey@hamill.in", "age":54}
```

The CLI is incredibly flexible with the --schema flag. You can point it at:

- A local .proto file or a directory of them.
- A compiled protobuf descriptor set (binpb).
- A remote Buf Schema Registry repository (e.g., buf.build/acme/auth).

## Closing thoughts

FauxRPC is not a data modeling tool; it generates values consistent with protobuf types and constraints. This makes it useful for quickly standing up services, testing clients, and exploring API shapes.

It works with both ConnectRPC and grpc-go, and provides hooks for custom data generation. For full details, see the [pkg.go.dev reference](https://pkg.go.dev/github.com/sudorandom/fauxrpc).
