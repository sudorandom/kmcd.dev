package main

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"log"
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

func main() {
	// 1. Setup TLS with ALPN
	config := &tls.Config{
		NextProtos: []string{"h2"},
	}

	conn, err := tls.Dial("tcp", Server, config)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if state.NegotiatedProtocol != "h2" {
		log.Fatalf("Server did not negotiate HTTP/2: %s", state.NegotiatedProtocol)
	}

	fmt.Printf("Connected to %s using %s\n", Server, state.NegotiatedProtocol)

	// 2. Send Connection Preface
	if _, err = conn.Write([]byte(Preface)); err != nil {
		log.Fatalf("Failed to send preface: %v", err)
	}
	fmt.Println("Preface sent.")

	// 3. Initial Handshake Loop
	hpackDec := NewHPACKDecoder()

	// Send our initial empty settings
	mySettings := []byte{0, 0, 0, FrameSettings, 0, 0, 0, 0, 0}
	conn.Write(mySettings)

	serverSettingsAcked := false
	mySettingsAcked := false

	for !serverSettingsAcked || !mySettingsAcked {
		frame, err := ReadFrame(conn)
		if err != nil {
			log.Fatalf("Handshake read error: %v", err)
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
			log.Fatalf("Server sent GOAWAY during handshake")
		}
	}

	// 4. Send the Request (GET /)
	authority := "kmcd.dev"
	requestPayload := []byte{
		HpackMethodGet,
		HpackPathRoot,
		HpackSchemeHttps,
		HpackAuthority,
		byte(len(authority)), // The length prefix for the string literal
	}
	requestPayload = append(requestPayload, []byte(authority)...)

	header := make([]byte, 9)
	payloadLen := len(requestPayload)
	header[0] = byte(payloadLen >> 16)
	header[1] = byte(payloadLen >> 8)
	header[2] = byte(payloadLen)
	header[3] = FrameHeaders
	header[4] = FlagEndStream | FlagEndHeaders
	binary.BigEndian.PutUint32(header[5:9], 1)

	if _, err = conn.Write(append(header, requestPayload...)); err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
	fmt.Println(">>> Sent HEADERS (Stream 1)")

	// 5. Main Processing Loop
	for {
		frame, err := ReadFrame(conn)
		if err != nil {
			log.Printf("Connection closed: %v", err)
			break
		}

		fmt.Printf("<<< Frame Type=%d, Flags=%d, Stream=%d\n",
			frame.Header.Type, frame.Header.Flags, frame.Header.StreamID)

		switch frame.Header.Type {
		case FrameData:
			fmt.Printf("      [DATA] %s\n", string(frame.Payload))
		case FrameHeaders:
			fmt.Println("      [HEADERS] Decoding...")
			if err := hpackDec.Decode(frame.Payload); err != nil {
				log.Printf("HPACK Error: %v", err)
			}
		case FrameGoAway:
			lastStream := binary.BigEndian.Uint32(frame.Payload[0:4]) & 0x7FFFFFFF
			errCode := binary.BigEndian.Uint32(frame.Payload[4:8])
			fmt.Printf("!!! GOAWAY: Last Stream %d, Error Code %d\n", lastStream, errCode)
			return
		case FrameWindowUpdate:
			fmt.Println("      [WINDOW_UPDATE]")
		}

		// Use the global constants for the exit condition
		isEndFrame := frame.Header.Type == FrameData || frame.Header.Type == FrameHeaders
		if isEndFrame && (frame.Header.Flags&FlagEndStream != 0) {
			fmt.Println("Stream finished. Exiting.")
			break
		}
	}
}
