---
categories: ["article"]
tags: ["protobuf", "grpc", "api"]
date: "2025-08-05T10:00:00Z"
description: "How to avoid breaking gRPC clients."
cover: "cover.webp"
images: ["/posts/breaking-grpc/cover.webp"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Breaking gRPC"
slug: "breaking-grpc"
type: "posts"
devtoSkip: true
canonical_url: "https://kmcd.dev/posts/breaking-grpc/"
---

When we use gRPC, we often praise its efficiency and strong contracts defined by Protocol Buffers (`.proto` files). We know that gRPC uses protobuf's binary format for fast, compact, and forward/backward-compatible communication. But what happens when you expose your gRPC service to clients who speak JSON, like a web frontend?

The encoding you use (binary protobuf or transcoded JSON) dramatically changes the rules of what constitutes a "safe" or "breaking" change to your API. A change that is perfectly harmless for a protobuf client can completely break a JSON client. Let's dig into this more.

## How Encodings Work

First, a quick refresher on how each format represents data. Consider this simple protobuf message:

```proto
syntax = "proto3";

package my_service.v1;

message User {
  // A unique identifier for the user.
  int64 user_id = 1;
  // The user's full name.
  string name = 2;
}
```

### Protobuf: It's All About the Numbers

On the wire, the binary protobuf encoding doesn't care about the field names (`user_id`, `name`). It only cares about the **field numbers** (`1`, `2`) and their wire types. A simplified view of the encoded data is a series of key-value pairs where the key is the field number. I dig into this further in my [gRPC from Scratch](/posts/grpc-from-scratch-part-3/) series, where I discuss the binary protobuf encoding.

Because of this, you can rename a field in your `.proto` file, and as long as the field number and type remain the same, it's a **non-breaking change** for protobuf clients.

### JSON: It's All About the Names

When a gRPC gateway or library transcodes this message to JSON, it produces a standard JSON object. JSON is also a perfectly valid encoding to use with gRPC. By default, it uses the protobuf field names (converted to `lowerCamelCase`) as the JSON keys:

```json
{
  "userId": 12345,
  "name": "Alex"
}
```

Since JSON clients are coupled to these names, changing them will inevitably break the integration. **JSON clients are coupled to field names, not field numbers.** This fundamental difference is the source of many potential compatibility issues.

-----

## Analyzing API Changes: Breaking vs. Non-Breaking

Let's look at common changes you might make to a `.proto` file and see their impact on each encoding.

| Change | Protobuf Impact | JSON Impact | Explanation |
| :--- | :--- | :--- | :--- |
| **Renaming a field** (`name` to `full_name`) | ✅ **Non-breaking** | 💥 **Breaking** | Protobuf clients only see the field number (`2`), which hasn't changed. JSON clients expect the key `"name"` but will now see `"fullName"`. |
| **Changing a field number** (`= 2` to `= 3`) | 💥 **Breaking** | ✅ **Non-breaking** | This is a cardinal sin in the protobuf world. A client expecting field `2` will no longer find it. JSON clients, however, still see the key `"name"` and are unaffected. |
| **Adding a new field** (`email = 3`) | ✅ **Non-breaking** | ✅ **Non-breaking** | Well-behaved clients in both formats are designed to ignore unknown fields, making this a safe operation. |
| **Removing or deprecating a field** | ✅ **Non-breaking** | ✅ **Non-breaking** | Similar to adding a field, clients should handle missing fields gracefully. It's best practice to `deprecate` a field before removing it. |
| **Changing a compatible type** (`int32` to `int64`) | ✅ **Non-breaking** | ✅ **Non-breaking** | These types have compatible wire formats in protobuf. For JSON, both are simply numbers, so there's no issue. |
| **Changing an incompatible type** (`int64` to `string`) | 💥 **Breaking** | 💥 **Breaking** | The wire format for a number and a string are different, breaking protobuf clients. The data type in JSON also changes (e.g., `123` vs. `"123"`), which will break any client expecting a number. |

-----

## The Solution: Decouple Names with `json_name`

So, how do you refactor your `.proto` field names without breaking your JSON clients? The protobuf specification provides a simple and elegant solution: the **`json_name`** field option.

This option lets you explicitly set the JSON key for a field, decoupling it from the `.proto` field name.

Let's revise our `User` message. Suppose we want to rename `name` to `full_name` for clarity in our Go or Python code, but we can't break existing JSON clients that rely on the `"name"` key.

```proto
syntax = "proto3";

package my_service.v1;

message User {
  int64 user_id = 1 [json_name = "userId"];

  // The field is now 'full_name' in code, but will still be 'name' in JSON.
  string full_name = 2 [json_name = "name"];
}
```

With `json_name = "name"`, we've instructed the transcoder to do the following:

1.  **For Protobuf:** Continue using field number `2`. The field name `full_name` is used by the code generator.
2.  **For JSON:** Always use the key `"name"` during serialization, regardless of what the `.proto` field is called.

Now, you are free to change the `full_name` field to something else (e.g., `user_display_name`) in the future, and your JSON contract remains stable.

## Automating Your Safety Net with `buf breaking`

Remembering all these nuanced rules across different encodings is difficult and error-prone. This is where automated tooling becomes essential. The popular [Buf toolchain](https://buf.build/) includes a powerful command, **`buf breaking`**, designed specifically for this problem.

The `buf breaking` command compares your current `.proto` files against a previous state (like your main git branch) and reports any changes that would break your API consumers. Crucially, it understands that "breaking" means different things to different clients. You can configure it to check against multiple compatibility strategies.

In your `buf.yaml` configuration file, you can specify which rule sets to check against:

  * `FILE`: Checks for backward-incompatible changes at the `.proto` file level, like deleting a field or changing a field number. This protects your **protobuf-based clients**.
  * `WIRE_JSON`: Checks for backward-incompatible changes for the JSON wire format. This catches things like renaming a field without using `json_name`. This protects your **JSON-based clients**.
  * `PACKAGE`: Checks for source-code-level breaking changes in the generated stubs for languages like Go and Java. This protects the **developers using your generated code**.

A typical configuration for a service with both gRPC and JSON clients might look like this:

```yaml
# buf.yaml
version: v2
breaking:
  use:
    - FILE
    - WIRE_JSON
```

By integrating `buf breaking` into your CI/CD pipeline, you can automatically prevent developers from merging changes that would break any of your consumers, whether they speak protobuf or JSON.

-----

## Conclusion

Evolving an API for both protobuf and JSON clients is a recipe for a very specific kind of headache, the kind that pages you at 3 AM. You've got protobuf, which only cares about numbers, and JSON, which only cares about names. A "safe" refactor for one is a production-breaking slap in the face for the other. This is where a schema-first approach, backed by powerful schema-aware tooling, isn't just a good idea; it's the only thing keeping you from questioning all your life choices.

Protobuf's semantics, like the `json_name` option, give you a powerful escape hatch. It makes certain refactors, like renaming a field for internal clarity, trivial *if* you have the right tooling in place. You can change your code without your JSON clients ever knowing you touched a thing. This decoupling is a superpower, but only if you use it correctly.

And that's the catch: don't rely on developers' goldfish sized memory or manual code reviews to enforce these complex, conflicting rules. That's how you break production at 3 AM. Instead, let the robots do the heavy lifting. Integrating a tool like `buf breaking` into your CI pipeline is like having an unblinking, unforgiving guardian for your API. It understands the different breaking change rules for both protobuf and JSON and will stop a bad change before it ever gets merged. This is the real strength of a schema-first workflow: it makes complex refactors not just possible, but safe. You can merge with confidence and keep all your clients (binary or JSON) happy.
