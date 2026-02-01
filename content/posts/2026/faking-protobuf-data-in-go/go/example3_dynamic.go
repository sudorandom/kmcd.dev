package main

import (
	"fmt"
	"log"

	elizav1 "buf.build/gen/go/connectrpc/eliza/protocolbuffers/go/connectrpc/eliza/v1"
	"github.com/sudorandom/fauxrpc"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
    msg, err := fauxrpc.NewMessage(
        elizav1.File_connectrpc_eliza_v1_eliza_proto.Messages().ByName("SayResponse"),
        fauxrpc.GenOptions{},
    )
    if err != nil {
        log.Fatalf("err: %s", err)
    }
    b, err := protojson.MarshalOptions{Indent: "  "}.Marshal(msg)
    if err != nil {
        log.Fatalf("err: %s", err)
    }
    fmt.Println(string(b))
}
