package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const maxFramePayload = 16 << 20 // 16 MiB guard against 62-bit varint OOM

// HTTP/3 Frame Types (RFC 9114 Section 7.2)
// Notice how these match their HTTP/2 counterparts!
const (
	FrameData     uint64 = 0x00
	FrameHeaders  uint64 = 0x01
	FrameSettings uint64 = 0x04
	FrameGoAway   uint64 = 0x07
)

// Frame represents an HTTP/3 frame.
// Unlike HTTP/2's 9-byte header, there is no StreamID or Flags field here:
// QUIC streams already give us stream IDs and clean FIN termination!
type Frame struct {
	Type    uint64
	Payload []byte
}

// ReadVarint reads a QUIC variable-length integer (RFC 9000 Section 16).
// The first two bits (00, 01, 10, 11) tell us whether the integer is 1, 2, 4, or 8 bytes long.
func ReadVarint(r io.Reader) (uint64, error) {
	var first [1]byte
	if _, err := io.ReadFull(r, first[:]); err != nil {
		return 0, err
	}

	// Shift the top 2 bits right by 6 to get 0, 1, 2, or 3.
	// 1 << prefix gives the byte length: 1, 2, 4, or 8.
	prefix := first[0] >> 6
	length := 1 << prefix

	// Mask out the 2-bit length prefix from the first byte.
	val := uint64(first[0] & 0x3f)
	if length == 1 {
		return val, nil
	}

	rest := make([]byte, length-1)
	if _, err := io.ReadFull(r, rest); err != nil {
		return 0, fmt.Errorf("reading varint continuation: %w", err)
	}
	for _, b := range rest {
		val = (val << 8) | uint64(b)
	}
	return val, nil
}

// AppendVarint appends a QUIC variable-length integer (RFC 9000 Section 16) to buf.
func AppendVarint(buf []byte, v uint64) []byte {
	switch {
	case v <= 63: // Fits in 6 bits (1 byte, prefix 00)
		return append(buf, byte(v))
	case v <= 16383: // Fits in 14 bits (2 bytes, prefix 01)
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], uint16(v)|0x4000)
		return append(buf, b[:]...)
	case v <= 1073741823: // Fits in 30 bits (4 bytes, prefix 10)
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(v)|0x80000000)
		return append(buf, b[:]...)
	default: // Up to 62 bits (8 bytes, prefix 11)
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], v|0xc000000000000000)
		return append(buf, b[:]...)
	}
}

// ReadFrame reads a single HTTP/3 frame from r.
func ReadFrame(r io.Reader) (Frame, error) {
	fType, err := ReadVarint(r)
	if err != nil {
		return Frame{}, err
	}
	length, err := ReadVarint(r)
	if err != nil {
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return Frame{}, fmt.Errorf("reading frame length: %w", err)
	}
	if length > maxFramePayload {
		return Frame{}, fmt.Errorf("frame payload length %d exceeds max %d", length, maxFramePayload)
	}
	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return Frame{}, fmt.Errorf("reading frame payload: %w", err)
		}
	}
	return Frame{Type: fType, Payload: payload}, nil
}

// WriteFrame encodes and writes an HTTP/3 frame to w.
func WriteFrame(w io.Writer, fType uint64, payload []byte) error {
	var buf []byte
	buf = AppendVarint(buf, fType)
	buf = AppendVarint(buf, uint64(len(payload)))
	buf = append(buf, payload...)
	_, err := w.Write(buf)
	return err
}
