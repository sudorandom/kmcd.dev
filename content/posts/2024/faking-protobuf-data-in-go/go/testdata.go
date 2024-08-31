package main

import (
	"fmt"
	"log"

	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	elizav1 "buf.build/gen/go/connectrpc/eliza/protocolbuffers/go/connectrpc/eliza/v1"
	"github.com/sudorandom/fauxrpc"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	{
		msg := &elizav1.SayResponse{}
		fauxrpc.SetDataOnMessage(msg, fauxrpc.GenOptions{})
		b, err := protojson.MarshalOptions{Indent: "  "}.Marshal(msg)
		if err != nil {
			log.Fatalf("err: %s", err)
		}
		fmt.Println(string(b))
	}
	{
		msg := &ownerv1.Owner{}
		fauxrpc.SetDataOnMessage(msg, fauxrpc.GenOptions{})
		b, err := protojson.MarshalOptions{Indent: "  "}.Marshal(msg)
		if err != nil {
			log.Fatalf("err: %s", err)
		}
		fmt.Println(string(b))
	}
	{
		msg := fauxrpc.NewMessage(elizav1.File_connectrpc_eliza_v1_eliza_proto.Messages().ByName("SayResponse"), fauxrpc.GenOptions{})
		b, err := protojson.MarshalOptions{Indent: "  "}.Marshal(msg)
		if err != nil {
			log.Fatalf("err: %s", err)
		}
		fmt.Println(string(b))
	}
	{
		msg := fauxrpc.NewMessage(ownerv1.File_buf_registry_owner_v1_owner_proto.Messages().ByName("Owner"), fauxrpc.GenOptions{})
		b, err := protojson.MarshalOptions{Indent: "  "}.Marshal(msg)
		if err != nil {
			log.Fatalf("err: %s", err)
		}
		fmt.Println(string(b))
	}
}
