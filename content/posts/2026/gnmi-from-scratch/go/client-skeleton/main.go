package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"gnmi-scratch/gen/gnmi/v1/gnmiv1"
	"gnmi-scratch/gen/gnmi/v1/gnmiv1connect"
)

func main() {
	client := gnmiv1connect.NewGNMIServiceClient(
		http.DefaultClient,
		"http://localhost:8080",
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("--- Calling Get RPC ---")
	// TODO: Build and send GetRequest

	log.Println("\n--- Calling Subscribe RPC ---")
	// TODO: Build and send SubscribeRequest, then loop on responses
}
