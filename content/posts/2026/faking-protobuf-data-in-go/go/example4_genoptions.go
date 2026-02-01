package main

import (
	"fmt"
	"log"

	"github.com/brianvoe/gofakeit/v7"
	elizav1 "buf.build/gen/go/connectrpc/eliza/protocolbuffers/go/connectrpc/eliza/v1"
	"github.com/sudorandom/fauxrpc"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	faker := gofakeit.New(123) // Seed the faker for deterministic output
	msg := &elizav1.SayResponse{}
	if err := fauxrpc.SetDataOnMessage(msg, fauxrpc.GenOptions{Faker: faker}); err != nil {
		log.Fatalf("err: %s", err)
	}
	b, err := protojson.MarshalOptions{Indent: "  "}.Marshal(msg)
	if err != nil {
		log.Fatalf("err: %s", err)
	}
	fmt.Println(string(b))
}
