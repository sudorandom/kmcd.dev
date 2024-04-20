package go_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sudorandom/sudorandom.dev/grpc-from-scratch/gen"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestDecodeString(t *testing.T) {
	b, err := proto.Marshal(&gen.TestMessage{
		StringValue: "Hello World!",
	})
	if err != nil {
		require.NoError(t, err)
	}

	tagNumber, protoType, n := protowire.ConsumeTag(b)
	require.GreaterOrEqual(t, n, 0)
	require.Equal(t, n, 1)
	require.Equal(t, protowire.BytesType, protoType)
	require.True(t, tagNumber.IsValid())
	assert.Equal(t, protowire.Number(7), tagNumber)

	b = b[n:]

	str, n := protowire.ConsumeString(b)
	require.GreaterOrEqual(t, n, 0)
	assert.Equal(t, 13, n)
	assert.Equal(t, "Hello World!", str)
}

func TestEncodeString(t *testing.T) {
	var buf []byte
	buf = protowire.AppendTag(buf, protowire.Number(7), protowire.BytesType)
	buf = protowire.AppendString(buf, "Hello World!")

	res := gen.TestMessage{}
	require.NoError(t, proto.Unmarshal(buf, &res))
	assert.Equal(t, "Hello World!", res.StringValue)
}

func TestDecodeInt32(t *testing.T) {
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

func TestEncodeInt32(t *testing.T) {
	var buf []byte
	buf = protowire.AppendTag(buf, protowire.Number(1), protowire.VarintType)
	buf = protowire.AppendVarint(buf, 1234)

	res := gen.TestMessage{}
	require.NoError(t, proto.Unmarshal(buf, &res))
	assert.Equal(t, int32(1234), res.IntValue)
}

func TestDecodeInt32Array(t *testing.T) {
	b, err := proto.Marshal(&gen.TestMessage{
		RepeatedIntValue: []int32{100002130, 2, 3, 4, 5},
	})
	if err != nil {
		require.NoError(t, err)
	}

	tagNumber, protoType, n := protowire.ConsumeTag(b)
	require.GreaterOrEqual(t, n, 0)
	require.Equal(t, n, 1)
	require.Equal(t, protowire.BytesType, protoType)
	require.True(t, tagNumber.IsValid())
	assert.Equal(t, protowire.Number(9), tagNumber)

	b = b[n:]
	int32buf, n := protowire.ConsumeBytes(b)
	require.GreaterOrEqual(t, n, 0)
	assert.Equal(t, 9, n)
	res := []int32{}
	for len(int32buf) > 0 {
		v, n := protowire.ConsumeVarint(int32buf)
		require.GreaterOrEqual(t, n, 0)
		res = append(res, int32(v))
		int32buf = int32buf[n:]
	}
	assert.Equal(t, []int32{100002130, 2, 3, 4, 5}, res)
}

func TestEncodeInt32Array(t *testing.T) {
	arr := []int32{100002130, 2, 3, 4, 5}
	var buf []byte
	buf = protowire.AppendVarint(buf, protowire.EncodeTag(protowire.Number(9), protowire.BytesType))
	size := 0
	for i := 0; i < len(arr); i++ {
		size += protowire.SizeVarint(uint64(arr[i]))
	}
	buf = protowire.AppendVarint(buf, uint64(size))
	for i := 0; i < len(arr); i++ {
		buf = protowire.AppendVarint(buf, uint64(arr[i]))
	}

	t.Log(buf)

	res := gen.TestMessage{}
	require.NoError(t, proto.Unmarshal(buf, &res))
	assert.Equal(t, []int32{100002130, 2, 3, 4, 5}, res.RepeatedIntValue)
}

func TestAppendVarint(t *testing.T) {
	buf := []byte{}
	buf = protowire.AppendVarint(buf, 123456)
	assert.Equal(t, []byte{0xc0, 0xc4, 0x7}, buf)
}
