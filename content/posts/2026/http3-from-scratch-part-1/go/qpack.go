package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/net/http2/hpack"
)

// RFC 9204: QPACK: Field Compression for HTTP/3
// https://www.rfc-editor.org/rfc/rfc9204.html

const (
	// QPACK Field Line representation masks and patterns (RFC 9204 Section 4.5)
	maskQPACKIndexed     = 0x80 // 1xxxxxxx: Indexed Field Line
	flagQPACKIndexedStat = 0x40 // 11xxxxxx: T=1 means Static Table (6-bit index prefix)

	maskQPACKLiteralNameRef     = 0xc0 // 01xxxxxx: Literal Field Line with Name Reference
	patternQPACKLiteralNameRef  = 0x40 // 01NTxxxx
	flagQPACKLiteralNameRefStat = 0x10 // 01x1xxxx: T=1 means Static Table (4-bit index prefix)

	maskQPACKLiteral     = 0xe0 // 001xxxxx: Literal Field Line with Literal Name
	patternQPACKLiteral  = 0x20 // 001NHxxx (H=bit 3, 3-bit name length prefix)
	flagQPACKLiteralHuff = 0x08 // H bit for literal name

	HuffmanFlagMask         = 0x80
	IntegerContinuationMask = 0x80
)

// HeaderField represents an HTTP header name-value pair.
type HeaderField struct {
	Name, Value string
}

// QPACKStaticTable is the 99-entry static table from RFC 9204 Appendix A.
// Unlike HPACK (which is 1-indexed with 61 entries), QPACK is 0-indexed with 99 entries!
var QPACKStaticTable = []HeaderField{
	0:  {Name: ":authority", Value: ""},
	1:  {Name: ":path", Value: "/"},
	2:  {Name: "age", Value: "0"},
	3:  {Name: "content-disposition", Value: ""},
	4:  {Name: "content-length", Value: "0"},
	5:  {Name: "cookie", Value: ""},
	6:  {Name: "date", Value: ""},
	7:  {Name: "etag", Value: ""},
	8:  {Name: "if-modified-since", Value: ""},
	9:  {Name: "if-none-match", Value: ""},
	10: {Name: "last-modified", Value: ""},
	11: {Name: "link", Value: ""},
	12: {Name: "location", Value: ""},
	13: {Name: "referer", Value: ""},
	14: {Name: "set-cookie", Value: ""},
	15: {Name: ":method", Value: "CONNECT"},
	16: {Name: ":method", Value: "DELETE"},
	17: {Name: ":method", Value: "GET"},
	18: {Name: ":method", Value: "HEAD"},
	19: {Name: ":method", Value: "OPTIONS"},
	20: {Name: ":method", Value: "POST"},
	21: {Name: ":method", Value: "PUT"},
	22: {Name: ":scheme", Value: "http"},
	23: {Name: ":scheme", Value: "https"},
	24: {Name: ":status", Value: "103"},
	25: {Name: ":status", Value: "200"},
	26: {Name: ":status", Value: "304"},
	27: {Name: ":status", Value: "404"},
	28: {Name: ":status", Value: "503"},
	29: {Name: "accept", Value: "*/*"},
	30: {Name: "accept", Value: "application/dns-message"},
	31: {Name: "accept-encoding", Value: "gzip, deflate, br"},
	32: {Name: "accept-ranges", Value: "bytes"},
	33: {Name: "access-control-allow-headers", Value: "cache-control"},
	34: {Name: "access-control-allow-headers", Value: "content-type"},
	35: {Name: "access-control-allow-origin", Value: "*"},
	36: {Name: "cache-control", Value: "max-age=0"},
	37: {Name: "cache-control", Value: "max-age=2592000"},
	38: {Name: "cache-control", Value: "max-age=604800"},
	39: {Name: "cache-control", Value: "no-cache"},
	40: {Name: "cache-control", Value: "no-store"},
	41: {Name: "cache-control", Value: "public, max-age=31536000"},
	42: {Name: "content-encoding", Value: "br"},
	43: {Name: "content-encoding", Value: "gzip"},
	44: {Name: "content-type", Value: "application/dns-message"},
	45: {Name: "content-type", Value: "application/javascript"},
	46: {Name: "content-type", Value: "application/json"},
	47: {Name: "content-type", Value: "application/x-www-form-urlencoded"},
	48: {Name: "content-type", Value: "image/gif"},
	49: {Name: "content-type", Value: "image/jpeg"},
	50: {Name: "content-type", Value: "image/png"},
	51: {Name: "content-type", Value: "text/css"},
	52: {Name: "content-type", Value: "text/html; charset=utf-8"},
	53: {Name: "content-type", Value: "text/plain"},
	54: {Name: "content-type", Value: "text/plain;charset=utf-8"},
	55: {Name: "range", Value: "bytes=0-"},
	56: {Name: "strict-transport-security", Value: "max-age=31536000"},
	57: {Name: "strict-transport-security", Value: "max-age=31536000; includesubdomains"},
	58: {Name: "strict-transport-security", Value: "max-age=31536000; includesubdomains; preload"},
	59: {Name: "vary", Value: "accept-encoding"},
	60: {Name: "vary", Value: "origin"},
	61: {Name: "x-content-type-options", Value: "nosniff"},
	62: {Name: "x-frame-options", Value: "deny"},
	63: {Name: "x-frame-options", Value: "sameorigin"},
	64: {Name: ":status", Value: "100"},
	65: {Name: ":status", Value: "204"},
	66: {Name: ":status", Value: "206"},
	67: {Name: ":status", Value: "302"},
	68: {Name: ":status", Value: "400"},
	69: {Name: ":status", Value: "403"},
	70: {Name: ":status", Value: "421"},
	71: {Name: ":status", Value: "425"},
	72: {Name: ":status", Value: "500"},
	73: {Name: "accept-language", Value: ""},
	74: {Name: "access-control-allow-credentials", Value: "FALSE"},
	75: {Name: "access-control-allow-credentials", Value: "TRUE"},
	76: {Name: "access-control-allow-headers", Value: "*"},
	77: {Name: "access-control-allow-methods", Value: "get"},
	78: {Name: "access-control-allow-methods", Value: "get, post, options"},
	79: {Name: "access-control-allow-methods", Value: "options"},
	80: {Name: "access-control-expose-headers", Value: "content-length"},
	81: {Name: "access-control-request-headers", Value: "content-type"},
	82: {Name: "access-control-request-method", Value: "get"},
	83: {Name: "access-control-request-method", Value: "post"},
	84: {Name: "alt-svc", Value: "clear"},
	85: {Name: "authorization", Value: ""},
	86: {Name: "content-security-policy", Value: "script-src 'none'; object-src 'none'; base-uri 'none'"},
	87: {Name: "early-data", Value: "1"},
	88: {Name: "expect-ct", Value: ""},
	89: {Name: "forwarded", Value: ""},
	90: {Name: "if-range", Value: ""},
	91: {Name: "origin", Value: ""},
	92: {Name: "purpose", Value: "prefetch"},
	93: {Name: "server", Value: ""},
	94: {Name: "timing-allow-origin", Value: "*"},
	95: {Name: "upgrade-insecure-requests", Value: "1"},
	96: {Name: "user-agent", Value: ""},
	97: {Name: "x-forwarded-for", Value: ""},
	98: {Name: "x-xss-protection", Value: "1; mode=block"},
}

var (
	qpackStaticMap     = make(map[HeaderField]int)
	qpackStaticNameMap = make(map[string]int)
)

func init() {
	for i, hf := range QPACKStaticTable {
		qpackStaticMap[hf] = i
		if _, ok := qpackStaticNameMap[hf.Name]; !ok {
			qpackStaticNameMap[hf.Name] = i
		}
	}
}

type QPACKDecoder struct{}

func NewQPACKDecoder() *QPACKDecoder {
	return &QPACKDecoder{}
}

// Decode parses a QPACK Encoded Field Section (RFC 9204 Section 4.5).
func (d *QPACKDecoder) Decode(payload []byte) ([]HeaderField, error) {
	r := bytes.NewReader(payload)

	// 1. Read the 2-part Encoded Field Section Prefix (RFC 9204 Section 4.5.1):
	//    - Required Insert Count (8-bit prefix integer)
	//    - S bit + Delta Base (7-bit prefix integer)
	b1, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("reading QPACK required insert count: %w", err)
	}
	reqInsertCount, n := decodeInt(b1, r, 8)
	if n < 0 || reqInsertCount != 0 {
		return nil, fmt.Errorf("unsupported QPACK dynamic table insert count: %d", reqInsertCount)
	}

	b2, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("reading QPACK delta base: %w", err)
	}
	if _, n := decodeInt(b2, r, 7); n < 0 {
		return nil, fmt.Errorf("invalid QPACK delta base")
	}

	// 2. Decode Field Lines
	var headers []HeaderField
	for r.Len() > 0 {
		b, _ := r.ReadByte()
		switch {
		case b&maskQPACKIndexed != 0:
			// Indexed Field Line (1 T Index(6+))
			isStatic := b&flagQPACKIndexedStat != 0
			index, n := decodeInt(b, r, 6)
			if n < 0 || !isStatic || index >= len(QPACKStaticTable) {
				return nil, fmt.Errorf("invalid QPACK static index: %d", index)
			}
			headers = append(headers, QPACKStaticTable[index])

		case b&maskQPACKLiteralNameRef == patternQPACKLiteralNameRef:
			// Literal Field Line with Name Reference (01 N T NameIndex(4+))
			isStatic := b&flagQPACKLiteralNameRefStat != 0
			index, n := decodeInt(b, r, 4)
			if n < 0 || !isStatic || index >= len(QPACKStaticTable) {
				return nil, fmt.Errorf("invalid QPACK name reference index: %d", index)
			}
			val, err := decodeString(r)
			if err != nil {
				return nil, err
			}
			headers = append(headers, HeaderField{
				Name:  QPACKStaticTable[index].Name,
				Value: val,
			})

		case b&maskQPACKLiteral == patternQPACKLiteral:
			// Literal Field Line with Literal Name (001 N H NameLen(3+) Name Value)
			nameHuffman := b&flagQPACKLiteralHuff != 0
			nameLen, n := decodeInt(b, r, 3)
			if n < 0 || r.Len() < nameLen {
				return nil, fmt.Errorf("invalid QPACK literal name length")
			}
			nameBytes := make([]byte, nameLen)
			r.Read(nameBytes)
			var name string
			if nameHuffman {
				name, err = hpack.HuffmanDecodeToString(nameBytes)
				if err != nil {
					return nil, err
				}
			} else {
				name = string(nameBytes)
			}
			val, err := decodeString(r)
			if err != nil {
				return nil, err
			}
			headers = append(headers, HeaderField{Name: name, Value: val})

		default:
			return nil, fmt.Errorf("unsupported QPACK field line prefix: %08b", b)
		}
	}
	return headers, nil
}

type QPACKEncoder struct{}

func NewQPACKEncoder() *QPACKEncoder {
	return &QPACKEncoder{}
}

// Encode serializes headers into a QPACK Encoded Field Section using the Static Table and Literals.
func (e *QPACKEncoder) Encode(headers []HeaderField) []byte {
	var buf bytes.Buffer
	// Field Section Prefix: Required Insert Count = 0, Delta Base = 0
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)

	for _, hf := range headers {
		// 1. Exact Name + Value match in QPACK Static Table (11xxxxxx, 6-bit prefix)
		if index, ok := qpackStaticMap[hf]; ok {
			encodeInt(&buf, index, 6, maskQPACKIndexed|flagQPACKIndexedStat)
			continue
		}
		// 2. Name match in QPACK Static Table (0101xxxx, 4-bit prefix)
		if index, ok := qpackStaticNameMap[hf.Name]; ok {
			encodeInt(&buf, index, 4, patternQPACKLiteralNameRef|flagQPACKLiteralNameRefStat)
			encodeString(&buf, hf.Value)
			continue
		}
		// 3. Literal Name + Literal Value (00100xxx, 3-bit name length prefix)
		encodeInt(&buf, len(hf.Name), 3, patternQPACKLiteral)
		buf.WriteString(hf.Name)
		encodeString(&buf, hf.Value)
	}
	return buf.Bytes()
}

// decodeString, decodeInt, encodeInt, and encodeString are reused directly from our HTTP/2 HPACK code!
func decodeString(r *bytes.Reader) (string, error) {
	b, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	huffman := b&HuffmanFlagMask == HuffmanFlagMask
	length, n := decodeInt(b, r, 7)
	if n < 0 {
		return "", fmt.Errorf("failed to decode string length")
	}
	if r.Len() < length {
		return "", io.ErrUnexpectedEOF
	}
	data := make([]byte, length)
	r.Read(data)
	if huffman {
		return hpack.HuffmanDecodeToString(data)
	}
	return string(data), nil
}

func decodeInt(b byte, r *bytes.Reader, n int) (int, int) {
	mask := (1 << n) - 1
	i := int(b) & mask
	if i < mask {
		return i, 1
	}

	var m uint = 0
	bytesRead := 1
	var val uint64
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, -bytesRead
		}
		bytesRead++
		val |= uint64(b&127) << m
		m += 7
		if b&IntegerContinuationMask == 0 {
			break
		}
	}
	return i + int(val), bytesRead
}

func encodeInt(buf *bytes.Buffer, i int, n int, pattern byte) {
	mask := (1 << n) - 1
	if i < mask {
		buf.WriteByte(pattern | byte(i))
	} else {
		buf.WriteByte(pattern | byte(mask))
		i -= mask
		varint := make([]byte, binary.MaxVarintLen64)
		c := binary.PutUvarint(varint, uint64(i))
		buf.Write(varint[:c])
	}
}

func encodeString(buf *bytes.Buffer, s string) {
	encodeInt(buf, len(s), 7, 0x00)
	buf.WriteString(s)
}
