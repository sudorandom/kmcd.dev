---
categories: ["article"]
tags: ["protobuf", "grpc", "protovalidate", "buf", "web"]
date: "2025-02-11T10:00:00Z"
description: "Effortless input validation for Protobuf! Protovalidate lets you define rules directly in your .proto files."
cover: "cover.jpg"
images: ["/posts/protovalidate/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Protovalidate: Can Input Validation Be This Easy?"
slug: "protovalidate"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/protovalidate/
---

> Since writing this article, ECMAScript support for protovalidate was created and has had its first release with a new library called [protovalidate-es](https://github.com/bufbuild/protovalidate-es)! Finally, an easy way to share validation rules across backend, frontend and to all kinds of other clients and message-based systems is here.

User input can be absolute garbage. I can't be trusted to type my name correctly half of the time, so it's obvious that we need to validate input to catch the most obvious mistakes. Zip codes don't include letters, number of pets can't be negative and Hitachi Rail Italy Driverless Metro is not a model of car.

Input validation is a consistent problem in the web services industry. Usually there are constraints that are defined at multiple levels:
- Databases often have constraints. For example, the size of a single field is usually constrained in some way and there are usually types that offer some guarantees.
- Statically typed programming languages have some amount of safety for input. You know an integer type isn't going to magically become a string based on user input.
- There are many libraries that assist with this, but backend programmers will have to add constraints on input from users. Usernames can't be 1000 characters long, someone can't be aged negative 10, etc. **This validation is often the most important for data integrity** because it's the last part of the architecture where developers have 100% control.
- On the frontend, many of the constraints and rules from the backend are often replicated. This is usually done for UX (User eXperience) reasons. You don't want to wait until you submit a large form with many inputs before realizing you messed up. It's often way better to highlight issues before users try to submit. Often, these constraints are communicated by hand-written documentation that is often not kept up-to-date or not at all, forcing frontend developers to duplicate backend validation logic.

{{< diagram >}}
{{< markdown >}}
```d2
direction: right
style: {
    fill: transparent
}

input: User Input
input.shape: person
frontend-validation: {
    label: validation
    frontend: Frontend
}
backend-validation: {
    label: validation
    backend: Backend
}
attacker: Attacker
attacker.shape: person
database: Database

input -> frontend-validation
frontend-validation.frontend -> backend-validation
attacker -> backend-validation
backend-validation.backend -> database
```
{{< /markdown >}}
{{< /diagram >}}

This is where [protovalidate](https://github.com/bufbuild/protovalidate) comes in. Protovalidate allows you to specify constraints beside your protobuf-defined API and type definitions. I [talked before about API contracts](https://kmcd.dev/posts/api-contracts/), but protovalidate goes a step above what protobuf offers by default. In addition to the cross-language type safety that protobuf offers, protovalidate allows you to define additional constraints for each field. So now the answer to many input validation questions can be answered by directly looking at this file or by using a protovalidate library written for several languages. Now both the frontend and backend logic can be powered by the same declarative schema.

{{< diagram >}}
{{< markdown >}}
```d2
direction: right
style: {
    fill: transparent
}

input: User Input
input.shape: person
frontend-validation: {
    label: validation
    frontend: Frontend
}
backend-validation: {
    label: validation
    backend: Backend
}
attacker: Attacker
attacker.shape: person
database: Database
protovalidate {
    near: top-center
    style {
        fill: "#FFA500"
        font-size: 28
        font-color: "#000000"
    }
}
protovalidate.style.animated: true

input -> frontend-validation
frontend-validation.frontend -> backend-validation
attacker -> backend-validation
backend-validation.backend -> database

protovalidate -> frontend-validation {
    style.animated: true
}
protovalidate -> backend-validation {
    style.animated: true
}
```
{{< /markdown >}}
{{< /diagram >}}

## Why protovalidate?
The traditional approach to input validation is often a fragmented and inconsistent mess. Frontend and backend teams may implement their own validation rules, leading to duplicated code, discrepancies, and the constant struggle to keep everything in sync. Imagine the headache of trying to track down and fix a validation bug that exists in three different places! Protovalidate offers a much better way. It provides a centralized, declarative approach, allowing you to define your validation rules directly in your `.proto` files – the same place you define your data structures. For example, specifying that a username must be between 3 and 50 characters is as simple as adding a few annotations to your Protobuf definition: `string name = 1 [(buf.validate.field).string.min_len = 3, (buf.validate.field).string.max_len = 50];`. This not only significantly reduces boilerplate code but also ensures consistency across your entire application. Because the rules are defined alongside your data structures, they become a living part of the API contract, improving communication between teams and reducing the risk of misinterpretations.

Protovalidate supports multiple languages, including [Go](https://github.com/bufbuild/protovalidate-go), [Java](https://github.com/bufbuild/protovalidate-java), [Python](https://github.com/bufbuild/protovalidate-python/), and [C++](https://github.com/bufbuild/protovalidate-cc), with community support for others like [.NET](https://github.com/telus-oss/protovalidate-net). Built on top of the Common Expression Language (CEL), Protovalidate offers a powerful and extensible way to define even the most complex validation rules, ensuring your application's data remains clean and reliable.

## Showing it off
I can tell you how cool protovalidate is, but I figured it would be better to show you. Let's look at some of the constraints.

### Built-in constraints
Protovalidate comes with a set of built-in constraints for common data types. Let's explore some of these, starting with strings.

#### Strings
Let's start simple. Here's some constraints that you might see for a username. It has a minimum length of 3 characters and a max length of 50. And this is a required field:

```protobuf
string name = 1 [
  (buf.validate.field).string = {
    min_len: 3
    max_len: 50
  },
  (buf.validate.field).required = true
];
```

Email addresses can also be validated:
```protobuf
string email = 3 [(buf.validate.field).string.email = true];
```

You can use regex to have precisely describe what a valid value looks like:
```protobuf
// City of residence, must only contain letters, spaces, and hyphens.
string city = 5 [(buf.validate.field).string.pattern = "^[a-zA-Z]+(?:[\\s-][a-zA-Z]+)*$"];
```

#### Integers
```protobuf
int32 age = 2 [(buf.validate.field).int32.gte = 0, (buf.validate.field).int32.lte = 150];
```

#### Enums
One potential issue with Protobuf enums is that they don't inherently enforce validation against defined values. For example for an enum defined like this:
```protobuf
enum Status { UNKNOWN = 0; ACTIVE = 1; }
```
Without explicit validation, a server could receive an undefined enum value (e.g., `420`) and potentially misinterpret it. To fix this glaring issue, protovalidate has a constraint to limit enums to known and defined values only. Let's look at a field that has this constraint:
```protobuf
Status status = 4 [(buf.validate.field).enum.defined_only = true];
```
With a single annotation you just removed a whole class of issues and simplified input validation.

#### Repeated Fields
There are also constraints for repeated fields. You can constrain the number of items in a list. You can force the items to be unique. You can even add constraints for each item in the list as well.

```protobuf
repeated string tags = 7 [
  (buf.validate.field).repeated.min_items = 1,
  (buf.validate.field).repeated.max_items = 2,
  (buf.validate.field).repeated.unique = true,
  (buf.validate.field).repeated.items.string.min_len = 4
];
```

### CEL-powered custom constraints
One thing that sets protovalidate apart from other validation libraries is the extendability. Protovalidate is built on top of [CEL](https://cel.dev/), which is an embeddable expression engine. What this means for protovalidate is that you can write your own constraints. All of the examples above constrain you on a single field, but it can be incredibly powerful to use multiple fields. This can be useful for many situations but in particular it is useful when you have a defined start and end range where the start strictly needs to come before the end. So let's take a look at what that looks like:

```protobuf
message Event {
  int64 start_time = 1;
  int64 end_time = 2;
  option (buf.validate.message).cel = {
    id: "event.start_time_before_end_time",
    message: "Start time must be before end time",
    expression: "this.start_time < this.end_time",
  };
}
```
This example demonstrates a constraint involving two fields, `start_time` and `end_time`. A reasonable constraint is that start time must go before end time, so that's what this custom constraint is doing. The expression field contains the CEL expression that enforces this rule. CEL's simplicity makes it easy to define custom constraints. `message` allows you to customize the message.

And now are you ready for the big reveal? Every single standard constraint in protovalidate works the same way as this. It's all powered by CEL expressions. This not only provides you with a good reference to make your own custom constraints but it also makes it much easier to add language support for protovalidate. All you need is a good protobuf and CEL library, and you have most of the work already done for you.

### Complete example
[See the full prototype from this post here.](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2025/protovalidate/example.proto)

See a full example protobuf file here. Reference protovalidate's [`validate.proto` file](https://buf.build/bufbuild/protovalidate/docs/main:buf.validate) to see every option available.

Now, let's see Protovalidate in action. We'll use `buf curl`, a command-line tool for interacting with gRPC and Connect services, to send a request to our example service and observe how Protovalidate enforces the defined validation rules.

```bash
buf curl \
    --protocol=grpcweb \
    --http2-prior-knowledge \
    -d '{"profile": {"user": {}}}' \
    http://localhost:6660/example.ExampleService/CreateProfile
{
   "code": "invalid_argument",
   "message": "validation error:\n - profile.user.name: value is required [required]\n - profile.user.email: value is empty, which is not a valid email address [string.email_empty]\n - profile.user.city: value does not match regex pattern `^[a-zA-Z]+(?:[\\s-][a-zA-Z]+)*$` [string.pattern]\n - profile.user.account_id: value must be greater than 0 [int64.gt]\n - profile.user.tags: value must contain at least 1 item(s) [repeated.min_items]\n - profile.user.created_at: value is required [required]",
   "details": [
      {
         "type": "buf.validate.Violations",
         "value": "CjAKEXByb2ZpbGUudXNlci5uYW1lEghyZXF1aXJlZBoRdmFsdWUgaXMgcmVxdWlyZWQKXAoScHJvZmlsZS51c2VyLmVtYWlsEhJzdHJpbmcuZW1haWxfZW1wdHkaMnZhbHVlIGlzIGVtcHR5LCB3aGljaCBpcyBub3QgYSB2YWxpZCBlbWFpbCBhZGRyZXNzCmgKEXByb2ZpbGUudXNlci5jaXR5Eg5zdHJpbmcucGF0dGVybhpDdmFsdWUgZG9lcyBub3QgbWF0Y2ggcmVnZXggcGF0dGVybiBgXlthLXpBLVpdKyg/Oltccy1dW2EtekEtWl0rKSokYApBChdwcm9maWxlLnVzZXIuYWNjb3VudF9pZBIIaW50NjQuZ3QaHHZhbHVlIG11c3QgYmUgZ3JlYXRlciB0aGFuIDAKTgoRcHJvZmlsZS51c2VyLnRhZ3MSEnJlcGVhdGVkLm1pbl9pdGVtcxoldmFsdWUgbXVzdCBjb250YWluIGF0IGxlYXN0IDEgaXRlbShzKQo2Chdwcm9maWxlLnVzZXIuY3JlYXRlZF9hdBIIcmVxdWlyZWQaEXZhbHVlIGlzIHJlcXVpcmVk",
         "debug": {
            "violations": [
               {
                  "fieldPath": "profile.user.name",
                  "constraintId": "required",
                  "message": "value is required"
               },
               {
                  "fieldPath": "profile.user.email",
                  "constraintId": "string.email_empty",
                  "message": "value is empty, which is not a valid email address"
               },
               {
                  "fieldPath": "profile.user.city",
                  "constraintId": "string.pattern",
                  "message": "value does not match regex pattern `^[a-zA-Z]+(?:[\\s-][a-zA-Z]+)*$`"
               },
               {
                  "fieldPath": "profile.user.account_id",
                  "constraintId": "int64.gt",
                  "message": "value must be greater than 0"
               },
               {
                  "fieldPath": "profile.user.tags",
                  "constraintId": "repeated.min_items",
                  "message": "value must contain at least 1 item(s)"
               },
               {
                  "fieldPath": "profile.user.created_at",
                  "constraintId": "required",
                  "message": "value is required"
               }
            ]
         }
      }
   ]
}
```
Here's the response from the server. Notice the code: "invalid_argument" and the message field, which clearly explains the validation errors. Protovalidate caught six distinct issues: the `name` field was missing, the `email` was empty (and therefore invalid), the `city` didn't match the required regex, the `account_id` was zero (not greater than zero), the `tags` list was empty (requiring at least one item), and the `created_at` field was missing.

Now, let's correct the errors by providing valid data. We'll include a name, a valid email, and so on. When we fix all of these issues, suddenly it works!
```bash
$ buf curl \
    --protocol=grpcweb \
    --http2-prior-knowledge \
    -d '{
        "profile": {
            "address": {
                "city": "New York",
                "postal_code": "10001",
                "street": "25th Street West"
            },
            "user": {
                "name": "Bob",
                "email": "bob@example.com",
                "account_id": 10,
                "tags": ["new_user"],
                "created_at": "2023-01-01T00:00:00Z"}}}' \
    http://localhost:6660/example.ExampleService/CreateProfile
{}
```

As you can see, this validation ensures the data conforms to our defined rules.

## My side projects that leverage protovalidate
You've probably realized by now that I'm a big fan of protovalidate. So it's no surprise for you to know that I've used it in both of my larger personal projects recently:

### protoc-gen-connect-openapi
[`protoc-gen-connect-openapi`](https://github.com/sudorandom/protoc-gen-connect-openapi) generates OpenAPI specifications for ConnectRPC servers. This uses protovalidate rules to further annotate the OpenAPI spec.

### FauxRPC
[`FauxRPC`](https://fauxrpc.com/) is my project that is a self-mocking protobuf driven server that supports gRPC/gRPC-Web/ConnectRPC/REST. Not only will it enforce protovalidate annotations on requests made to it but it also uses these annotations to generate more realistic mock data automatically. I feel like this really shows off the power of contract-driven APIs. I actually generated the examples above using FauxRPC!
```bash
$ buf build ssh://git@github.com/sudorandom/kmcd.dev.git#branch=main,subdir=content/posts/2025/protovalidate -o protovalidate-example.binpb
$ fauxrpc run --schema protovalidate-example.binpb
```

## Run in the web
Currently, protovalidate doesn't run on a web frontend, at least [not yet](https://github.com/bufbuild/protovalidate/issues/67). I feel like this is the last piece that can finally unite input validation in the frontend and backend. I mentioned at the start of this article about how the frontend developers have to duplicate many of the rules that already exist in the backend. However, once a typescript version of protovalidate exists suddenly all of this work is completely handled simply by adding some protobuf annotations.

The potential of this library is incredible. I feel like we can have a world where types are defined in one place and used in any number of languages.

## Rough Edges
Protovalidate does have a few rough edges. Overusing validation rules can sometimes make it hard to re-use types for multiple purposes. If you have a type that you use for a user and define a bunch of rules about required fields for that user type, how would you then update only a few of the fields for a user in a different call? I think there are ways around it, but `required` fields may in fact be the issue yet again. There's a past debate within Google about required fields and the result was the removal of the feature in core protobuf from proto2 to proto3. I feel like this may be the same issues manifesting again.

The recommended way of dealing with this situation is to avoid re-using types. Buf, the creators of protovalidate heavily dogfood their own validation library in their own APIs. You can see that they separate types meant [for creating](https://buf.build/bufbuild/registry/docs/main:buf.registry.module.v1#buf.registry.module.v1.CreateModulesRequest) and types meant [for updating](https://buf.build/bufbuild/registry/docs/main:buf.registry.module.v1#buf.registry.module.v1.UpdateModulesRequest). This isn't super ideal because you likely want to keep most of the rules synchronized between both of these types but based on hard-earned experience, you often do have less problems if you simply create types for each use-case.

So instead of re-using a `User` type for 'create' and 'update' operations you could have another type for updating specific fields, like this:
```protobuf
message User {
    string name = 1;
    string email = 2;
    //... other fields
}

message UserProfileUpdate {
    optional string email = 1;
    optional string about_me = 2;
    //... other updatable fields
}
```

## What to take away
Protovalidate is a powerful and versatile tool that simplifies input validation for Protobuf-based applications. By defining validation rules directly in your `.proto` files, you can ensure data integrity, reduce boilerplate code, and streamline your development process. With its support for custom constraints using CEL expressions, Protovalidate offers flexibility and extensibility for various use cases. While it currently lacks direct integration for web frontends, the potential for a unified validation approach across frontend and backend is super exciting.

So, why wait? Dive into protovalidate today and discover how it can revolutionize your protobuf development workflow. If you're already leveraging ConnectRPC or gRPC in one of the supported languages, this library is a no-brainer to try.
