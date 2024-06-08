package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	greetv1 "example/gen/greet/v1"
)

// start
func TestGreetTable(t *testing.T) {
	service := &greeterService{}
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	testCases := []struct {
		name    string
		ctx     context.Context
		req     *connect.Request[greetv1.GreetRequest]
		want    *connect.Response[greetv1.GreetResponse]
		wantErr string
	}{
		{
			name:    "Success",
			req:     connect.NewRequest(&greetv1.GreetRequest{Name: "Bob"}),
			want:    connect.NewResponse(&greetv1.GreetResponse{Greeting: "Hello, Bob"}),
			wantErr: "",
		},
		{
			name:    "Empty Name",
			req:     connect.NewRequest(&greetv1.GreetRequest{}),
			want:    nil, // Expecting an error
			wantErr: "missing name",
		},
		{
			name:    "Context Cancelled",
			ctx:     cancelledCtx,
			req:     connect.NewRequest(&greetv1.GreetRequest{Name: "Alice"}),
			want:    nil,
			wantErr: "context canceled",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := service.Greet(ctx, tc.req)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
