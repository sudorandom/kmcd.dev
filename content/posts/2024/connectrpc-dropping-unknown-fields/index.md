---
categories: ["tutorial"]
tags: ["connectrpc", "protobuf", "rest", "api", "rpc", "grpc", "http"]
date: "2024-04-02"
description: "Learn how to drop unknown fields in ConnectRPC to enhance the security of your gRPC services exposed to the internet."
cover: "cover.jpg"
images: ["/posts/connectrpc-dropping-unknown-fields/cover.jpg"]
title: "Dropping Unknown Fields in ConnectRPC"
slug: "connectrpc-dropping-unknown-fields"
type: "posts"
devtoId: 1808828
devtoPublished: true
devtoSkip: false
canonical_url: https://sudorandom.dev/posts/connectrpc-dropping-unknown-fields
mastodonID: "112277345241838737"
---

gRPC, with its focus on performance and language neutrality, remains a popular choice for building microservices and APIs. But when exposing your gRPC service to the internet, there are a few security considerations to account for. Protobuf, the serialization format often used with gRPC, offers various encoding options that can significantly impact your service's security posture. 

One crucial optimization for internet-facing gRPC services is customizing the behavior towards **unknown fields**. I've talked about [unknown fields in a previous post](/posts/protobuf-unknown-fields/), so read that one if unknown fields are still a mystery to you and then come back here. By default, protobuf messages can contain fields that are not defined in the current version of the proto schema. While convenient for development and can help with forward compatibility, this poses a security risk in a public environment.

Here's why you should consider dropping unknown fields when exposing gRPC to the internet:

* **Preventing Malicious Data:** Unknown fields can be exploited by malicious actors to inject unexpected data into your service. This could lead to potential security vulnerabilities like code injection or unexpected behavior.
* **Ensuring Compatibility:** Uncontrolled unknown fields can cause compatibility issues if your clients are using different versions of the proto schema. Dropping them enforces stricter adherence to the defined message format.
* **Improving Performance:** Skipping unknown fields during message parsing can lead to performance gains, especially when dealing with large datasets.

### How to Drop Unknown Fields

Here is how you can drop unknown fields while using the standard `proto.UnmarshalOptions` struct provided by the `google.golang.org/protobuf/proto` package. Here's how to do it in your Go code:

```go
import (
	"google.golang.org/protobuf/proto"
	...
)

// Configure unmarshalling options to discard unknown fields
opts := proto.UnmarshalOptions{
	DiscardUnknown: true,
}

// Use the options when unmarshalling incoming messages
msg := &MyMessage{}
err := proto.Unmarshal(data, msg, opts)
if err != nil {
	// Handle error
}
```

By setting the `DiscardUnknown` field to `true` in the `proto.UnmarshalOptions` struct before unmarshalling incoming messages, you ensure that any unknown fields are ignored. This helps mitigate the security risks associated with unknown fields while processing internet-facing gRPC requests.

## How to Drop Unknown Fields in Connect RPC Servers

```go
package main

import (
	"log"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"go.akshayshah.org/connectproto"
)

func main() {
	greeter := &GreetServer{}
	mux := http.NewServeMux()
	path, handler := greetv1connect.NewGreetServiceHandler(
		greeter,
		// Add an option that customizes protobuf marshalling/unmarshalling behavior
		connectproto.WithBinary(
			proto.MarshalOptions{},
			proto.UnmarshalOptions{DiscardUnknown: true},
		),
		// Add an option to customize JSON marshalling/unmachalling
		connectproto.WithJSON(
			protojson.MarshalOptions{},
			protojson.UnmarshalOptions{DiscardUnknown: true},
		)
	)
	mux.Handle(path, handler)
	log.Fatal(http.ListenAndServe(
		"localhost:9000",
		h2c.NewHandler(mux, &http2.Server{}),
	))
}
```
In this example, `connectproto.WithBinary` ensures only messages with defined fields are processed, enhancing the security of your gRPC service. `connectproto.WithJSON` does the same thing but with JSON.

### Additional Considerations

While dropping unknown fields is a valuable security practice, it's important to consider potential trade-offs:

* **Backward compatibility:** Clients using older versions of the proto schema will encounter errors if they rely on previously defined unknown fields. 
* **Logging and Debugging:** Dropping unknown fields might make it harder to identify the source of unexpected behavior during development or debugging.

In such cases, it's recommended to document these trade-offs and have a clear versioning policy for your gRPC service and client applications.

### Conclusion

Exposing gRPC services to the internet requires careful security considerations. By customizing protobuf encoding options, specifically by dropping unknown fields using `proto.UnmarshalOptions`, you can significantly improve the security posture of your service. Remember to weigh the benefits against potential drawbacks and implement a solution that aligns with your specific needs.
