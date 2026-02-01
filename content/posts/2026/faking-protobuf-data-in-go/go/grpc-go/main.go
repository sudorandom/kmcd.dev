package main

import (
	"context"
	"log"
	"net"

	"github.com/sudorandom/fauxrpc"
	_ "github.com/sudorandom/fauxrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"buf.build/gen/go/connectrpc/eliza/grpc/go/connectrpc/eliza/v1/elizav1grpc"
	elizav1 "buf.build/gen/go/connectrpc/eliza/protocolbuffers/go/connectrpc/eliza/v1"
)

var _ elizav1grpc.ElizaServiceServer = (*server)(nil)

type server struct {
	elizav1grpc.UnimplementedElizaServiceServer
}

// Say implements elizav1grpc.ElizaServiceHandler.
func (s *server) Say(ctx context.Context, req *elizav1.SayRequest) (*elizav1.SayResponse, error) {
	msg := &elizav1.SayResponse{}
	fauxrpc.SetDataOnMessage(msg, fauxrpc.GenOptions{})
	return msg, nil
}

func main() {
	srv := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	elizav1grpc.RegisterElizaServiceServer(srv, &server{})

	addr := "127.0.0.1:6660"
	log.Printf("Starting connectrpc on %s", addr)

	lis, err := net.Listen("tcp", "127.0.0.1:6660")
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("error: %s", err)
	}
}
