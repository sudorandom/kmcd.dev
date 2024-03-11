+++
categories = ["project"]
tags = ["connectrpc", "grpc", "protobuf"]
date = "2024-03-19"
description = "unknownconnect-go is library that helps developers using gRPC identify compatibility issues caused by mismatched message definitions."
cover = "cover.jpg"
images = ["/posts/unknownconnect-go/cover.jpg"]
featured = ""
featuredalt = ""
featuredpath = "date"
linktitle = ""
title = "Introducing unknownconnect-go"
slug = "unknownconnect-go"
type = "posts"
+++

gRPC systems can be quite complex. When making additions to protobuf files the server or the client often gets updated at different times. In a perfect world, this would all be synchronized. But we live in reality. Sometimes release schedules differ between components. Sometimes you just forget to update a component. Many times you might be consuming a gRPC service managed by another team and *they don't tell you that they're changing things*. I made something that will bring unique insight into this problem with very little work.

## Let's make things better

[unknownconnect-go](https://github.com/sudorandom/unknownconnect-go) is an interceptor for [ConnectRPC](https://connectrpc.com/) clients and servers that tells you if you are receiving protobuf messages with unknown fields. Now you can know when you should upgrade your gRPC clients or servers to the latest version. Let's discuss how to use it.

1. **Install the library:**

```bash
go get -u github.com/sudorandom/unknownconnect-go
```

2. **Import the library:**

```go
import (
    unknownconnect "github.com/sudorandom/unknownconnect-go"
)
```

## Server-side usage

Here are two examples demonstrating how to use `unknownconnect-go` on the server-side:

**Short example:**

```go
unknownconnect.NewInterceptor(func(ctx context.Context, spec connect.Spec, msg proto.Message) error {
    slog.Warn("received a protobuf message with unknown fields", slog.Any("spec", spec), slog.Any("msg", msg))
    return nil
})
```

This example creates a new interceptor using `unknownconnect.NewInterceptor`. The interceptor function receives three arguments:

* `ctx`: The [context](https://pkg.go.dev/context#Context) object
* `spec`: The [gRPC service specification](https://pkg.go.dev/connectrpc.com/connect#Spec)
* `msg`: The received [protobuf message](https://pkg.go.dev/google.golang.org/protobuf/proto#Message). Note that the actual message with unknown field(s) can be nested deeper within this message.

In the previous example, when a message with unknown fields is received, the interceptor will log a warning message using the `slog.Warn` function. It includes information about the message specification and the message itself. 

**Full example:**
Here is a full example that shows you how to register the `unknownconnect.Interceptor` with a ConnectRPC handler:
```go
func main() {
    greeter := &GreetServer{}
    mux := http.NewServeMux()
    path, handler := greetv1connect.NewGreetServiceHandler(greeter, connect.WithInterceptors(
        unknownconnect.NewInterceptor(func(ctx context.Context, spec connect.Spec, msg proto.Message) error {
            return connect.NewError(connect.InvalidArgument, errors.New("protobuf version missmatch; received unknown fields"))
        }),
    ))
    mux.Handle(path, handler)
    http.ListenAndServe("localhost:8080", h2c.NewHandler(mux, &http2.Server{}))
}
```

The interceptor function in this example returns an error with the `connect.InvalidArgument` code, which will cause the server to reject the request if it receives a message with unknown fields.

**Customization options:**

The two examples above show two ways to handle messages with unknown fields but you can customize the behavior of the interceptor to suit your specific needs. Here are some ideas:

* **Log the event:** As shown in the first example, you can simply log a warning message when an unknown field is encountered. This can help debug and monitor the cause.
* **Add to a metric** With this approach, you can emit metrics whenever unknown fields are encountered. This can be helpful to give more monitoring insight.
* **Fail the request/response:** This approach, demonstrated in the second example, can be useful in pre-production environments to prevent unexpected behavior caused by mismatched message definitions.
* **Add an annotation to the context:** This allows you to pass information about the unknown field to your service handler.

## Client-side usage

And it works the same for clients, too:

```go
package main

import (
    "context"
    "log/slog"
    "net/http"

    greetv1 "example/gen/greet/v1"
    "example/gen/greet/v1/greetv1connect"

    "connectrpc.com/connect"
)

func main() {
    client := greetv1connect.NewGreetServiceClient(
        http.DefaultClient,
        "http://localhost:8080",
        connect.WithInterceptors(
            unknownconnect.NewInterceptor(func(ctx context.Context, spec connect.Spec, msg proto.Message) error {
                slog.Warn("received a protobuf message with unknown fields", slog.Any("spec", spec), slog.Any("msg", msg))
                return nil
            })
        ),
    )
    res, err := client.Greet(
        context.Background(),
        connect.NewRequest(&greetv1.GreetRequest{Name: "Jane"}),
    )
    if err != nil {
        slog.Error(err.Error())
        return
    }
    slog.Info(res.Msg.Greeting)
}
```

This example works in a similar way to how the server interceptor. It creates a new gRPC client for the Greet service and adds the unknownconnect interceptor using the `connect.WithInterceptors` function.

## Conclusion

`unknownconnect-go` provides a simple and effective way to identify potential compatibility issues in your gRPC systems by detecting messages with unknown fields. It offers flexibility in how you handle these situations, allowing you to log warnings, reject requests, or implement custom logic as needed. By integrating `unknownconnect-go` into your development workflow, you can gain valuable insights into potential version mismatches and ensure smoother operation of your gRPC systems.

GitHub Link: [github.com/sudorandom/sudorandom.dev](https://github.com/sudorandom/sudorandom.dev/)
