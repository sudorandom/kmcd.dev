package main

import (
	"context"
	"github.com/sudorandom/fauxrpc"
	elizav1 "buf.build/gen/go/connectrpc/eliza/protocolbuffers/go/connectrpc/eliza/v1"
)

func Say(ctx context.Context, req *elizav1.SayRequest) (*elizav1.SayResponse, error) {
	msg := &elizav1.SayResponse{}
	if err := fauxrpc.SetDataOnMessage(msg, fauxrpc.GenOptions{}); err != nil {
		return nil, err
	}
	return msg, nil
}
