package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
)

const (
	// Protocol constants
	Preface = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
	Server  = "kmcd.dev:443"

	// Frame Types (RFC 9113 Section 6)
	FrameData         uint8 = 0x0
	FrameHeaders      uint8 = 0x1
	FramePriority     uint8 = 0x2
	FrameRstStream    uint8 = 0x3
	FrameSettings     uint8 = 0x4
	FramePushPromise  uint8 = 0x5
	FramePing         uint8 = 0x6
	FrameGoAway       uint8 = 0x7
	FrameWindowUpdate uint8 = 0x8
	FrameContinuation uint8 = 0x9

	// Common Flags
	FlagAck        uint8 = 0x01 // For SETTINGS/PING
	FlagEndStream  uint8 = 0x01 // For DATA/HEADERS
	FlagEndHeaders uint8 = 0x04 // For HEADERS/PUSH_PROMISE/CONTINUATION
)

type Client struct {
	addr     string
	hpackDec *HPACKDecoder
	hpackEnc *HPACKEncoder
}

func NewClient(addr string) *Client {
	return &Client{
		addr:     addr,
		hpackDec: NewHPACKDecoder(4096),
		hpackEnc: NewHPACKEncoder(4096),
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// 1. Setup TLS with ALPN
	config := &tls.Config{
		NextProtos: []string{"h2"},
	}

	conn, err := tls.Dial("tcp", c.addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close() // Close connection after Do returns

	state := conn.ConnectionState()
	if state.NegotiatedProtocol != "h2" {
		return nil, fmt.Errorf("server did not negotiate HTTP/2: %s", state.NegotiatedProtocol)
	}

	fmt.Printf("Connected to %s using %s\n", c.addr, state.NegotiatedProtocol)

	// 2. Send Connection Preface
	if _, err = conn.Write([]byte(Preface)); err != nil {
		return nil, fmt.Errorf("failed to send preface: %w", err)
	}
	fmt.Println("Preface sent.")

	// 3. Initial Handshake Loop (using the Client's hpackDec)
	// Send our initial empty settings
	mySettings := []byte{0, 0, 0, FrameSettings, 0, 0, 0, 0, 0}
	conn.Write(mySettings)

	serverSettingsAcked := false
	mySettingsAcked := false

	for !serverSettingsAcked || !mySettingsAcked {
		frame, err := ReadFrame(conn) // ReadFrame needs the raw conn
		if err != nil {
			return nil, fmt.Errorf("handshake read error: %w", err)
		}

		fmt.Printf("<<< [Handshake] Frame Type=%d, Flags=%d, Stream=%d\n",
			frame.Header.Type, frame.Header.Flags, frame.Header.StreamID)

		switch frame.Header.Type {
		case FrameSettings:
			if frame.Header.Flags&FlagAck != 0 {
				mySettingsAcked = true
				fmt.Println("<<< Server ACK'd our settings")
			} else {
				ack := []byte{0, 0, 0, FrameSettings, FlagAck, 0, 0, 0, 0}
				conn.Write(ack)
				serverSettingsAcked = true
				fmt.Println(">>> Sent SETTINGS ACK")
			}
		case FrameWindowUpdate:
			fmt.Println("<<< Server provided flow control window")
		case FrameGoAway:
			return nil, fmt.Errorf("server sent GOAWAY during handshake")
		}
	}

	// Now the actual request sending logic from previous Do method
	headers := []HeaderField{
		{Name: ":method", Value: req.Method},
		{Name: ":scheme", Value: req.URL.Scheme},
		{Name: ":authority", Value: req.URL.Host},
		{Name: ":path", Value: req.URL.Path},
	}
	for name, values := range req.Header {
		for _, value := range values {
			headers = append(headers, HeaderField{Name: name, Value: value})
		}
	}

	requestPayload := c.hpackEnc.Encode(headers)

	header := make([]byte, 9)
	payloadLen := len(requestPayload)
	header[0] = byte(payloadLen >> 16)
	header[1] = byte(payloadLen >> 8)
	header[2] = byte(payloadLen)
	header[3] = FrameHeaders
	header[4] = FlagEndStream | FlagEndHeaders
	binary.BigEndian.PutUint32(header[5:9], 1)

	if _, err := conn.Write(append(header, requestPayload...)); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	fmt.Println(">>> Sent HEADERS (Stream 1)")

	// Read response (using the conn local to Do)
	var respHeaders []HeaderField
	var respBody []byte
	for {
		frame, err := ReadFrame(conn) // ReadFrame needs the raw conn
		if err != nil {
			return nil, fmt.Errorf("connection closed: %w", err)
		}

		fmt.Printf("<<< Frame Type=%d, Flags=%d, Stream=%d\n",
			frame.Header.Type, frame.Header.Flags, frame.Header.StreamID)

		switch frame.Header.Type {
		case FrameData:
			respBody = append(respBody, frame.Payload...)
			fmt.Printf("      [DATA] %s\n", string(frame.Payload))
		case FrameHeaders:
			fmt.Println("      [HEADERS] Decoding...")
			headers, err := c.hpackDec.Decode(frame.Payload)
			if err != nil {
				return nil, fmt.Errorf("hpack error: %w", err)
			}
			respHeaders = append(respHeaders, headers...)
		case FrameGoAway:
			lastStream := binary.BigEndian.Uint32(frame.Payload[0:4]) & 0x7FFFFFFF
			errCode := binary.BigEndian.Uint32(frame.Payload[4:8])
			return nil, fmt.Errorf("GOAWAY: Last Stream %d, Error Code %d", lastStream, errCode)
		case FrameWindowUpdate:
			fmt.Println("      [WINDOW_UPDATE]")
		}

		isEndFrame := frame.Header.Type == FrameData || frame.Header.Type == FrameHeaders
		if isEndFrame && (frame.Header.Flags&FlagEndStream != 0) {
			fmt.Println("Stream finished.")
			break
		}
	}

	// Build http.Response
	httpResp := &http.Response{
		StatusCode: 200, // Hardcoded for now
		Proto:      "HTTP/2.0",
		ProtoMajor: 2,
		ProtoMinor: 0,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(respBody)),
	}

	for _, h := range respHeaders {
		httpResp.Header.Add(h.Name, h.Value)
		if h.Name == ":status" {
			fmt.Sscanf(h.Value, "%d", &httpResp.StatusCode)
		}
	}
	httpResp.Status = fmt.Sprintf("%d %s", httpResp.StatusCode, http.StatusText(httpResp.StatusCode))

	return httpResp, nil
}
