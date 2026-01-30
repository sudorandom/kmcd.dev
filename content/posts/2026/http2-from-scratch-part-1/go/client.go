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
}
