package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// QUIC Packet & Frame constants (RFC 9000 Sections 17 & 19)
const (
	QUICVersion1 uint32 = 0x00000001

	// Long Header Packet Types (bits 4-5 when bit 7 = 1)
	PacketTypeInitial   uint8 = 0x00
	PacketType0RTT      uint8 = 0x01
	PacketTypeHandshake uint8 = 0x02

	// Core QUIC Frame Types (RFC 9000 Section 19)
	QUICFramePadding         uint64 = 0x00
	QUICFramePing            uint64 = 0x01
	QUICFrameAck             uint64 = 0x02
	QUICFrameCrypto          uint64 = 0x06
	QUICFrameStreamBase      uint64 = 0x08 // 0x08..0x0f depending on OFF, LEN, FIN bits
	QUICFrameConnectionClose uint64 = 0x1d // Application-layer CONNECTION_CLOSE

	// STREAM Frame low 3 flag bits (RFC 9000 Section 19.8)
	StreamFlagOFF uint64 = 0x04 // Offset field is present
	StreamFlagLEN uint64 = 0x02 // Length field is present
	StreamFlagFIN uint64 = 0x01 // Reached end of stream

	// Stream ID low 2 bits (RFC 9000 Section 2.1)
	StreamIDServerInitiated uint64 = 0x01 // 0 = Client, 1 = Server
	StreamIDUnidirectional  uint64 = 0x02 // 0 = Bidirectional, 1 = Unidirectional
)

// Packet represents a parsed QUIC packet (Long Header or 1-RTT Short Header).
type Packet struct {
	IsLongHeader bool
	LongType     uint8
	Version      uint32
	DestConnID   uint64
	SrcConnID    uint64
	PacketNumber uint32
	Frames       []QUICFrame
}

// QUICFrame represents a single frame carried inside a QUIC packet payload.
type QUICFrame struct {
	Type uint64

	// STREAM frame fields (0x08..0x0f)
	StreamID uint64
	Offset   uint64
	Fin      bool
	Data     []byte

	// ACK frame fields (0x02)
	LargestAcked uint64

	// CRYPTO frame fields (0x06)
	CryptoOffset uint64
	CryptoData   []byte

	// CONNECTION_CLOSE fields (0x1d)
	ErrorCode    uint64
	ReasonPhrase string
}

// EncodePacket serializes a QUIC packet and its constituent frames into a UDP datagram.
func EncodePacket(pkt Packet) []byte {
	var buf []byte
	if pkt.IsLongHeader {
		// Long Header: 1 (Header Form) | 1 (Fixed Bit) | LongType (2 bits) | 0000
		firstByte := byte(0xc0 | ((pkt.LongType & 0x03) << 4))
		buf = append(buf, firstByte)

		var v [4]byte
		binary.BigEndian.PutUint32(v[:], pkt.Version)
		buf = append(buf, v[:]...)

		// 8-byte Destination Connection ID & 8-byte Source Connection ID
		buf = append(buf, 8)
		var dcid [8]byte
		binary.BigEndian.PutUint64(dcid[:], pkt.DestConnID)
		buf = append(buf, dcid[:]...)

		buf = append(buf, 8)
		var scid [8]byte
		binary.BigEndian.PutUint64(scid[:], pkt.SrcConnID)
		buf = append(buf, scid[:]...)
	} else {
		// Short Header (1-RTT): 0 (Header Form) | 1 (Fixed Bit) | 000000
		buf = append(buf, 0x40)
		var dcid [8]byte
		binary.BigEndian.PutUint64(dcid[:], pkt.DestConnID)
		buf = append(buf, dcid[:]...)
	}

	var pn [4]byte
	binary.BigEndian.PutUint32(pn[:], pkt.PacketNumber)
	buf = append(buf, pn[:]...)

	for _, f := range pkt.Frames {
		buf = AppendQUICFrame(buf, f)
	}
	return buf
}

// DecodePacket parses a UDP datagram into a QUIC Packet and its slice of QUICFrames.
func DecodePacket(datagram []byte) (Packet, error) {
	r := bytes.NewReader(datagram)
	first, err := r.ReadByte()
	if err != nil {
		return Packet{}, err
	}

	var pkt Packet
	if first&0x80 != 0 {
		pkt.IsLongHeader = true
		pkt.LongType = (first >> 4) & 0x03
		if err := binary.Read(r, binary.BigEndian, &pkt.Version); err != nil {
			return Packet{}, err
		}
		dcidLen, err := r.ReadByte()
		if err != nil || dcidLen != 8 {
			return Packet{}, fmt.Errorf("expected 8-byte DCID")
		}
		if err := binary.Read(r, binary.BigEndian, &pkt.DestConnID); err != nil {
			return Packet{}, err
		}
		scidLen, err := r.ReadByte()
		if err != nil || scidLen != 8 {
			return Packet{}, fmt.Errorf("expected 8-byte SCID")
		}
		if err := binary.Read(r, binary.BigEndian, &pkt.SrcConnID); err != nil {
			return Packet{}, err
		}
	} else {
		if err := binary.Read(r, binary.BigEndian, &pkt.DestConnID); err != nil {
			return Packet{}, err
		}
	}

	if err := binary.Read(r, binary.BigEndian, &pkt.PacketNumber); err != nil {
		return Packet{}, err
	}

	for r.Len() > 0 {
		f, err := ReadQUICFrame(r)
		if err != nil {
			return Packet{}, err
		}
		if f.Type == QUICFramePadding {
			continue
		}
		pkt.Frames = append(pkt.Frames, f)
	}
	return pkt, nil
}

// AppendQUICFrame appends a single RFC 9000 QUIC frame to buf.
func AppendQUICFrame(buf []byte, f QUICFrame) []byte {
	switch {
	case f.Type == QUICFramePing:
		return AppendVarint(buf, QUICFramePing)

	case f.Type == QUICFrameAck:
		buf = AppendVarint(buf, QUICFrameAck)
		buf = AppendVarint(buf, f.LargestAcked)
		buf = AppendVarint(buf, 0) // ACK Delay
		buf = AppendVarint(buf, 0) // ACK Range Count
		buf = AppendVarint(buf, 0) // First ACK Range
		return buf

	case f.Type == QUICFrameCrypto:
		buf = AppendVarint(buf, QUICFrameCrypto)
		buf = AppendVarint(buf, f.CryptoOffset)
		buf = AppendVarint(buf, uint64(len(f.CryptoData)))
		return append(buf, f.CryptoData...)

	case f.Type >= 0x08 && f.Type <= 0x0f:
		frameType := QUICFrameStreamBase | StreamFlagLEN
		if f.Offset > 0 {
			frameType |= StreamFlagOFF
		}
		if f.Fin {
			frameType |= StreamFlagFIN
		}
		buf = AppendVarint(buf, frameType)
		buf = AppendVarint(buf, f.StreamID)
		if f.Offset > 0 {
			buf = AppendVarint(buf, f.Offset)
		}
		buf = AppendVarint(buf, uint64(len(f.Data)))
		return append(buf, f.Data...)

	case f.Type == QUICFrameConnectionClose:
		buf = AppendVarint(buf, QUICFrameConnectionClose)
		buf = AppendVarint(buf, f.ErrorCode)
		buf = AppendVarint(buf, uint64(len(f.ReasonPhrase)))
		return append(buf, []byte(f.ReasonPhrase)...)

	default:
		return buf
	}
}

// ReadQUICFrame decodes a single RFC 9000 QUIC frame from r.
func ReadQUICFrame(r io.Reader) (QUICFrame, error) {
	fType, err := ReadVarint(r)
	if err != nil {
		return QUICFrame{}, err
	}

	switch {
	case fType == QUICFramePadding || fType == QUICFramePing:
		return QUICFrame{Type: fType}, nil

	case fType == QUICFrameAck:
		largest, err := ReadVarint(r)
		if err != nil {
			return QUICFrame{}, err
		}
		_, _ = ReadVarint(r) // Delay
		_, _ = ReadVarint(r) // Range Count
		_, _ = ReadVarint(r) // First Range
		return QUICFrame{Type: QUICFrameAck, LargestAcked: largest}, nil

	case fType == QUICFrameCrypto:
		offset, err := ReadVarint(r)
		if err != nil {
			return QUICFrame{}, err
		}
		length, err := ReadVarint(r)
		if err != nil {
			return QUICFrame{}, err
		}
		data := make([]byte, length)
		if _, err := io.ReadFull(r, data); err != nil {
			return QUICFrame{}, err
		}
		return QUICFrame{Type: QUICFrameCrypto, CryptoOffset: offset, CryptoData: data}, nil

	case fType >= 0x08 && fType <= 0x0f:
		streamID, err := ReadVarint(r)
		if err != nil {
			return QUICFrame{}, err
		}
		var offset uint64
		if fType&StreamFlagOFF != 0 {
			offset, err = ReadVarint(r)
			if err != nil {
				return QUICFrame{}, err
			}
		}
		var data []byte
		if fType&StreamFlagLEN != 0 {
			length, err := ReadVarint(r)
			if err != nil {
				return QUICFrame{}, err
			}
			data = make([]byte, length)
			if _, err := io.ReadFull(r, data); err != nil {
				return QUICFrame{}, err
			}
		} else {
			data, err = io.ReadAll(r)
			if err != nil {
				return QUICFrame{}, err
			}
		}
		return QUICFrame{
			Type:     QUICFrameStreamBase,
			StreamID: streamID,
			Offset:   offset,
			Fin:      fType&StreamFlagFIN != 0,
			Data:     data,
		}, nil

	case fType == QUICFrameConnectionClose:
		code, err := ReadVarint(r)
		if err != nil {
			return QUICFrame{}, err
		}
		reasonLen, err := ReadVarint(r)
		if err != nil {
			return QUICFrame{}, err
		}
		reason := make([]byte, reasonLen)
		if _, err := io.ReadFull(r, reason); err != nil {
			return QUICFrame{}, err
		}
		return QUICFrame{
			Type:         QUICFrameConnectionClose,
			ErrorCode:    code,
			ReasonPhrase: string(reason),
		}, nil

	default:
		return QUICFrame{}, fmt.Errorf("unknown QUIC frame type: 0x%x", fType)
	}
}
