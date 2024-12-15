---
categories: ["article"]
tags: ["protobuf", "grpc", "testing"]
date: "2024-09-03"
description: ""
cover: "cover.jpg"
images: ["/posts/faking-protobuf-data-in-go/cover.jpg"]
featured: ""
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Faking protobuf data in Go"
slug: "faking-protobuf-data-in-go"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/faking-protobuf-data-in-go/
draft: true
---

In the realm of Go development, working with Protocol Buffers (protobuf) and gRPC often involves creating mock or test data. FauxRPC contains a handy [Go library](https://pkg.go.dev/github.com/sudorandom/fauxrpc) that simplifies this process, allowing you to generate fake protobuf messages effortlessly. Let's explore how FauxRPC streamlines protobuf data generation, especially when working with generated protobuf code or dynamic protobuf environments.

## Core FauxRPC Functions
FauxRPC offers two primary functions to cater to different use cases:

- **[fauxrpc.SetDataOnMessage](https://pkg.go.dev/github.com/sudorandom/fauxrpc#SetDataOnMessage)**: Populates a concrete protobuf message with fake data. This is ideal when you have a specific protobuf message type generated from your .proto files.
- **[fauxrpc.NewMessage](https://pkg.go.dev/github.com/sudorandom/fauxrpc#NewMessage)**: Creates a new protobuf message based on a MessageDescriptor. This is useful in dynamic scenarios where you might not have concrete message types available at compile time.

## Generating Fake Protobuf Data
Here's how you can generate a message filled with fake data, using the Eliza service example.

```go
package main

import (
	"fmt"
	"log"

	elizav1 "buf.build/gen/go/connectrpc/eliza/protocolbuffers/go/connectrpc/eliza/v1"
	"github.com/sudorandom/fauxrpc"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	msg := &elizav1.SayResponse{}
	fauxrpc.SetDataOnMessage(msg)
	b, err := protojson.MarshalOptions{Indent: "  "}.Marshal(msg)
	if err != nil {
		log.Fatalf("err: %s", err)
	}
	fmt.Println(string(b))
}
```

Example output:
```json
{
  "sentence": "Jean shorts."
}
```

This text will be randomly generated each time. That's nice, but Here's one that's a bit more complex.
```go
package main

import (
	"fmt"
	"log"

	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	"github.com/sudorandom/fauxrpc"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
		msg := &ownerv1.Owner{}
		fauxrpc.SetDataOnMessage(msg)
		b, err := protojson.MarshalOptions{Indent: "  "}.Marshal(msg)
		if err != nil {
			log.Fatalf("err: %s", err)
		}
		fmt.Println(string(b))
}
```
In this example, `fauxrpc.SetDataOnMessage` fills the `ownerv1.Owner` message with randomly generated data. The protojson package then formats the output for readability.

Example output:
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

### dynamicpb
Sometimes you don't want to have the compiled protobuf code because you want the proxy or service to be more dynamic. In that case, you typically would use [dynamicpb](https://pkg.go.dev/google.golang.org/protobuf/types/dynamicpb). FauxRPC works there, too. Here's an example:

```go
package main

import (
	"fmt"
	"log"

	elizav1 "buf.build/gen/go/connectrpc/eliza/protocolbuffers/go/connectrpc/eliza/v1"
	"github.com/sudorandom/fauxrpc"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
    msg := fauxrpc.NewMessage(elizav1.File_connectrpc_eliza_v1_eliza_proto.Messages().ByName("SayResponse"))
    b, err := protojson.MarshalOptions{Indent: "  "}.Marshal(msg)
    if err != nil {
        log.Fatalf("err: %s", err)
    }
    fmt.Println(string(b))
}
```

### Implementing the server handlers (ConnectRPC)

```go
import (
    ...
    "github.com/sudorandom/fauxrpc"
    "buf.build/gen/go/connectrpc/eliza/connectrpc/go/connectrpc/eliza/v1/elizav1connect"
    elizav1 "buf.build/gen/go/connectrpc/eliza/protocolbuffers/go/connectrpc/eliza/v1"   
)
...

func (h *handler) Say(ctx context.Context, req *connect.Request[elizav1.SayRequest]) (*connect.Response[elizav1.SayResponse], error) {
    msg := &elizav1.SayResponse{}
    fauxrpc.SetDataOnMessage(msg) // Populate with fake data
    return connect.NewResponse(msg), nil
}

...
```
In this snippet, the Say handler generates a fake SayResponse using FauxRPC.

Example output:
```shell
$ buf curl --schema=buf.build/connectrpc/eliza \
         -d '{"sentence": "Hello world!"}' \
         http://127.0.0.1:6660/connectrpc.eliza.v1.ElizaService/Say
{
  "sentence": "Microdosing."
}
```

### Implementing the server handlers (grpc-go)

```go
import (
    ...
    "github.com/sudorandom/fauxrpc"
    "buf.build/gen/go/connectrpc/eliza/connectrpc/go/connectrpc/eliza/v1/elizav1connect"
    elizav1 "buf.build/gen/go/connectrpc/eliza/protocolbuffers/go/connectrpc/eliza/v1"   
)
...
func (s *server) Say(ctx context.Context, req *elizav1.SayRequest) (*elizav1.SayResponse, error) {
	msg := &elizav1.SayResponse{}
	fauxrpc.SetDataOnMessage(msg, fauxrpc.GenOptions{})
	return msg, nil
}
...
```
In this snippet, the Say handler generates a fake SayResponse using FauxRPC.

Example output:
```shell
$ buf curl --schema=buf.build/connectrpc/eliza \
         -d '{"sentence": "Hello world!"}' \
         --protocol=grpc --http2-prior-knowledge \
         http://127.0.0.1:6660/connectrpc.eliza.v1.ElizaService/Say
{
  "sentence": "Microdosing."
}
```

There are other functions to help generate specific fields. Please reference the [reference documentation](https://pkg.go.dev/github.com/sudorandom/fauxrpc) for more information on that.
