package main

import (
	"encoding/binary"
	"fmt"
	"io"
)

// FrameHeader represents the 9-byte fixed header of every HTTP/2 frame.
type FrameHeader struct {
	Length   uint32
	Type     uint8
	Flags    uint8
	StreamID uint32
}

// Frame represents a complete HTTP/2 frame including its payload.
type Frame struct {
	Header  FrameHeader
	Payload []byte
}

// ReadFrame reads a header and then the corresponding payload from the connection.
func ReadFrame(r io.Reader) (Frame, error) {
	// Read the 9-byte header
	headerBuf := make([]byte, 9)
	_, err := io.ReadFull(r, headerBuf)
	if err != nil {
		return Frame{}, fmt.Errorf("reading header: %w", err)
	}

	// Parse the header fields using bit-shifting
	header := FrameHeader{
		Length:   uint32(headerBuf[0])<<16 | uint32(headerBuf[1])<<8 | uint32(headerBuf[2]),
		Type:     headerBuf[3],
		Flags:    headerBuf[4],
		StreamID: binary.BigEndian.Uint32(headerBuf[5:9]) & 0x7FFFFFFF,
	}

	// Read the payload based on the Length field
	payload := make([]byte, header.Length)
	if header.Length > 0 {
		_, err = io.ReadFull(r, payload)
		if err != nil {
			return Frame{}, fmt.Errorf("reading payload: %w", err)
		}
	}

	return Frame{Header: header, Payload: payload}, nil
}
