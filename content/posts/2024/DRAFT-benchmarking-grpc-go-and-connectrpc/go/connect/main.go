package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"connectrpc.com/connect"
	grpcbench "github.com/sybogames/grpc-bench"
	"github.com/sybogames/grpc-bench/proto"
	"github.com/sybogames/grpc-bench/proto/protoconnect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

type GreetServer struct{}

// SayHello implements protoconnect.GreeterHandler.
func (*GreetServer) SayHello(ctx context.Context, req *connect.Request[proto.HelloRequest]) (*connect.Response[proto.HelloReply], error) {
	res := connect.NewResponse(&proto.HelloReply{
		Message: fmt.Sprintf("Hello, %s!", req.Msg.Name),
	})
	return res, nil
}

func main() {
	flag.Parse()

	greeter := &GreetServer{}
	mux := http.NewServeMux()
	path, handler := protoconnect.NewGreeterHandler(greeter)
	mux.Handle(path, handler)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("server listening at %v", lis.Addr())

	cleanupFn := grpcbench.SetupProfiles()
	defer cleanupFn()

	go func() {
		if err := http.Serve(lis, h2c.NewHandler(mux, &http2.Server{})); err != nil {
			log.Fatalf("serv err: %v", err)
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
}
