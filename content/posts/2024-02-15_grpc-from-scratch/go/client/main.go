package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	greetv1 "github.com/sudorandom/sudorandom.dev/content/posts/2024-02-15_grpc-from-scratch/gen"

	"golang.org/x/net/http2"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
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

	req := &greetv1.GreetRequest{Name: "World"}
	resp := &greetv1.GreetResponse{}
	if err := RPC(context.Background(), httpClient, "greet.v1.GreetService/Greet", req, resp); err != nil {
		log.Fatalf("err: %s", err)
	}
}

func RPC(ctx context.Context, httpClient *http.Client, procedure string, protoMsg proto.Message, protoResp proto.Message) error {
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()

	eg, ctx := errgroup.WithContext(ctx)
	req, err := http.NewRequestWithContext(ctx, "POST", "http://127.0.0.1:9000/"+procedure, r)
	if err != nil {
		return fmt.Errorf("http request failure: %w", err)
	}
	req.Header.Add("Content-Type", "application/grpc+proto")

	eg.Go(func() error {
		if err := writeMessage(w, protoMsg); err != nil {
			return err
		}

		if err := w.Close(); err != nil {
			return err
		}

		return nil
	})

	eg.Go(func() error {
		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("http failure: %w", err)
		}
		defer resp.Body.Close()

		if err := readMessage(resp.Body, protoResp); err != nil {
			if err == io.EOF {
				fmt.Println("HTTP Trailers", resp.Trailer)
				return nil
			}
			return err
		}
		return nil
	})

	return eg.Wait()
}

func writeMessage(w io.Writer, protoMsg proto.Message) error {
	fmt.Println("send->", protoMsg)
	msg, err := proto.Marshal(protoMsg)
	if err != nil {
		return err
	}

	prefix := make([]byte, 5)
	binary.BigEndian.PutUint32(prefix[1:], uint32(len(msg)))
	if _, err := w.Write(prefix); err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	return nil
}

func readMessage(body io.Reader, protoResp proto.Message) error {
	prefixes := [5]byte{}
	if _, err := io.ReadFull(body, prefixes[:]); err != nil {
		if err == io.EOF {
			return err
		}
		return fmt.Errorf("failed to read envelope: %w", err)
	}

	buffer := &bytes.Buffer{}
	msgSize := int64(binary.BigEndian.Uint32(prefixes[1:5]))
	if _, err := io.CopyN(buffer, body, msgSize); err != nil {
		return fmt.Errorf("failed to read msg: %w", err)
	}

	if err := proto.Unmarshal(buffer.Bytes(), protoResp); err != nil {
		return fmt.Errorf("failed to unmarshal resp: %w", err)
	}

	fmt.Println("recv<-", protoResp)
	return nil
}
