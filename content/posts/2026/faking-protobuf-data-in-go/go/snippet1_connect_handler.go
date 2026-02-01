package main

import (
	"context"
	"github.com/sudorandom/fauxrpc"
	"github.com/bufbuild/connect-go"
	elizav1 "buf.build/gen/go/connectrpc/eliza/protocolbuffers/go/connectrpc/eliza/v1"
)

func Say(ctx context.Context, req *connect.Request[elizav1.SayRequest]) (*connect.Response[elizav1.SayResponse], error) {
    msg := &elizav1.SayResponse{}
    if err := fauxrpc.SetDataOnMessage(msg, fauxrpc.GenOptions{}); err != nil {
        return nil, err
    }
    return connect.NewResponse(msg), nil
}
