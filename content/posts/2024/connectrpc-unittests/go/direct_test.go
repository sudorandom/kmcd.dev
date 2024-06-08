package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	greetv1 "example/gen/greet/v1"
)

// start
func TestGreet(t *testing.T) {
	service := &greeterService{}
	response, err := service.Greet(context.Background(), connect.NewRequest(&greetv1.GreetRequest{Name: "Alice"}))
	if err != nil {
		t.Fatalf("Greet failed: %v", err)
	}
	if response.Msg.Greeting != "Hello, Alice" {
		t.Errorf("Unexpected greeting: got %q, want %q", response.Msg.Greeting, "Hello, Alice")
	}
}
