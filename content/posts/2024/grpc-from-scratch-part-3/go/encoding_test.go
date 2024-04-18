package go_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	greetv1 "github.com/sudorandom/sudorandom.dev/grpc-from-scratch/gen"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestUnmarshal(t *testing.T) {
	b, err := proto.Marshal(&greetv1.GreetRequest{
		Name: "Hello World!",
	})
	if err != nil {
		require.NoError(t, err)
	}

	tagNumber, protoType, n := protowire.ConsumeTag(b)
	require.GreaterOrEqual(t, n, 0)
	require.Equal(t, n, 1)
	require.Equal(t, protowire.BytesType, protoType)
	require.True(t, tagNumber.IsValid())
	assert.Equal(t, protowire.Number(1), tagNumber)

	b = b[n:]

	str, n := protowire.ConsumeString(b)
	require.GreaterOrEqual(t, n, 0)
	assert.Equal(t, 13, n)
	assert.Equal(t, "Hello World!", str)
}

func TestMarshal(t *testing.T) {
	var buf []byte
	buf = protowire.AppendTag(buf, protowire.Number(1), protowire.BytesType)
	buf = protowire.AppendString(buf, "Hello World!")

	res := greetv1.GreetRequest{}
	require.NoError(t, proto.Unmarshal(buf, &res))
	assert.Equal(t, "Hello World!", res.Name)
}
