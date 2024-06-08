package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	greetv1 "example/gen/greet/v1"
	"example/gen/greet/v1/greetv1connect"
)

// start
func TestGreetWithServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(greetv1connect.NewGreetServiceHandler(&greeterService{}))

	server := httptest.NewServer(mux)
	t.Cleanup(func() { server.Close() })

	client := greetv1connect.NewGreetServiceClient(http.DefaultClient, server.URL)

	response, err := client.Greet(context.Background(), connect.NewRequest(&greetv1.GreetRequest{Name: "Alice"}))
	if err != nil {
		t.Fatalf("Greet failed: %v", err)
	}
	if response.Msg.Greeting != "Hello, Alice" {
		t.Errorf("Unexpected greeting: got %q, want %q", response.Msg.Greeting, "Hello, Alice")
	}
}
