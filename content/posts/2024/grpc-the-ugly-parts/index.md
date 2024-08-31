---
categories: ["article"]
tags: ["grpc", "protobuf", "api", "rpc", "webdev", "humor", "http2", "http3"]
series: ["gRPC: the good and the bad"]
date: "2024-09-03"
description: "The seedy underbelly of gRPC."
cover: "cover.jpg"
images: ["/posts/grpc-the-ugly-parts/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "gRPC: The Ugly Parts"
slug: "grpc-the-ugly-parts"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/grpc-the-ugly-parts/
draft: true
---

gRPC has undeniably become a powerful tool in the world of microservices, offering efficiency and performance benefits, but gRPC also has an ugly side. As someone who's spent a considerable amount of time with gRPC, I'd like to shed light on some of the uglier aspects of this technology.

### Generated Code Looks SO Ugly
To get started, I have to talk about how the generated code from Protobuf definitions and gRPC services can be truly ugly. It's often verbose, complex, and difficult to navigate. Even though it's not meant to be hand-edited, this can impact code readability and maintainability, especially when integrating gRPC into larger projects. This has improved a lot in some languages but even so, there are often some rough edges.

- **C++ Influence on Enums:** Protobuf's enum handling, with its strong typing and prefixing, can lead to cumbersome code. This reflects its C++ heritage, which might not be the most elegant solution for all programming languages.

```protobuf
enum FooBar {
  FOO_BAR_UNSPECIFIED = 0;
  FOO_BAR_FIRST_VALUE = 1;
  FOO_BAR_SECOND_VALUE = 2;
}
```
Why would scoping inside of the enum not be enough for the C++ compiler to generate unique names? Why is this flaw something that impacts the style guide? This is described better in the [buf lint rule](https://buf.build/docs/lint/rules#enum_value_prefix) for `ENUM_VALUE_PREFIX`.

### Editor Support with Code Generation Sucks
Editor integration for Protobuf code generation leaves a lot to be desired. It would be immensely helpful if editors could intelligently link generated code back to its Protobuf source. This would provide a more seamless experience, but the tooling just isn't smart enough yet. Also, I think everyone needs to run with [Buf's editor support](https://buf.build/docs/editor-integration). Having a linter built into your editor is the bare requirement nowadays. And with protobuf, there are [extremely real reasons](https://buf.build/docs/lint/rules) to follow the advice of the linter.

### Failure to Launch
While gRPC has undeniable advantages, its learning curve can be steep. Getting started with protobuf, understanding the tooling, and setting up the necessary infrastructure can be intimidating for newcomers, making the initial adoption hurdle higher than with simpler JSON-based APIs.

{{< image src="learning-curve.png" width="600px" class="center" >}}

The steep learning curve doesn't help when many people who use and rely on protobuf and gRPC actively don't want gRPC to extend to the frontend and think tools to help smooth out the transition from JSON to protobuf will lead to uninformed people encroaching on the magical backend domain. This is elitist gate-keeping and is unfortunately prevalent in this industry.

### gRPC Has a History
gRPC's initial focus on microservices and its close ties to HTTP/2 hindered its widespread adoption in web development. Even with the advent of gRPC-Web, there's still a perception that it's not a first-class citizen in the frontend ecosystem. The lack of robust integration with popular frontend libraries like TanStack Query further solidifies this notion.

{{< image src="bad-blood.png" width="800px" class="center" >}}

I think there's a real chance to get more frontend developers excited about gRPC with improved tooling. There's a giant industry-wide conversation happening right now around where the line between "frontend" and "backend" meet and I think no matter the outcome, we're going to see more typescript code using gRPC.

### The "g" in gRPC
While [the gRPC project claims](https://grpc.io/docs/what-is-grpc/faq/#what-does-grpc-stand-for) that the "g" in gRPC is a [backronym](https://en.wikipedia.org/wiki/Backronym) that stands for "gRPC", it originally stood for Google, because it was Google who developed and released both protobuf and gRPC.

{{< image src="google.png" width="600px" class="center" >}}

There's always a lingering question about Google's long-term commitment to gRPC and Protobuf. Will they continue to invest in these open-source projects, or could they pull the plug if priorities shift? Remember that Google has [recently layed off much of the Flutter, Dart and Python teams](https://techcrunch.com/2024/05/01/google-lays-off-staff-from-flutter-dart-python-weeks-before-its-developer-conference/). The Protobuf community is growing, but would it be self-sustaining enough to survive such a scenario?

### It's Not Finished
Others have said that gRPC is immature, not because of its age but by how developed the ecosystem is. I tend to agree, because it's missing features and tools that I would have expected from a mature product like this.

#### The missing package manager
Sharing protobuf definitions across multiple projects or repositories is a constant struggle without specialized tools. While solutions like [Bazel](https://bazel.build/reference/be/protocol-buffer), [Pants](https://www.pantsbuild.org/2.21/docs/go/integrations/protobuf), and [Buf's BSR](https://buf.build/product/bsr) exist, my experience with protobuf "in the real world"... mixed. There are prominent open source projects, some by Google, that have bash scripts scrapped together to download dependencies before evoking `protoc` manually. Just imagine if that's how any language's solution to package management worked. That's insane. I do think [Bazel](https://grpc.io/blog/bazel-rules-protobuf/) and [Buf tooling](https://buf.build/docs/ecosystem/cli-overview) solve this problem very well. I'm just frustrated that every repo I come across that uses protobuf solves the problem in the most bespoke way possible. The community needs to come together to improve this.

{{< image src="build.png" width="600px" class="center" >}}

Related to dependencies, I do want to call out that Google's "well-known" protobuf types get special privilege of being built into protoc. While these types are incredibly useful and are invaluable, their privilege makes it hard for other libraries of useful protobuf types to exist and thrive.

#### Required Fields
The maintainers of protobuf learned some hard lessons with required fields. They felt like they misstepped so badly, that they made a new version of protobuf, proto3, just to remove required fields from the spec. Why? The author of the "Required considered harmful" manifesto talks about this in a [lengthy hacker news comment](https://news.ycombinator.com/item?id=18190005), but the important bit is:

> Real-world practice has also shown that quite often, fields that originally seemed to be "required" turn out to be optional over time, hence the "required considered harmful" manifesto. In practice, you want to declare all fields optional to give yourself maximum flexibility for change.

This is [echoed by the official style guide of protobufs](https://protobuf.dev/programming-guides/dos-donts/#add-required), where they recommend adding a comment indicating that a field is required. If we're talking about getting a message from A to B, I totally agree with this line of thinking. However, just because which fields are "required" change over time doesn't mean they don't exist. There still needs to be code that enforces this requirement and I'd rather not write this code, to be honest. Therefore, I think the best way of handling required fields without writing a bunch of null checks everywhere is by using [protovalidate](https://github.com/bufbuild/protovalidate) or a similar library that has protobuf options that allow you to annotate which fields are required. Then there is code on the server and/or client that can enforce these requirements using a library. In my opinion, this has the best of both worlds: you can still declare required fields in a way that doesn't completely break message integrity.

I'm a big fan of [protovalidate](https://github.com/bufbuild/protovalidate) and I've used it a good amount. I've contributed to it. I now have two open source projects that use these field annotations to do useful work.

## Documentation is Ugly
I've never seen documentation generated from protobuf that wasn't super ugly. I think since gRPC has historically been a backend service, the backend devs never bothered to put any real effort into making pretty documentation output using a protoc plugin. I've solved this problem by [making a protoc plugin](https://github.com/sudorandom/protoc-gen-connect-openapi) that generates OpenAPI from given protobuf files. Then I use one of the many beautiful tools for displaying the OpenAPI spec. This was, by far, much easier than getting me to make a decent design.

Compare this (protoc-gen-doc):
{{< image src="protoc-gen-doc.png" width="800px" class="center" >}}

to this:
{{< image src="elements.png" width="800px" class="center" >}}

## Conclusion
gRPC is a powerful tool, but it's not without its flaws. From the ugliness of generated code to the challenges of sharing Protobuf definitions, there are areas where gRPC could be improved. The good news is that the community is actively working on solutions, and as gRPC continues to mature, we can expect a smoother and more enjoyable developer experience. 
