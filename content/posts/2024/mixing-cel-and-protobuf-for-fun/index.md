---
categories: ["article"]
tags: ["protobuf", "cel", "grpc", "testing"]
date: "2024-12-17T10:00:00Z"
description: ""
cover: "cover.jpg"
images: ["/posts/mixing-cel-and-protobuf-for-fun/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Mixing CEL and Protobuf for Fun"
slug: "mixing-cel-and-protobuf-for-fun"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/mixing-cel-and-protobuf-for-fun/
---

Protobufs offer a structured approach to data definition, but testing gRPC services built with them can be a hurdle. By leveraging FauxRPC and CEL, you can accelerate development cycles, enhance test coverage, and ensure the reliability of your microservices.

## Scaling Protobuf Testing

[gRPC](https://grpc.io/) has emerged as a powerful framework for building efficient and scalable microservices, enabling seamless communication across diverse technologies. Its language-agnostic nature, powered by Protobuf, allows services written in Go, Java, Python, or any other supported language to interact effortlessly.

However, while gRPC excels at interoperability, generating test data that conforms to Protobuf specifications can still be a hurdle and most tools focus on testing specific languages. Imagine this: you're building a gRPC service in Go, but your clients might be written in Python, Java, or even C++. Ensuring your tests cover a wide range of scenarios with valid Protobuf messages for all these languages can become quite cumbersome. Now consider that each language has its own tooling for generating and using stub data. While this probably works for most language, I feel like this fragmentation of the ecosystem goes against the mission of protobuf and gRPC. Instead, I think a better approach is a language agnostic approach.

This is where [**FauxRPC**](https://fauxrpc.com/) steps in, offering a streamlined approach to gRPC testing. Instead of crafting individual Protobuf responses for each test case, FauxRPC empowers you to define gRPC services that automatically generate realistic fake data for every response. This means you can effortlessly create a wide array of scenarios, simulate various data conditions, and thoroughly exercise your gRPC clients without the overhead of manually constructing complex Protobuf messages. However, this does fall short when you want to test specific test scenarios. That's where [FauxRPC stubs](https://fauxrpc.com/docs/server/stubs/) come in.

## The Pet Store Example

For the example, we are using [an example protobuf file from vanguard-go](https://github.com/connectrpc/vanguard-go/blob/main/internal/examples/pets/internal/proto/io/swagger/petstore/v2/pets.proto) for this demonstration.

### Creating a basic stub

First let's create a stub file at `get-pets.yaml` to stub the response to the `GetPetByID` method.
```yaml
---
stubs:
- id: get-pets-by-id-id-1
  target: io.swagger.petstore.v2.PetService/GetPetByID
  content:
    id: 1
    category:
      id: 1
      name: cat
    name: Whiskers
    photo_urls:
    - https://cataas.com/cat
    tags:
    - id: 1
      name: cute
    - id: 2
      name: kid-friendly
    status: available
```
This is a simple static stub. And will be returned any time `GetPetByID` is called.

Now start the FauxRPC server:
```shell
$ buf build ssh://git@github.com/connectrpc/vanguard-go.git -o petstore.binpb --path internal/examples/pets/internal/proto/io/swagger/petstore/v2
$ fauxrpc run --schema=petstore.binpb --only-stubs --stubs=get-pets.yaml
```

Notice that I actually ran two commands: the first one uses `buf build` to build a [buf image](https://buf.build/docs/reference/images/) using protobuf schema defined in the vanguard-go repo. The other actually starts the server. Let's break down the options used real quick:
- **`--schema=petstore.binpb`**: tells FauxRPC about the schema. We use the buf image from the previous command for this.
- **`--only-stubs`**: tells FauxRPC to only use pre-defined stubs and to avoid generating random data when there are no relevant stubs defined.
- **`--stubs=get-pets.yaml`**: tells FauxRPC to load stubs from the get-pets.yaml file from above. This option can be specified multiple times and if you give it a directory it will recursively look for stub files to use.

Now, when I hit the `GetPetByID` method, I now get Whiskers back.

```shell
$ buf curl --http2-prior-knowledge -d '{"pet_id": "1"}' http://127.0.0.1:6660/io.swagger.petstore.v2.PetService/GetPetByID
{
  "id": "1",
  "category": {
    "id": "1",
    "name": "cat"
  },
  "name": "Whiskers",
  "photoUrls": [
    "https://cataas.com/cat"
  ],
  "tags": [
    {
      "id": "1",
      "name": "cute"
    },
    {
      "id": "2",
      "name": "kid-friendly"
    }
  ],
  "status": "available"
}
```

If I were to add more stub entries, the response will be random amongst the applicable set of stubs. This is fine for some situations, but this also falls flat when you want to set up more complex test scenarios.

### Let's Improve This

Enter [**CEL** (Common Expression Language)](https://cel.dev/), a powerful and versatile expression language. With the latest release, FauxRPC now supports using CEL to define Protobuf messages. This unlocks a whole new level of conciseness, flexibility, and readability for your gRPC tests. Imagine creating dynamic messages, generating test data on the fly, and expressing complex scenarios with ease – all within your familiar Go environment.

So what does this mean? Well, there's three new attributes for FauxRPC stubs: `active_if` `priority` and `cel_content`. Let's improve upon the stubs that we defined above:

```yaml
---
stubs:
- id: get-pets-by-id-id-1
  target: io.swagger.petstore.v2.PetService/GetPetByID
  active_if: req.pet_id == 1
  priority: 100
  content:
    id: 1
    category:
      id: 1
      name: cat
    name: Whiskers
    photo_urls:
    - https://cataas.com/cat
    tags:
    - id: 1
      name: cute
    - id: 2
      name: kid-friendly
    status: available
- id: get-pets-by-id-default
  target: io.swagger.petstore.v2.PetService/GetPetByID
  cel_content: |
    {
        'id': req.pet_id,
        'category': {'id': gen, 'name': 'gen'},
        'name': gen,
        'photo_urls': [gen, gen],
        'tags': [{'id': gen, 'name': gen}],
        'status': gen
    }
```

Well, the first entry now has two new lines: `active_if: req.pet_id == 1` and `priority: 100`. The `active_if` property lets us define an expression to decide if this stub should be used or not. In this case, if the request has `pet_id == 1` then we will use this stub. The second option sets this stub to the highest priority so it will be considered first.

Next, we added a new stub that uses `cel_contents`. This new field defines a CEL expression to generate a response for us. In this case we use `req.pet_id` from the request. This demonstrates You can reference the request message in order to craft more believable response messages. For every other field we generate random responses using the special `gen` value. Using `gen` will use the field descriptor to decide what kinds of data should be generated.

Here's what it looks like to call this metheod with a `pet_id` set to something other than 2:
```shell
$ buf curl --http2-prior-knowledge -d '{"pet_id": "2"}' http://127.0.0.1:6660/io.swagger.petstore.v2.PetService/GetPetByID
{
  "id": "2",
  "category": {
    "id": "6775158014153383345",
    "name": gen
  },
  "name": "Mohammad",
  "photoUrls": [
    "https://picsum.photos/400",
    "https://picsum.photos/400"
  ],
  "tags": [
    {
      "id": "3433292194552134769",
      "name": "Chad"
    }
  ],
  "status": "deleted"
}
```

As you can see, this makes stubs much more dynamic. The output can be based on the input so at the very least the `id` field can match. Let's delve deeper into how CEL can be used to create more sophisticated and dynamic test scenarios.

#### Conditional Logic

Imagine a scenario where you want to return different responses based on specific conditions in the request. You could use CEL to define these conditions:

```yaml
- id: get-pets-by-id-conditional
  target: io.swagger.petstore.v2.PetService/GetPetByID
  cel_content: |
    {
        'id': req.pet_id,
        'category': {'id': 1, 'name': 'cat'},
        'name': 'Whiskers',
        'photoUrls': ['https://cataas.com/cat'],
        'tags': [{'id': 1, 'name': 'cute'}, {'id': 2, 'name': 'kid-friendly'}],
        'status': 'available'
    }å
    if req.pet_id == 1 else {
        'id': req.pet_id,
        'category': {'id': 2, 'name': 'dog'},
        'name': 'Buddy',
        'photoUrls': ['https://dog.ceo/api/breeds/image/random'],
        'tags': [{'id': 3, 'name': 'friendly'}, {'id': 4, 'name': 'playful'}],
        'status': 'pending'
    }
```

In this example, if the `pet_id` is 1, the first response is returned. Otherwise, the second response is returned.

#### Dynamic Data Generation

CEL can be used to generate dynamic data based on various factors, such as integers, usernames, sentences, random numbers, or values from the request:

```yaml
- id: get-pets-by-id-dynamic
  target: io.swagger.petstore.v2.PetService/GetPetByID
  cel_content: |
    {
        'id': req.pet_id,
        'category': [{'id': 1, 'name': 'cat'}, {'id': 2, 'name': 'dog'}][fake_int() % 2],
        'name': ['Mr', 'Ms'][fake_int() % 2] + ' ' + fake_first_name(),
        'photoUrls': ['https://picsum.photos/200'],
        'tags': [{'id': gen, 'name': gen}],
        'status': ['available', 'pending', 'sold'][fake_int() % 3]
    }
```

- **`id`**: This represents the unique identifier of a pet. It takes its value directly from `req.pet_id`, which is pulled the pet ID from the incoming request.
- **`category`**: This defines the pet's category. `fake_int() % 2` is used to randomly select one of the categories since the result of the modulo operation will be either 0 or 1.
- **`name`**: This generates a random pet name. It combines 'Mr' or 'Ms' (chosen randomly using `fake_int() % 2`) with a fake first name generated by the `fake_first_name()` function.
- **`photoUrls`**: This field is meant to hold an array of URLs pointing to pet photos. In this case, it's a single-element array with a placeholder URL from `https://picsum.photos/200` (which generates a random image).
- **`tags`**: This defines an array of tags associated with the pet. `gen` is used to generate a random (but type appropriate) value for a single tag's ID and name fields.
- **`status`**: This indicates the pet's current status. It randomly selects one of three possible values ('available', 'pending', or 'sold') using `fake_int() % 3`.

All of the functions starting with `fake_` are provided by a new FauxRPC package, [celfakeit](https://github.com/sudorandom/fauxrpc/tree/main/celfakeit). This library exposes many functions from [gofakeit](https://github.com/brianvoe/gofakeit) as CEL functions.

## Recap

Since I've been making a lot of changes and improvements, it seems like a good idea to recap how FauxRPC works and how it's different from other similar tools.

- FauxRPC does NOT require code generation. It takes in [protobuf files](https://protobuf.dev/), [descriptors](https://buf.build/docs/reference/descriptors/), or [buf images](https://buf.build/docs/reference/images/).
- FauxRPC is meant to be language agnostic. This was important to me since gRPC is often used across many languages so tying it to just Go, for example, adds un-needed ecosystem fragmentation.
- FauxRPC has [protovalidate support](https://kmcd.dev/posts/fauxrpc-protovalidate/)
- FauxRPC [works with testcontainers](https://kmcd.dev/posts/fauxrpc-testcontainers/). There is a [built in library](https://pkg.go.dev/github.com/sudorandom/fauxrpc/testcontainers) for Go (see [the testcontainer tests](https://github.com/sudorandom/fauxrpc/blob/main/testcontainers/testcontainers_test.go) for examples of how this can be used). It shouldn't be too hard to create similar libraries in other languages.
- You can [now return pre-defined stubs](https://fauxrpc.com/docs/server/stubs/) which can include errors, static responses, or dynamic (CEL-based) content.

{{< diagram >}}
{{< image src="diagram.svg" width="800px" class="center" >}}
{{< /diagram >}}

## Conclusion

In conclusion, the combination of Protobuf and CEL offers a powerful and flexible approach to testing gRPC services. By leveraging CEL's expressive power, developers can define dynamic and realistic test data, significantly enhancing the efficiency and effectiveness of their testing efforts.

FauxRPC, with the integration of CEL, simplifies the process of generating complex Protobuf messages, allowing developers to focus on writing robust and reliable gRPC services. By embracing this innovative approach, teams can accelerate their development cycles and deliver high-quality gRPC applications with confidence.

*... and don't forget to [star the repo on GitHub](https://github.com/sudorandom/fauxrpc). It helps more than you know!*
