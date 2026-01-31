package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/net/http2/hpack"
)

// RFC 7541: HPACK: Header Compression for HTTP/2
// https://datatracker.ietf.org/doc/html/rfc7541

const (
	// Masks for HPACK header field types
	maskIndexed            = 0x80 // 10000000
	maskLiteralIncremental = 0xc0 // 11000000
	maskDynamicTableSize   = 0xe0 // 11100000
	maskLiteral            = 0xf0 // 11110000

	// Patterns for HPACK header field types
	patternIndexed            = 0x80 // 10000000
	patternLiteralIncremental = 0x40 // 01000000
	patternDynamicTableSize   = 0x20 // 00100000
	patternLiteralNever       = 0x10 // 00010000
	patternLiteral            = 0x00 // 00000000

	HuffmanFlagMask         = 0x80
	IntegerContinuationMask = 0x80
)

// HeaderField represents a header field with a name and value.
type HeaderField struct {
	Name, Value string
}

// StaticTable is the predefined, unchangeable table of header fields, as defined in RFC 7541 Appendix A.
var StaticTable = []HeaderField{
	{Name: ":authority", Value: ""},
	{Name: ":method", Value: "GET"},
	{Name: ":method", Value: "POST"},
	{Name: ":path", Value: "/"},
	{Name: ":path", Value: "/index.html"},
	{Name: ":scheme", Value: "http"},
	{Name: ":scheme", Value: "https"},
	{Name: ":status", Value: "200"},
	{Name: ":status", Value: "204"},
	{Name: ":status", Value: "206"},
	{Name: ":status", Value: "304"},
	{Name: ":status", Value: "400"},
	{Name: ":status", Value: "404"},
	{Name: ":status", Value: "500"},
	{Name: "accept-charset", Value: ""},
	{Name: "accept-encoding", Value: "gzip, deflate"},
	{Name: "accept-language", Value: ""},
	{Name: "accept-ranges", Value: ""},
	{Name: "accept", Value: ""},
	{Name: "access-control-allow-origin", Value: ""},
	{Name: "age", Value: ""},
	{Name: "allow", Value: ""},
	{Name: "authorization", Value: ""},
	{Name: "cache-control", Value: ""},
	{Name: "content-disposition", Value: ""},
	{Name: "content-encoding", Value: ""},
	{Name: "content-language", Value: ""},
	{Name: "content-length", Value: ""},
	{Name: "content-location", Value: ""},
	{Name: "content-range", Value: ""},
	{Name: "content-type", Value: ""},
	{Name: "cookie", Value: ""},
	{Name: "date", Value: ""},
	{Name: "etag", Value: ""},
	{Name: "expect", Value: ""},
	{Name: "expires", Value: ""},
	{Name: "from", Value: ""},
	{Name: "host", Value: ""},
	{Name: "if-match", Value: ""},
	{Name: "if-modified-since", Value: ""},
	{Name: "if-none-match", Value: ""},
	{Name: "if-range", Value: ""},
	{Name: "if-unmodified-since", Value: ""},
	{Name: "last-modified", Value: ""},
	{Name: "link", Value: ""},
	{Name: "location", Value: ""},
	{Name: "max-forwards", Value: ""},
	{Name: "proxy-authenticate", Value: ""},
	{Name: "proxy-authorization", Value: ""},
	{Name: "range", Value: ""},
	{Name: "referer", Value: ""},
	{Name: "retry-after", Value: ""},
	{Name: "server", Value: ""},
	{Name: "set-cookie", Value: ""},
	{Name: "strict-transport-security", Value: ""},
	{Name: "transfer-encoding", Value: ""},
	{Name: "user-agent", Value: ""},
	{Name: "vary", Value: ""},
	{Name: "via", Value: ""},
	{Name: "www-authenticate", Value: ""},
}

var (
	staticTableMap     = make(map[HeaderField]int)
	staticTableNameMap = make(map[string]int)
)

func init() {
	for i, hf := range StaticTable {
		staticTableMap[hf] = i + 1
		if _, ok := staticTableNameMap[hf.Name]; !ok {
			staticTableNameMap[hf.Name] = i + 1
		}
	}
}

type DynamicTable struct {
	headers []HeaderField
	size    uint32
	maxSize uint32
}

func NewDynamicTable(maxSize uint32) *DynamicTable {
	return &DynamicTable{
		maxSize: maxSize,
	}
}

func (d *DynamicTable) At(i int) (HeaderField, bool) {
	if i < 0 || i >= len(d.headers) {
		return HeaderField{}, false
	}
	return d.headers[i], true
}

func (d *DynamicTable) Add(h HeaderField) {
	size := uint32(len(h.Name) + len(h.Value) + 32)
	for d.size+size > d.maxSize && len(d.headers) > 0 {
		last := d.headers[len(d.headers)-1]
		d.size -= uint32(len(last.Name) + len(last.Value) + 32)
		d.headers = d.headers[:len(d.headers)-1]
	}
	d.headers = append([]HeaderField{h}, d.headers...)
	d.size += size
}

func (d *DynamicTable) SetMaxSize(size uint32) {
	d.maxSize = size
	for d.size > d.maxSize && len(d.headers) > 0 {
		last := d.headers[len(d.headers)-1]
		d.size -= uint32(len(last.Name) + len(last.Value) + 32)
		d.headers = d.headers[:len(d.headers)-1]
	}
}

type HPACKDecoder struct {
	dynamicTable *DynamicTable
}

func NewHPACKDecoder(maxSize uint32) *HPACKDecoder {
	return &HPACKDecoder{
		dynamicTable: NewDynamicTable(maxSize),
	}
}

func (h *HPACKDecoder) Header(i int) (HeaderField, bool) {
	if i <= 0 {
		return HeaderField{}, false
	}
	if i <= len(StaticTable) {
		return StaticTable[i-1], true
	}
	return h.dynamicTable.At(i - len(StaticTable) - 1)
}

func (h *HPACKDecoder) Decode(payload []byte) ([]HeaderField, error) {
	var headers []HeaderField
	r := bytes.NewReader(payload)
	for r.Len() > 0 {
		b, _ := r.ReadByte()
		if b&maskIndexed == patternIndexed { // Indexed Header Field
			index, n := decodeInt(b, r, 7)
			if n < 0 {
				return nil, fmt.Errorf("failed to decode integer")
			}
			header, ok := h.Header(index)
			if !ok {
				return nil, fmt.Errorf("invalid header index: %d", index)
			}
			headers = append(headers, header)
			fmt.Printf("  [Header] %s: %s\n", header.Name, header.Value)
		} else if b&maskLiteralIncremental == patternLiteralIncremental { // Literal Header Field with Incremental Indexing
			index, n := decodeInt(b, r, 6)
			if n < 0 {
				return nil, fmt.Errorf("failed to decode integer")
			}
			header, err := h.decodeLiteralHeader(r, index, true)
			if err != nil {
				return nil, err
			}
			headers = append(headers, header)
			fmt.Printf("  [Header] %s: %s\n", header.Name, header.Value)
		} else if b&maskLiteral == patternLiteral || b&maskLiteral == patternLiteralNever { // Literal Header Field without or never indexed
			index, n := decodeInt(b, r, 4)
			if n < 0 {
				return nil, fmt.Errorf("failed to decode integer")
			}
			header, err := h.decodeLiteralHeader(r, index, false)
			if err != nil {
				return nil, err
			}
			headers = append(headers, header)
			fmt.Printf("  [Header] %s: %s\n", header.Name, header.Value)
		} else if b&maskDynamicTableSize == patternDynamicTableSize { // Dynamic Table Size Update
			size, n := decodeInt(b, r, 5)
			if n < 0 {
				return nil, fmt.Errorf("failed to decode integer")
			}
			h.dynamicTable.SetMaxSize(uint32(size))
		} else {
			return nil, fmt.Errorf("not implemented: unknown header field type %08b", b)
		}
	}
	return headers, nil
}

func (h *HPACKDecoder) decodeLiteralHeader(r *bytes.Reader, index int, addToDynamicTable bool) (HeaderField, error) {
	var name string
	var err error
	if index > 0 {
		header, ok := h.Header(index)
		if !ok {
			return HeaderField{}, fmt.Errorf("invalid header index: %d", index)
		}
		name = header.Name
	} else {
		name, err = h.decodeString(r)
		if err != nil {
			return HeaderField{}, err
		}
	}
	value, err := h.decodeString(r)
	if err != nil {
		return HeaderField{}, err
	}
	header := HeaderField{Name: name, Value: value}
	if addToDynamicTable {
		h.dynamicTable.Add(header)
	}
	return header, nil
}

func (h *HPACKDecoder) decodeString(r *bytes.Reader) (string, error) {
	b, _ := r.ReadByte()
	huffman := b&HuffmanFlagMask == HuffmanFlagMask
	length, n := decodeInt(b, r, 7)
	if n < 0 {
		return "", fmt.Errorf("failed to decode integer")
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

type HPACKEncoder struct {
	dynamicTable *DynamicTable
}

func NewHPACKEncoder(maxSize uint32) *HPACKEncoder {
	return &HPACKEncoder{
		dynamicTable: NewDynamicTable(maxSize),
	}
}

func (e *HPACKEncoder) Encode(headers []HeaderField) []byte {
	var buf bytes.Buffer
	for _, hf := range headers {
		// 1. Find a match in static table
		if index, ok := staticTableMap[hf]; ok {
			encodeInt(&buf, index, 7, patternIndexed)
			continue
		}

		// 2. Find a name match in static table
		if index, ok := staticTableNameMap[hf.Name]; ok {
			encodeInt(&buf, index, 6, patternLiteralIncremental)
			encodeString(&buf, hf.Value)
			e.dynamicTable.Add(hf)
			continue
		}

		// 3. Literal with literal name
		encodeInt(&buf, 0, 6, patternLiteralIncremental)
		encodeString(&buf, hf.Name)
		encodeString(&buf, hf.Value)
		e.dynamicTable.Add(hf)
	}
	return buf.Bytes()
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
	// no huffman for now
	encodeInt(buf, len(s), 7, 0x00) // This is patternLiteral. We are encoding a raw string.
	buf.WriteString(s)
}
