package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	QUICVersion1 uint32 = 0x00000001

	PacketTypeInitial   uint8 = 0x00
	PacketType0RTT      uint8 = 0x01
	PacketTypeHandshake uint8 = 0x02

	QUICFramePadding         uint64 = 0x00
	QUICFramePing            uint64 = 0x01
	QUICFrameAck             uint64 = 0x02
	QUICFrameAckECN          uint64 = 0x03
	QUICFrameResetStream     uint64 = 0x04
	QUICFrameStopSending     uint64 = 0x05
	QUICFrameCrypto          uint64 = 0x06
	QUICFrameNewToken        uint64 = 0x07
	QUICFrameStreamBase      uint64 = 0x08
	QUICFrameMaxData         uint64 = 0x10
	QUICFrameMaxStreamData   uint64 = 0x11
	QUICFrameMaxStreamsBidi  uint64 = 0x12
	QUICFrameMaxStreamsUni   uint64 = 0x13
	QUICFrameNewConnectionID uint64 = 0x18
	QUICFrameRetireConnID    uint64 = 0x19
	QUICFramePathChallenge   uint64 = 0x1a
	QUICFramePathResponse    uint64 = 0x1b
	QUICFrameConnCloseQUIC   uint64 = 0x1c
	QUICFrameConnectionClose uint64 = 0x1d
	QUICFrameHandshakeDone   uint64 = 0x1e

	StreamFlagOFF uint64 = 0x04
	StreamFlagLEN uint64 = 0x02
	StreamFlagFIN uint64 = 0x01

	StreamIDServerInitiated uint64 = 0x01
	StreamIDUnidirectional  uint64 = 0x02
)

type Packet struct {
	IsLongHeader bool
	LongType     uint8
	Version      uint32
	DestConnID   []byte
	SrcConnID    []byte
	PacketNumber uint32
	Frames       []QUICFrame
}

type QUICFrame struct {
	Type         uint64
	StreamID     uint64
	Offset       uint64
	Fin          bool
	Data         []byte
	LargestAcked uint64
	CryptoOffset uint64
	CryptoData   []byte
	ErrorCode    uint64
	ReasonPhrase string
}

func EncodeProtectedPacket(pkt Packet, keys *packetKeys, minSize int) ([]byte, error) {
	var payload []byte
	for _, f := range pkt.Frames {
		payload = AppendQUICFrame(payload, f)
	}

	var hdr []byte
	if pkt.IsLongHeader {
		firstByte := byte(0xc0 | ((pkt.LongType & 0x03) << 4) | 0x03)
		hdr = append(hdr, firstByte)

		var v [4]byte
		binary.BigEndian.PutUint32(v[:], pkt.Version)
		hdr = append(hdr, v[:]...)

		hdr = append(hdr, byte(len(pkt.DestConnID)))
		hdr = append(hdr, pkt.DestConnID...)
		hdr = append(hdr, byte(len(pkt.SrcConnID)))
		hdr = append(hdr, pkt.SrcConnID...)

		if pkt.LongType == PacketTypeInitial {
			hdr = AppendVarint(hdr, 0) // Token Length = 0
		}
	} else {
		hdr = append(hdr, 0x43) // Short Header (0x40) | 4-byte Packet Number (0x03)
		hdr = append(hdr, pkt.DestConnID...)
	}

	// Pad payload with PADDING (0x00) frames if needed (e.g. 1200-byte minimum for Initial packets)
	overhead := len(hdr) + 4 + 16
	if pkt.IsLongHeader {
		overhead += 2 // 2-byte Length varint
	}
	if len(payload)+overhead < minSize {
		pad := make([]byte, minSize-(len(payload)+overhead))
		payload = append(payload, pad...)
	}

	if pkt.IsLongHeader {
		// Length field covers the 4-byte Packet Number + encrypted payload + 16-byte AEAD tag
		pktLen := uint64(4 + len(payload) + 16)
		hdr = AppendVarint(hdr, pktLen)
	}

	return keys.protectPacket(hdr, pkt.PacketNumber, payload)
}

func DecodeCoalescedPackets(datagram []byte, localCIDLen int, getKeys func(isLong bool, longType uint8) (*packetKeys, uint64)) ([]Packet, error) {
	var packets []Packet
	rest := datagram

	for len(rest) > 0 {
		if rest[0] == 0 {
			break
		}
		var pkt Packet
		var pnOffset int
		var pktTotalLen int

		if rest[0]&0x80 != 0 {
			pkt.IsLongHeader = true
			pkt.LongType = (rest[0] >> 4) & 0x03
			if len(rest) < 6 {
				break
			}
			pkt.Version = binary.BigEndian.Uint32(rest[1:5])
			pos := 5

			dcidLen := int(rest[pos])
			pos++
			if len(rest) < pos+dcidLen+1 {
				break
			}
			pkt.DestConnID = append([]byte(nil), rest[pos:pos+dcidLen]...)
			pos += dcidLen

			scidLen := int(rest[pos])
			pos++
			if len(rest) < pos+scidLen {
				break
			}
			pkt.SrcConnID = append([]byte(nil), rest[pos:pos+scidLen]...)
			pos += scidLen

			r := bytes.NewReader(rest[pos:])
			if pkt.LongType == PacketTypeInitial {
				tokLen, err := ReadVarint(r)
				if err != nil {
					return packets, err
				}
				if _, err := r.Seek(int64(tokLen), io.SeekCurrent); err != nil {
					return packets, err
				}
			}
			length, err := ReadVarint(r)
			if err != nil {
				return packets, err
			}
			pnOffset = len(rest) - r.Len()
			pktTotalLen = pnOffset + int(length)
			if pktTotalLen > len(rest) {
				pktTotalLen = len(rest)
			}
		} else {
			if len(rest) < 1+localCIDLen {
				break
			}
			pkt.DestConnID = append([]byte(nil), rest[1:1+localCIDLen]...)
			pnOffset = 1 + localCIDLen
			pktTotalLen = len(rest)
		}

		rawPkt := rest[:pktTotalLen]
		rest = rest[pktTotalLen:]

		keys, expectedPN := getKeys(pkt.IsLongHeader, pkt.LongType)
		if keys == nil {
			continue
		}
		pn, plaintext, err := keys.unprotectPacket(rawPkt, pnOffset, expectedPN)
		if err != nil {
			return packets, err
		}
		pkt.PacketNumber = uint32(pn)

		frReader := bytes.NewReader(plaintext)
		for frReader.Len() > 0 {
			f, err := ReadQUICFrame(frReader)
			if err != nil {
				return packets, err
			}
			if f.Type == QUICFramePadding {
				continue
			}
			pkt.Frames = append(pkt.Frames, f)
		}
		packets = append(packets, pkt)
	}
	return packets, nil
}

func AppendQUICFrame(buf []byte, f QUICFrame) []byte {
	switch {
	case f.Type == QUICFramePing:
		return AppendVarint(buf, QUICFramePing)

	case f.Type == QUICFrameAck:
		buf = AppendVarint(buf, QUICFrameAck)
		buf = AppendVarint(buf, f.LargestAcked)
		buf = AppendVarint(buf, 0)
		buf = AppendVarint(buf, 0)
		buf = AppendVarint(buf, 0)
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

	case f.Type == QUICFrameHandshakeDone:
		return AppendVarint(buf, QUICFrameHandshakeDone)

	default:
		return buf
	}
}

func ReadQUICFrame(r io.Reader) (QUICFrame, error) {
	fType, err := ReadVarint(r)
	if err != nil {
		return QUICFrame{}, err
	}

	switch {
	case fType == QUICFramePadding || fType == QUICFramePing || fType == QUICFrameHandshakeDone:
		return QUICFrame{Type: fType}, nil

	case fType == QUICFrameAck || fType == QUICFrameAckECN:
		largest, err := ReadVarint(r)
		if err != nil {
			return QUICFrame{}, err
		}
		_, _ = ReadVarint(r) // Delay
		rangeCount, err := ReadVarint(r)
		if err != nil {
			return QUICFrame{}, err
		}
		_, _ = ReadVarint(r) // First Range
		for i := uint64(0); i < rangeCount; i++ {
			_, _ = ReadVarint(r) // Gap
			_, _ = ReadVarint(r) // Ack Range Length
		}
		if fType == QUICFrameAckECN {
			_, _ = ReadVarint(r)
			_, _ = ReadVarint(r)
			_, _ = ReadVarint(r)
		}
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

	case fType == QUICFrameNewToken:
		length, err := ReadVarint(r)
		if err != nil {
			return QUICFrame{}, err
		}
		if _, err := io.CopyN(io.Discard, r, int64(length)); err != nil {
			return QUICFrame{}, err
		}
		return QUICFrame{Type: fType}, nil

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

	case fType == QUICFrameMaxData || fType == QUICFrameMaxStreamsBidi || fType == QUICFrameMaxStreamsUni || fType == QUICFrameRetireConnID:
		_, err := ReadVarint(r)
		return QUICFrame{Type: fType}, err

	case fType == QUICFrameMaxStreamData || fType == QUICFrameStopSending:
		_, _ = ReadVarint(r)
		_, err := ReadVarint(r)
		return QUICFrame{Type: fType}, err

	case fType == QUICFrameResetStream:
		_, _ = ReadVarint(r)
		_, _ = ReadVarint(r)
		_, err := ReadVarint(r)
		return QUICFrame{Type: fType}, err

	case fType == QUICFrameNewConnectionID:
		_, _ = ReadVarint(r) // Sequence Number
		_, _ = ReadVarint(r) // Retire Prior To
		var cidLen [1]byte
		if _, err := io.ReadFull(r, cidLen[:]); err != nil {
			return QUICFrame{}, err
		}
		if _, err := io.CopyN(io.Discard, r, int64(cidLen[0])+16); err != nil {
			return QUICFrame{}, err
		}
		return QUICFrame{Type: fType}, nil

	case fType == QUICFramePathChallenge || fType == QUICFramePathResponse:
		var b [8]byte
		_, err := io.ReadFull(r, b[:])
		return QUICFrame{Type: fType, Data: b[:]}, err

	case fType == QUICFrameConnCloseQUIC || fType == QUICFrameConnectionClose:
		code, err := ReadVarint(r)
		if err != nil {
			return QUICFrame{}, err
		}
		if fType == QUICFrameConnCloseQUIC {
			_, _ = ReadVarint(r) // Frame Type
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
