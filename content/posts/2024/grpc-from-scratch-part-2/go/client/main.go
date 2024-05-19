package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"

	"connectrpc.com/connect"
	greetv1 "github.com/sudorandom/kmcd.dev/grpc-from-scratch-part-2/gen"
	"github.com/sudorandom/kmcd.dev/grpc-from-scratch-part-2/gen/greetv1connect"

	"golang.org/x/net/http2"
)

func main() {
	httpClient := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}
	client := greetv1connect.NewGreetServiceClient(httpClient, "http://127.0.0.1:9000", connect.WithGRPC())
	req := &greetv1.GreetRequest{Name: "World"}
	fmt.Printf("send-> %v\n", req)
	resp, err := client.Greet(context.Background(), connect.NewRequest(req))
	if err != nil {
		log.Fatalf("err: %s", err)
	}

	fmt.Printf("recv<- %v\n", resp.Msg)
}
