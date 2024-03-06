+++
categories = ["opinion"]
tags = ["connectrpc", "grpc"]
date = "2024-03-19"
description = ""
cover = "cover.jpg"
images = ["/posts/unknownconnect-go/cover.jpg"]
featured = ""
featuredalt = ""
featuredpath = "date"
linktitle = ""
title = "Introducing unknownconnect-go"
slug = "unknownconnect-go"
type = "posts"
draft = true
+++

gRPC systems can be quite complex. When making additions to protobuf files the server or the client often gets updated at different times. In a perfect world, this would all be synchronized. But we live in reality. Sometimes release schedules differ between components. Sometimes you just forget to update a component. Many times you might be consuming a gRPC service managed by another team and *they don't tell you that they're changing things*. I made something that will bring unique insight into this problem with very little work.

# Introducing unknownconnect-go

[unknownconnect-go](https://github.com/sudorandom/unknownconnect-go) is an interceptor for [ConnectRPC](https://connectrpc.com/) clients and servers that tells you if you are receiving protobuf messages with unknown fields. Now you can know when you should upgrade your gRPC clients or servers to the latest version. Let's discuss how to use it.

```bash
go get -u github.com/sudorandom/unknownconnect-go
```

## Server Examples
Short example:
```go
import (
    "log/slog"

    unknownconnect "github.com/sudorandom/unknownconnect-go"
)

...
unknownconnect.NewInterceptor(func(ctx context.Context, spec connect.Spec, msg proto.Message) error {
    slog.Warn("received a protobuf message with unknown fields", slog.Any("spec", spec), slog.Any("msg", msg))
    return nil
})
```

Full:
```go
import (
    "log/slog"

    "connectrpc.com/connect"
    unknownconnect "github.com/sudorandom/unknownconnect-go"
)

func main() {
    greeter := &GreetServer{}
    mux := http.NewServeMux()
    path, handler := greetv1connect.NewGreetServiceHandler(greeter, connect.WithInterceptors(
        unknownconnect.NewInterceptor(func(ctx context.Context, spec connect.Spec, msg proto.Message) error {
            return connect.NewError(connect.InvalidArgument, err)
        }),
    ))
    mux.Handle(path, handler)
    http.ListenAndServe("localhost:8080", h2c.NewHandler(mux, &http2.Server{}))
}
```

The first example simply emits a warning log and the second example will fail the request if the server receives a message with unknown fields. You can decide what to do. Here are some ideas:

- Add to a metric that counts how often this happens
- Fail the request/response; maybe the most useful in non-production integration environments
- Emit a log
- Add an annotation to the context to be used in the handler
- ???

## Client Examples
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
