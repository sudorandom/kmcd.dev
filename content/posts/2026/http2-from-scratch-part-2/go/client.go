package main

import (
	"crypto/tls"
	"fmt"
	"log"
)

const (
	// The "Magic" preface required by RFC 9113
	preface = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
	server  = "kmcd.dev:443"
)

func main() {
	// We configure TLS to specifically look for "h2" via ALPN
	config := &tls.Config{
		NextProtos: []string{"h2"},
	}

	// Dial the server and perform the handshake
	conn, err := tls.Dial("tcp", server, config)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Ensure the server actually agreed to speak HTTP/2
	state := conn.ConnectionState()
	if state.NegotiatedProtocol != "h2" {
		log.Fatalf("Server did not negotiate HTTP/2: %s", state.NegotiatedProtocol)
	}

	fmt.Printf("Connected to %s using %s\n", server, state.NegotiatedProtocol)

	// Send the Connection Preface to initialize the H2 session
	_, err = conn.Write([]byte(preface))
	if err != nil {
		log.Fatalf("Failed to send preface: %v", err)
	}

	fmt.Println("Preface sent successfully. The connection is open.")

	frame, err := ReadFrame(conn)
	if err != nil {
		log.Fatalf("Failed to read server settings: %v", err)
	}

	// Check the Type on the Header field of our Frame struct
	if frame.Header.Type != 0x04 {
		log.Fatalf("Expected SETTINGS frame (0x04), got: %d", frame.Header.Type)
	}
	fmt.Printf("Received SETTINGS: %d bytes on Stream %d\n", frame.Header.Length, frame.Header.StreamID)

	// 2. Acknowledge the settings frame
	ackHeader := []byte{
		0x00, 0x00, 0x00, // Length: 0
		0x04,                   // Type: SETTINGS
		0x01,                   // Flags: ACK (0x01)
		0x00, 0x00, 0x00, 0x00, // Stream ID: 0 (Connection level)
	}

	if _, err = conn.Write(ackHeader); err != nil {
		log.Fatalf("Failed to send SETTINGS ACK: %v", err)
	}
	fmt.Println("Sent SETTINGS ACK.")
}
