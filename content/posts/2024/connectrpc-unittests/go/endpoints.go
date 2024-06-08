package main

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	greetv1 "example/gen/greet/v1"
	"example/gen/greet/v1/greetv1connect"
)

// start
type greeterService struct{}

var _ greetv1connect.GreetServiceHandler = (*greeterService)(nil)

func (g *greeterService) Greet(ctx context.Context, req *connect.Request[greetv1.GreetRequest]) (*connect.Response[greetv1.GreetResponse], error) {
	if req.Msg.Name == "" {
		return nil, errors.New("missing name")
	}

	// Simulate some network call that takes 10ms
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(10 * time.Millisecond):
	}
	return connect.NewResponse(&greetv1.GreetResponse{Greeting: "Hello, " + req.Msg.Name}), nil
}
