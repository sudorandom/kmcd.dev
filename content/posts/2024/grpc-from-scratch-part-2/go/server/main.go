package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/proto"

	greetv1 "github.com/sudorandom/kmcd.devcratch-part-2/gen"
)

var (
	gRPCStatusHeader  = "Grpc-Status"
	gRPCMessageHeader = "Grpc-Message"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/greet.v1.GreetService/Greet", http.HandlerFunc(greetHandler))
	log.Fatal(http.ListenAndServe(
		"localhost:9000",
		h2c.NewHandler(mux, &http2.Server{}),
	))
}

func greetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Trailer", gRPCStatusHeader+", "+gRPCMessageHeader)
	w.Header().Set("Content-Type", "application/grpc+proto")
	w.WriteHeader(http.StatusOK)
	defer r.Body.Close()

	// Read Request
	req := &greetv1.GreetRequest{}
	if err := readMessage(r.Body, req); err != nil {
		writeError(w, err)
		return
	}

	// Write Response
	if err := writeMessage(w, &greetv1.GreetResponse{
		Greeting: fmt.Sprintf("Hello, %s!", req.Name),
	}); err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set(gRPCStatusHeader, "0")
	w.Header().Set(gRPCMessageHeader, "")
}

func writeError(w http.ResponseWriter, err error) {
	log.Printf("read err: %s", err)
	w.Header().Set(gRPCStatusHeader, "1")
	w.Header().Set(gRPCMessageHeader, err.Error())
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
