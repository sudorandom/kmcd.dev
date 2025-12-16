package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

// whoisData now holds a single, static WHOIS record for debugging purposes.
var whoisData = map[string]string{
	"google.com": `Domain Name: google.com
Registrar: My Go Server
Creation Date: 2025-12-15T00:00:00Z
`,
	"example.com": `Domain Name: example.com
Registrar: My Go Server
Creation Date: 2025-12-15T00:00:00Z
`,
}

func handleConnection(conn net.Conn) {
	log.Printf("new connection from %s", conn.RemoteAddr())
	defer log.Printf("connection to %s closed", conn.RemoteAddr())
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	var clientQuery string

	// Read the first line from the client.
	if scanner.Scan() {
		clientQuery = strings.TrimSpace(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from client: %v", err)
		return
	}

	if clientQuery == "" {
		log.Printf("Client disconnected without sending a query or sent an empty query.")
		// Send a minimal response in case the client expects *something*
		_, err := fmt.Fprint(conn, "No query provided.\r\n")
		if err != nil {
			log.Printf("Error writing empty query response: %v", err)
		}
		return
	}

	log.Printf("Received query: %q", clientQuery)

	var response string
	if data, ok := whoisData[strings.ToLower(clientQuery)]; ok {
		response = strings.ReplaceAll(data, "\n", "\r\n")
	} else {
		response = fmt.Sprintf("No match for %s\r\n", clientQuery)
	}

	// Write the response and close the connection.
	_, err := fmt.Fprint(conn, response)
	if err != nil {
		log.Printf("Error writing response to client: %v", err)
	}
}

func main() {
	addr := ":43"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Error listening: %v", err)
	}
	defer listener.Close()
	log.Printf("Static WHOIS server listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}
