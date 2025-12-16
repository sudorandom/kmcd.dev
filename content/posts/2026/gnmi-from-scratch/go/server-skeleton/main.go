package main

import (
	"context"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"gnmi-scratch/gen/gnmi/v1/gnmiv1"
	"gnmi-scratch/gen/gnmi/v1/gnmiv1connect"
)

type gnmiServer struct{}

func (s *gnmiServer) Capabilities(ctx context.Context, req *connect.Request[gnmiv1.CapabilityRequest]) (*connect.Response[gnmiv1.CapabilityResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *gnmiServer) Get(ctx context.Context, req *connect.Request[gnmiv1.GetRequest]) (*connect.Response[gnmiv1.GetResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *gnmiServer) Subscribe(ctx context.Context, stream *connect.BidiStream[gnmiv1.SubscribeRequest, gnmiv1.SubscribeResponse]) error {
	return connect.NewError(connect.CodeUnimplemented, nil)
}

func main() {
	server := &gnmiServer{}
	mux := http.NewServeMux()
	path, handler := gnmiv1connect.NewGNMIServiceHandler(server)
	mux.Handle(path, handler)
	log.Println("Starting gNMI server on :8080...")
	err := http.ListenAndServe("localhost:8080", h2c.NewHandler(mux, &http2.Server{}))
	if err != nil {
		log.Fatalf("listen and serve failed: %v", err)
	}
}
