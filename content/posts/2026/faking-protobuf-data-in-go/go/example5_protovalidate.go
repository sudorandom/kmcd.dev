package main

import (
	"fmt"

	"github.com/sudorandom/fauxrpc"
	gen "github.com/sudorandom/kmcd.dev/faking-protobuf-data-in-go/gen"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	user := &gen.User{}

	err := fauxrpc.SetDataOnMessage(user, fauxrpc.GenOptions{})
	if err != nil {
		panic(err)
	}

	out, err := protojson.MarshalOptions{Indent: "  "}.Marshal(user)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))
}
