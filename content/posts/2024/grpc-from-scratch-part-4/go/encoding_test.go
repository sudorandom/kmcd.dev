package grpcfromscratchpart3

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sudorandom/sudorandom.dev/grpc-from-scratch/gen"
	"google.golang.org/protobuf/proto"
)

func TestEncodeInt32(t *testing.T) {
	buf := &bytes.Buffer{}
	WriteFieldTag(buf, 1, 0)
	WriteUvarint(buf, 1234)

	res := gen.TestMessage{}
	require.NoError(t, proto.Unmarshal(buf.Bytes(), &res))
	assert.Equal(t, int32(1234), res.IntValue)
}

func TestDecodeInt32(t *testing.T) {
	b, err := proto.Marshal(&gen.TestMessage{
		IntValue: 1234,
	})
	if err != nil {
		require.NoError(t, err)
	}
	buf := bytes.NewBuffer(b)
	tagNumber, protoType, err := ReadFieldTag(buf)
	require.NoError(t, err)
	assert.Equal(t, int8(0), protoType)
	assert.Equal(t, int32(1), tagNumber)

	i, err := ReadUvarint(buf)
	assert.NoError(t, err)
	assert.Equal(t, int32(1234), int32(i))
}
func TestEncodeString(t *testing.T) {
	buf := &bytes.Buffer{}
	s := "hello world"
	WriteFieldTag(buf, 4, 2)
	WriteUvarint(buf, uint64(len(s)))
	buf.WriteString(s)

	res := gen.TestMessage{}
	require.NoError(t, proto.Unmarshal(buf.Bytes(), &res))
	assert.Equal(t, "hello world", res.StringValue)
}

func TestDecodeString(t *testing.T) {
	b, err := proto.Marshal(&gen.TestMessage{
		StringValue: "hello world",
	})
	if err != nil {
		require.NoError(t, err)
	}

	buf := bytes.NewBuffer(b)
	tagNumber, protoType, err := ReadFieldTag(buf)
	require.NoError(t, err)
	assert.Equal(t, int8(2), protoType)
	assert.Equal(t, int32(4), tagNumber)

	s, err := ReadString(buf)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(s))
}

func TestEncodeInt32Array(t *testing.T) {
	arr := []int32{100002130, 2, 3, 4, 5}
	buf := &bytes.Buffer{}
	WriteFieldTag(buf, 10, 2)
	size := 0
	for i := 0; i < len(arr); i++ {
		size += SizeVarint(uint64(arr[i]))
	}
	WriteUvarint(buf, uint64(size))
	for i := 0; i < len(arr); i++ {
		WriteUvarint(buf, uint64(arr[i]))
	}

	res := gen.TestMessage{}
	require.NoError(t, proto.Unmarshal(buf.Bytes(), &res))
	assert.Equal(t, []int32{100002130, 2, 3, 4, 5}, res.RepeatedIntValue)
}

func TestDecodeInt32Array(t *testing.T) {
	b, err := proto.Marshal(&gen.TestMessage{
		RepeatedIntValue: []int32{1, 2, 3, 400},
	})
	if err != nil {
		require.NoError(t, err)
	}

	buf := bytes.NewBuffer(b)
	tagNumber, protoType, err := ReadFieldTag(buf)
	require.NoError(t, err)
	assert.Equal(t, int8(2), protoType)
	assert.Equal(t, int32(10), tagNumber)

	packedBytes, err := ReadBytes(buf)
	assert.NoError(t, err)

	result, err := ReadRepeatedInt32(bytes.NewBuffer(packedBytes))
	assert.NoError(t, err)
	assert.Equal(t, []int32{1, 2, 3, 400}, result)
}
