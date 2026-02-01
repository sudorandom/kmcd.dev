package main

import (
	"fmt"
	"log"

	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	"github.com/sudorandom/fauxrpc"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
		msg := &ownerv1.Owner{}
		if err := fauxrpc.SetDataOnMessage(msg, fauxrpc.GenOptions{}); err != nil {
			log.Fatalf("err: %s", err)
		}
		b, err := protojson.MarshalOptions{Indent: "  "}.Marshal(msg)
		if err != nil {
			log.Fatalf("err: %s", err)
		}
		fmt.Println(string(b))
}
