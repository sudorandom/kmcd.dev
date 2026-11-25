package varintbench

import (
	"encoding/binary"
	"math/bits"
	"unsafe"

	"google.golang.org/protobuf/encoding/protowire"
)

// ---------------------------------------------------------------------------
// 1. Protobuf / LEB128 (three implementations: hand-rolled, protowire, stdlib)
// ---------------------------------------------------------------------------

func EncodeLEB128(dst []byte, vals []uint64) []byte {
	for _, v := range vals {
		if v < 0x80 {
			dst = append(dst, byte(v))
			continue
		}
		for v >= 0x80 {
			dst = append(dst, byte(v)|0x80)
			v >>= 7
		}
		dst = append(dst, byte(v))
	}
	return dst
}

func DecodeLEB128(src []byte, dst []uint64) int {
	i, n := 0, 0
	for i < len(src) {
		b := src[i]
		if b < 0x80 {
			dst[n] = uint64(b)
			i++
			n++
			continue
		}
		v := uint64(b & 0x7f)
		i++
		var shift uint = 7
		for {
			b = src[i]
			i++
			v |= uint64(b&0x7f) << shift
			if b < 0x80 {
				break
			}
			shift += 7
		}
		dst[n] = v
		n++
	}
	return n
}

func EncodeLEB128Protowire(dst []byte, vals []uint64) []byte {
	for _, v := range vals {
		dst = protowire.AppendVarint(dst, v)
	}
	return dst
}

func DecodeLEB128Protowire(src []byte, dst []uint64) int {
	i, n := 0, 0
	for i < len(src) {
		v, read := protowire.ConsumeVarint(src[i:])
		if read <= 0 {
			break
		}
		dst[n] = v
		n++
		i += read
	}
	return n
}

func EncodeLEB128Stdlib(dst []byte, vals []uint64) []byte {
	for _, v := range vals {
		dst = binary.AppendUvarint(dst, v)
	}
	return dst
}

func DecodeLEB128Stdlib(src []byte, dst []uint64) int {
	i, n := 0, 0
	for i < len(src) {
		v, read := binary.Uvarint(src[i:])
		if read <= 0 {
			break
		}
		dst[n] = v
		n++
		i += read
	}
	return n
}

// ---------------------------------------------------------------------------
// 2. QUIC / HTTP/3 (RFC 9000 Section 16: 2-bit MSB length, big-endian, 1/2/4/8B)
// Maximum representable value is 2^62 - 1.
// ---------------------------------------------------------------------------

func EncodeQUIC(dst []byte, vals []uint64) []byte {
	for _, v := range vals {
		switch {
		case v <= 63:
			dst = append(dst, byte(v))
		case v <= 16383:
			dst = binary.BigEndian.AppendUint16(dst, uint16(v)|0x4000)
		case v <= 1073741823:
			dst = binary.BigEndian.AppendUint32(dst, uint32(v)|0x80000000)
		default:
			dst = binary.BigEndian.AppendUint64(dst, v|0xc000000000000000)
		}
	}
	return dst
}

func DecodeQUIC(src []byte, dst []uint64) int {
	i, n := 0, 0
	for i < len(src) {
		b0 := src[i]
		switch b0 >> 6 {
		case 0:
			dst[n] = uint64(b0)
			i++
		case 1:
			dst[n] = uint64(binary.BigEndian.Uint16(src[i:]) & 0x3fff)
			i += 2
		case 2:
			dst[n] = uint64(binary.BigEndian.Uint32(src[i:]) & 0x3fffffff)
			i += 4
		case 3:
			dst[n] = binary.BigEndian.Uint64(src[i:]) & 0x3fffffffffffffff
			i += 8
		}
		n++
	}
	return n
}

// ---------------------------------------------------------------------------
// 3. OmniBuf Prefix-Varint (4-tier trailing-zero LSB prefix, LE, 1/2/4/9B)
// ---------------------------------------------------------------------------

func EncodeOmniBuf(dst []byte, vals []uint64) []byte {
	for _, v := range vals {
		switch {
		case v < 1<<7:
			dst = append(dst, byte(v<<1|1))
		case v < 1<<14:
			dst = binary.LittleEndian.AppendUint16(dst, uint16(v<<2|2))
		case v < 1<<29:
			dst = binary.LittleEndian.AppendUint32(dst, uint32(v<<3|4))
		default:
			dst = append(dst, 0x00)
			dst = binary.LittleEndian.AppendUint64(dst, v)
		}
	}
	return dst
}

func DecodeOmniBuf(src []byte, dst []uint64) int {
	i, n := 0, 0
	for i < len(src) {
		b0 := src[i]
		switch {
		case b0&1 != 0:
			dst[n] = uint64(b0 >> 1)
			i++
		case b0&2 != 0:
			dst[n] = uint64(binary.LittleEndian.Uint16(src[i:]) >> 2)
			i += 2
		case b0&4 != 0:
			dst[n] = uint64(binary.LittleEndian.Uint32(src[i:]) >> 3)
			i += 4
		case b0 == 0:
			dst[n] = binary.LittleEndian.Uint64(src[i+1:])
			i += 9
		default:
			return n
		}
		n++
	}
	return n
}

// ---------------------------------------------------------------------------
// 4. Prefix-9 (9-tier unary trailing-zero LSB prefix, LE, 1-9 bytes)
//    Includes both uniform branchless and hybrid 1-byte fast-path decoders.
// ---------------------------------------------------------------------------

func EncodePrefix9(dst []byte, vals []uint64) []byte {
	for _, v := range vals {
		switch {
		case v < 1<<7:
			dst = append(dst, byte(v<<1|1))
		case v < 1<<14:
			dst = binary.LittleEndian.AppendUint16(dst, uint16(v<<2|2))
		case v < 1<<21:
			w := uint32(v<<3 | 4)
			dst = append(dst, byte(w), byte(w>>8), byte(w>>16))
		case v < 1<<28:
			dst = binary.LittleEndian.AppendUint32(dst, uint32(v<<4|8))
		case v < 1<<35:
			w := v<<5 | 16
			dst = append(dst, byte(w), byte(w>>8), byte(w>>16), byte(w>>24), byte(w>>32))
		case v < 1<<42:
			w := v<<6 | 32
			dst = append(dst, byte(w), byte(w>>8), byte(w>>16), byte(w>>24), byte(w>>32), byte(w>>40))
		case v < 1<<49:
			w := v<<7 | 64
			dst = append(dst, byte(w), byte(w>>8), byte(w>>16), byte(w>>24), byte(w>>32), byte(w>>40), byte(w>>48))
		case v < 1<<56:
			dst = append(dst, 0x80)
			dst = append(dst, byte(v), byte(v>>8), byte(v>>16), byte(v>>24), byte(v>>32), byte(v>>40), byte(v>>48))
		default:
			dst = append(dst, 0x00)
			dst = binary.LittleEndian.AppendUint64(dst, v)
		}
	}
	return dst
}

func DecodePrefix9Branchless(src []byte, dst []uint64) int {
	i, n := 0, 0
	limit := len(src) - 9
	for i <= limit {
		b0 := src[i]
		if b0 == 0 {
			dst[n] = binary.LittleEndian.Uint64(src[i+1:])
			i += 9
			n++
			continue
		}
		tz := uint(bits.TrailingZeros8(b0))
		width := tz + 1
		word := binary.LittleEndian.Uint64(src[i:])
		shiftLeft := (8 - width) * 8
		dst[n] = (word << shiftLeft) >> (shiftLeft + width)
		i += int(width)
		n++
	}
	return decodePrefix9Tail(src, i, dst, n)
}

func DecodePrefix9Fast1B(src []byte, dst []uint64) int {
	i, n := 0, 0
	limit := len(src) - 9
	for i <= limit {
		b0 := src[i]
		if b0&1 != 0 {
			dst[n] = uint64(b0 >> 1)
			i++
			n++
			continue
		}
		if b0 == 0 {
			dst[n] = binary.LittleEndian.Uint64(src[i+1:])
			i += 9
			n++
			continue
		}
		tz := uint(bits.TrailingZeros8(b0))
		width := tz + 1
		word := binary.LittleEndian.Uint64(src[i:])
		shiftLeft := (8 - width) * 8
		dst[n] = (word << shiftLeft) >> (shiftLeft + width)
		i += int(width)
		n++
	}
	return decodePrefix9Tail(src, i, dst, n)
}

func decodePrefix9Tail(src []byte, i int, dst []uint64, n int) int {
	for i < len(src) {
		b0 := src[i]
		if b0 == 0 {
			dst[n] = binary.LittleEndian.Uint64(src[i+1:])
			i += 9
			n++
			continue
		}
		tz := uint(bits.TrailingZeros8(b0))
		width := int(tz + 1)
		var word uint64
		for k := 0; k < width; k++ {
			word |= uint64(src[i+k]) << (8 * k)
		}
		dst[n] = word >> width
		i += width
		n++
	}
	return n
}

// ---------------------------------------------------------------------------
// 5. MessagePack Unsigned Int (fixint 0..127 in 1B, else tag + 1/2/4/8B BE)
// ---------------------------------------------------------------------------

func EncodeMsgPack(dst []byte, vals []uint64) []byte {
	for _, v := range vals {
		switch {
		case v <= 127:
			dst = append(dst, byte(v))
		case v <= 0xff:
			dst = append(dst, 0xcc, byte(v))
		case v <= 0xffff:
			dst = append(dst, 0xcd)
			dst = binary.BigEndian.AppendUint16(dst, uint16(v))
		case v <= 0xffffffff:
			dst = append(dst, 0xce)
			dst = binary.BigEndian.AppendUint32(dst, uint32(v))
		default:
			dst = append(dst, 0xcf)
			dst = binary.BigEndian.AppendUint64(dst, v)
		}
	}
	return dst
}

func DecodeMsgPack(src []byte, dst []uint64) int {
	i, n := 0, 0
	for i < len(src) {
		b0 := src[i]
		if b0 <= 127 {
			dst[n] = uint64(b0)
			i++
		} else {
			switch b0 {
			case 0xcc:
				dst[n] = uint64(src[i+1])
				i += 2
			case 0xcd:
				dst[n] = uint64(binary.BigEndian.Uint16(src[i+1:]))
				i += 3
			case 0xce:
				dst[n] = uint64(binary.BigEndian.Uint32(src[i+1:]))
				i += 5
			case 0xcf:
				dst[n] = binary.BigEndian.Uint64(src[i+1:])
				i += 9
			}
		}
		n++
	}
	return n
}

// ---------------------------------------------------------------------------
// 6. Fixed-Width uint64 (safe loop with sliding slice vs unsafe memmove)
// ---------------------------------------------------------------------------

func EncodeFixed64(dst []byte, vals []uint64) []byte {
	for _, v := range vals {
		dst = binary.LittleEndian.AppendUint64(dst, v)
	}
	return dst
}

func DecodeFixed64(src []byte, dst []uint64) int {
	n := len(src) / 8
	dst = dst[:n]
	for j := range dst {
		dst[j] = binary.LittleEndian.Uint64(src[:8])
		src = src[8:]
	}
	return n
}

func EncodeFixed64Unsafe(dst []byte, vals []uint64) []byte {
	if len(vals) == 0 {
		return dst
	}
	raw := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(vals))), len(vals)*8)
	return append(dst, raw...)
}

func DecodeFixed64Unsafe(src []byte, dst []uint64) int {
	n := len(src) / 8
	if n == 0 {
		return 0
	}
	dst = dst[:n]
	out := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(dst))), n*8)
	copy(out, src[:n*8])
	return n
}
