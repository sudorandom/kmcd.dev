package go_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sudorandom/sudorandom.dev/grpc-from-scratch/gen"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestDecodeInt32Raw(t *testing.T) {
	b, err := proto.Marshal(&gen.TestMessage{
		IntValue: 1234,
	})
	if err != nil {
		require.NoError(t, err)
	}

	tagNumber, protoType, n := protowire.ConsumeTag(b)
	require.GreaterOrEqual(t, n, 0)
	require.Equal(t, n, 1)
	require.Equal(t, protowire.VarintType, protoType)
	require.True(t, tagNumber.IsValid())
	assert.Equal(t, protowire.Number(1), tagNumber)

	b = b[n:]
	i, n := protowire.ConsumeVarint(b)
	require.GreaterOrEqual(t, n, 0)
	assert.Equal(t, 2, n)
	assert.Equal(t, int32(1234), int32(i))
}

func TestEncodeRaw(t *testing.T) {
	var buf []byte
	// 10 is the field number; 0 is the type
	buf = append(buf, byte(1<<3|uint64(0)))
	val := uint32(1234)
	buf = append(buf, byte((val>>0)&0x7f|0x80), byte(val>>7))

	res := gen.TestMessage{}
	require.NoError(t, proto.Unmarshal(buf, &res))
	assert.Equal(t, int32(1234), res.IntValue)
}
