package main

import (
	"encoding/binary"
	"fmt"
)

// RFC 7541: HPACK: Header Compression for HTTP/2
// https://datatracker.ietf.org/doc/html/rfc7541

const (
	// HPACK Static Table Indices (Masked with 0x80 for Indexed Header Fields)
	// See RFC 7541 Appendix A
	HpackMethodGet   uint8 = 0x82 // Index 2: :method: GET
	HpackPathRoot    uint8 = 0x84 // Index 4: :path: /
	HpackSchemeHttps uint8 = 0x87 // Index 7: :scheme: https

	// 0x40 is the mask for Literal Header Field with Incremental Indexing
	HpackAuthority uint8 = 0x40 | 1 // Index 1: :authority
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
	staticTableMap = make(map[HeaderField]int)
)

func init() {
	for i, hf := range StaticTable {
		staticTableMap[hf] = i + 1
	}
}

type HPACKDecoder struct{}

func NewHPACKDecoder() *HPACKDecoder {
	return &HPACKDecoder{}
}

func (h *HPACKDecoder) Header(i int) (HeaderField, bool) {
	if i <= 0 {
		return HeaderField{}, false
	}
	staticIndex := i - 1
	if staticIndex < len(StaticTable) {
		return StaticTable[staticIndex], true
	}
	return HeaderField{}, false
}

func (h *HPACKDecoder) Decode(payload []byte) error {
	fmt.Printf("Decoding %d bytes\n", len(payload))
	for len(payload) > 0 {
		b := payload[0]
		if b&128 == 128 { // Indexed Header Field
			index, n := decodeInt(payload, 7)
			if n < 0 {
				return fmt.Errorf("failed to decode integer")
			}
			payload = payload[n:]
			header, ok := h.Header(index)
			if !ok {
				return fmt.Errorf("invalid header index: %d", index)
			}
			fmt.Printf("  [Header] %s: %s\n", header.Name, header.Value)
		} else {
			// Other header field types (literal, etc.) not implemented yet
			return fmt.Errorf("not implemented: literal header field")
		}
	}
	return nil
}

// decodeInt decodes a variable-length integer from a byte slice.
// It returns the decoded integer and the number of bytes consumed.
// See RFC 7541 section 5.1 for details.
func decodeInt(payload []byte, n int) (int, int) {
	if len(payload) == 0 {
		return 0, -1
	}
	mask := (1 << n) - 1
	i := int(payload[0]) & mask
	if i < mask {
		return i, 1
	}

	// The value overflows the first byte. The rest of the integer is a
	// standard varint.
	val, bytesRead := binary.Uvarint(payload[1:])
	if bytesRead <= 0 {
		return 0, -1 // Malformed varint
	}

	return i + int(val), 1 + bytesRead
}
