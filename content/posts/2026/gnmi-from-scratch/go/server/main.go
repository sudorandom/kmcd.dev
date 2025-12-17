package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	gnmiv1 "github.com/sudorandom/kmcd.dev/gnmi/gen/gnmi"
	gnmiv1connect "github.com/sudorandom/kmcd.dev/gnmi/gen/gnmi/gnmiconnect"
)

var _ gnmiv1connect.GNMIHandler = (*gnmiServer)(nil)

type gnmiServer struct {
	// In a real server, you'd have a data store, cache, etc.
}

func (s *gnmiServer) Capabilities(
	ctx context.Context,
	req *connect.Request[gnmiv1.CapabilityRequest],
) (*connect.Response[gnmiv1.CapabilityResponse], error) {
	// A real server would list its supported models and encodings.
	resp := &gnmiv1.CapabilityResponse{
		SupportedModels:    []*gnmiv1.ModelData{},
		SupportedEncodings: []gnmiv1.Encoding{gnmiv1.Encoding_JSON},
		GNMIVersion:        "0.7.0",
	}
	return connect.NewResponse(resp), nil
}

func (s *gnmiServer) Get(
	ctx context.Context,
	req *connect.Request[gnmiv1.GetRequest],
) (*connect.Response[gnmiv1.GetResponse], error) {
	// This is a simplified Get. A real one would parse paths and fetch data.
	// We'll just return a mock CPU value if asked.
	notification := &gnmiv1.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*gnmiv1.Update{
			{
				Path: req.Msg.GetPath()[0], // Assume one path for simplicity
				Val: &gnmiv1.TypedValue{
					Value: &gnmiv1.TypedValue_JsonVal{
						JsonVal: []byte(`{"openconfig-system:cpu": {"utilization": 42.5}}`),
					},
				},
			},
		},
	}

	resp := &gnmiv1.GetResponse{
		Notification: []*gnmiv1.Notification{notification},
	}
	return connect.NewResponse(resp), nil
}

func (s *gnmiServer) Subscribe(
	ctx context.Context,
	stream *connect.BidiStream[gnmiv1.SubscribeRequest, gnmiv1.SubscribeResponse],
) error {
	log.Println("Client connected for subscription")
	// A real server would manage multiple subscription requests from the stream.
	// For simplicity, we'll read one request and start a ticker.
	req, err := stream.Receive()
	if err != nil {
		return fmt.Errorf("could not receive first request: %w", err)
	}

	// Assuming a STREAM subscription
	path := req.GetSubscribe().GetSubscription()[0].GetPath()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Client disconnected.")
			return ctx.Err()
		case <-ticker.C:
			update := &gnmiv1.Update{
				Path: path,
				Val: &gnmiv1.TypedValue{
					Value: &gnmiv1.TypedValue_JsonVal{
						JsonVal: []byte(fmt.Sprintf(`{"used": %d}`, time.Now().Second()*100)),
					},
				},
			}
			notification := &gnmiv1.Notification{
				Timestamp: time.Now().UnixNano(),
				Update:    []*gnmiv1.Update{update},
			}
			resp := &gnmiv1.SubscribeResponse{
				Response: &gnmiv1.SubscribeResponse_Update{Update: notification},
			}
			if err := stream.Send(resp); err != nil {
				log.Printf("Failed to send update: %v", err)
				return err
			}
		}
	}
}

func (s *gnmiServer) Set(_ context.Context, _ *connect.Request[gnmiv1.SetRequest]) (*connect.Response[gnmiv1.SetResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("gNMI.Set is not implemented"))
}

func main() {
	server := &gnmiServer{}
	mux := http.NewServeMux()
	path, handler := gnmiv1connect.NewGNMIHandler(server)
	mux.Handle(path, handler)
	reflector := grpcreflect.NewStaticReflector("gnmi.gNMI")
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	log.Println("Starting gNMI server on :8080...")
	// Use h2c to support gRPC clients that don't use TLS
	err := http.ListenAndServe(
		"localhost:8080",
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		log.Fatalf("listen and serve failed: %v", err)
	}
}
