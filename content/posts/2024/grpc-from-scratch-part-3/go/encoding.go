package grpcfromscratchpart3

import (
	"bytes"
	"errors"
	"io"
	"math"
	"math/bits"
)

const (
	MaxVarintLen64 = 10
)

var (
	ErrOverflow  = errors.New("overflow")
	ErrTruncated = errors.New("truncated")
)

func WriteUvarint(buf *bytes.Buffer, x uint64) {
	for x >= 0x80 {
		buf.WriteByte(byte(x) | 0x80)
		x >>= 7
	}
	buf.WriteByte(byte(x))
}

func WriteFieldTag(buf *bytes.Buffer, field int32, protoType uint8) {
	WriteUvarint(buf, uint64(field<<3)|uint64(protoType))
}

func ReadFieldTag(buf *bytes.Buffer) (int32, int8, error) {
	field, err := ReadUvarint(buf)
	if err != nil {
		return 0, 0, err
	}
	if field>>3 > uint64(math.MaxInt32) {
		return 0, 0, ErrOverflow
	}
	return int32(field >> 3), int8(field & 7), nil
}

func ReadUvarint(buf *bytes.Buffer) (uint64, error) {
	var x uint64
	var s uint
	var i int
	for {
		b, err := buf.ReadByte()
		if err != nil {
			return 0, err
		}
		if i == MaxVarintLen64 {
			return 0, ErrOverflow // overflow
		}
		if b < 0x80 {
			if i == MaxVarintLen64-1 && b > 1 {
				return 0, ErrOverflow // overflow
			}
			return x | uint64(b)<<s, nil
		}
		x |= uint64(b&0x7f) << s
		s += 7
		i++
	}
}

func ReadBytes(buf *bytes.Buffer) ([]byte, error) {
	size, err := ReadUvarint(buf)
	if err != nil {
		return nil, ErrTruncated
	}
	if uint64(buf.Len()) < size {
		return nil, ErrTruncated
	}

	result := make([]byte, size)
	n, err := buf.Read(result)
	if err != nil {
		return nil, err
	}
	if uint64(n) != size {
		return nil, ErrTruncated
	}
	return result, nil
}

func ReadString(buf *bytes.Buffer) (string, error) {
	b, err := ReadBytes(buf)
	if err != nil {
		return "", err
	}
	return string(b), err
}

func ReadRepeatedInt32(buf *bytes.Buffer) ([]int32, error) {
	result := []int32{}
	for {
		res, err := ReadUvarint(buf)
		if err == io.EOF {
			return result, nil
		} else if err != nil {
			return nil, err
		}
		result = append(result, int32(res))
	}
}

func SizeVarint(v uint64) int {
	return int(9*uint32(bits.Len64(v))+64) / 64
}
