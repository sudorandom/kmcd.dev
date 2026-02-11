package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

func main() {
	// Connect to the Route Views server (standard Telnet port 23)
	// This server allows public read-only access to BGP tables.
	address := "route-views.routeviews.org:23"
	fmt.Printf("Connecting to %s...\n", address)

	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Buffer to store received data
	buf := make([]byte, 1024)
	var accumulated string

	// State tracking
	// 0: waiting for login prompt
	// 1: waiting for first prompt to disable paging
	// 2: waiting for prompt to run command
	// 3: reading command output
	state := 0

	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		chunk := string(buf[:n])
		fmt.Print(chunk)
		accumulated += chunk

		switch state {
		case 0:
			// We expect a login prompt. The username is usually "rviews".
			if strings.Contains(accumulated, "Username:") {
				fmt.Fprintf(conn, "rviews\n")
				accumulated = ""
				state = 1
			}
	
		case 1:
			// Once logged in, disable paging so we get the full output at once.
			// "terminal length 0" tells the router not to pause output.
			if strings.Contains(accumulated, "route-views>") {
				fmt.Fprintf(conn, "terminal length 0\n")
				accumulated = ""
				state = 2
			}
	
		case 2:
			// Wait for prompt again, then run the actual command.
			if strings.Contains(accumulated, "route-views>") {
				fmt.Fprintf(conn, "show ip bgp 8.8.8.8\n")
				accumulated = ""
				state = 3
			}
	
		case 3:
			// If we see the prompt again, the command has finished.
			if strings.Contains(accumulated, "route-views>") {
				return
			}
		}
	}
}