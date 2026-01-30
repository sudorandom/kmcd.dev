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

	"github.com/sudorandom/kmcd.dev/gnmi/gen/gnmi"
	"github.com/sudorandom/kmcd.dev/gnmi/gen/gnmi/gnmiconnect"
)

var _ gnmiconnect.GNMIHandler = (*gnmiServer)(nil)

var targetCPUUsagePath = &gnmi.Path{
	Elem: []*gnmi.PathElem{
		{Name: "system"},
		{Name: "cpu"},
		{Name: "state"},
		{Name: "used"},
	},
}

func pathsMatch(subPath, targetPath *gnmi.Path) bool {
	if subPath == nil || targetPath == nil {
		return false
	}

	subElems := subPath.GetElem()
	targetElems := targetPath.GetElem()

	if len(subElems) > len(targetElems) {
		return false
	}

	for i := range subElems {
		if subElems[i].GetName() != targetElems[i].GetName() {
			return false
		}
	}
	return true
}

type gnmiServer struct {
	// In a real server, you'd have a data store, cache, etc.
}

func (s *gnmiServer) Capabilities(
	ctx context.Context,
	req *connect.Request[gnmi.CapabilityRequest],
) (*connect.Response[gnmi.CapabilityResponse], error) {
	// A real server would list its supported models and encodings.
	resp := &gnmi.CapabilityResponse{
		SupportedModels:    []*gnmi.ModelData{},
		SupportedEncodings: []gnmi.Encoding{gnmi.Encoding_JSON},
		GNMIVersion:        "0.7.0",
	}
	return connect.NewResponse(resp), nil
}

func (s *gnmiServer) Get(
	ctx context.Context,
	req *connect.Request[gnmi.GetRequest],
) (*connect.Response[gnmi.GetResponse], error) {
	// This is a simplified Get. A real one would parse paths and fetch data.
	// We'll just return a mock CPU value if asked.
	cpuUsage := time.Now().Second() % 100 // Simulate CPU usage
	notification := &gnmi.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*gnmi.Update{
			{
				Path: targetCPUUsagePath, // Assume one path for simplicity
				Val: &gnmi.TypedValue{
					Value: &gnmi.TypedValue_IntVal{
						IntVal: int64(cpuUsage),
					},
				},
			},
		},
	}

	resp := &gnmi.GetResponse{
		Notification: []*gnmi.Notification{notification},
	}
	return connect.NewResponse(resp), nil
}

func (s *gnmiServer) Subscribe(
	ctx context.Context,
	stream *connect.BidiStream[gnmi.SubscribeRequest, gnmi.SubscribeResponse],
) error {
	log.Println("Client connected for subscription")
	// A real server would manage multiple subscription requests from the stream.
	// For simplicity, we'll read one request and start a ticker.
	req, err := stream.Receive()
	if err != nil {
		return fmt.Errorf("could not receive first request: %w", err)
	}

	// Assuming a STREAM subscription
	subPath := req.GetSubscribe().GetSubscription()[0].GetPath()

	// Determine if the subscribed path is a prefix of the target CPU path.
	shouldSendCPUUpdates := pathsMatch(subPath, targetCPUUsagePath)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Client disconnected.")
			return ctx.Err()
		case <-ticker.C:
			if shouldSendCPUUpdates {
				cpuUsage := time.Now().Second() % 100 // Simulate CPU usage
				update := &gnmi.Update{
					Path: targetCPUUsagePath, // Return the full path of the metric
					Val: &gnmi.TypedValue{
						Value: &gnmi.TypedValue_IntVal{
							IntVal: int64(cpuUsage),
						},
					},
				}
				notification := &gnmi.Notification{
					Timestamp: time.Now().UnixNano(),
					Update:    []*gnmi.Update{update},
				}
				resp := &gnmi.SubscribeResponse{
					Response: &gnmi.SubscribeResponse_Update{Update: notification},
				}
				if err := stream.Send(resp); err != nil {
					log.Printf("Failed to send update: %v", err)
					return err
				}
			} else {
				log.Printf("Subscribed path %v does not match target CPU path, not sending updates.", subPath)
			}
		}
	}
}

func (s *gnmiServer) Set(_ context.Context, _ *connect.Request[gnmi.SetRequest]) (*connect.Response[gnmi.SetResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("gNMI.Set is not implemented"))
}

func main() {
	server := &gnmiServer{}
	mux := http.NewServeMux()
	path, handler := gnmiconnect.NewGNMIHandler(server)
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
