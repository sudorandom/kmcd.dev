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

	// --- 1. Call Get ---
	log.Println("--- Calling Get RPC ---")
	getPath := &gnmiv1.Path{
		Elem: []*gnmiv1.PathElem{{Name: "system"}, {Name: "cpu"}, {Name: "utilization"}},
	}
	getReq := &gnmiv1.GetRequest{
		Path: []*gnmiv1.Path{getPath},
		Encoding: gnmiv1.Encoding_JSON_IETF,
	}
	getResp, err := client.Get(ctx, connect.NewRequest(getReq))
	if err != nil {
		log.Fatalf("Get failed: %v", err)
	}
	log.Printf("Get Response: %s", getResp.Msg.String())


	// --- 2. Call Subscribe ---
	log.Println("\n--- Calling Subscribe RPC ---")
	subPath := &gnmiv1.Path{
		Elem: []*gnmiv1.PathElem{{Name: "system"}, {Name: "memory"}, {Name: "state"}, {Name: "used"}},
	}
	subReq := &gnmiv1.SubscribeRequest{
		Request: &gnmiv1.SubscribeRequest_Subscribe{
			Subscribe: &gnmiv1.SubscriptionList{
				Subscription: []*gnmiv1.Subscription{
					{
						Path: subPath,
						Mode: gnmiv1.SubscriptionList_STREAM,
					},
				},
				Mode: gnmiv1.SubscriptionList_STREAM,
			},
		},
	}

	stream := client.Subscribe(ctx)
	if err := stream.Send(subReq); err != nil {
		log.Fatalf("Failed to send subscription request: %v", err)
	}

	for stream.Receive() {
		resp := stream.Msg()
		update := resp.GetUpdate()
		if update != nil {
			val := update.GetUpdate()[0].GetVal().GetJsonIetfVal()
			log.Printf("Subscribe Response: Path=%s, Value=%s", gnmiv1.PathToString(update.GetUpdate()[0].GetPath()), string(val))
		}
	}

	if err := stream.Err(); err != nil {
		log.Fatalf("Stream ended with error: %v", err)
	}
}
