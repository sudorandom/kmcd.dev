package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run traceroute.go <destination>")
		os.Exit(1)
	}
	destination := os.Args[1]
	dstAddr, err := net.ResolveIPAddr("ip4", destination)
	if err != nil {
		log.Fatalf("Could not resolve destination: %s", err)
	}

	// Listen for ICMP packets
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Fatalf("ListenPacket failed: %s", err)
	}
	defer c.Close()

	fmt.Printf("Traceroute to %s (%s)\n", destination, dstAddr)

	for ttl := 1; ttl <= 64; ttl++ {
		start := time.Now()

		// Set TTL
		if err := c.IPv4PacketConn().SetTTL(ttl); err != nil {
			log.Fatalf("SetTTL failed: %s", err)
		}

		// Create ICMP Echo Message
		m := icmp.Message{
			Type: ipv4.ICMPTypeEcho, Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  ttl,
				Data: []byte("HELLO-TRACEROUTE"),
			},
		}
		b, err := m.Marshal(nil)
		if err != nil {
			log.Fatalf("Marshal failed: %s", err)
		}

		// Send
		if _, err := c.WriteTo(b, dstAddr); err != nil {
			log.Fatalf("WriteTo failed: %s", err)
		}

		// Wait for reply
		reply := make([]byte, 1500) // 1500 is the standard MTU (Maximum Transmission Unit) for Ethernet
		if err := c.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
			log.Fatalf("SetReadDeadline failed: %s", err)
		}
		n, peer, err := c.ReadFrom(reply)
		if err != nil {
			fmt.Printf("%d\t*\t*\t*\n", ttl) // Timeout
			continue
		}
		elapsed := time.Since(start)

		// Parse the reply message
		rm, err := icmp.ParseMessage(1, reply[:n]) // 1 for ICMPv4
		if err != nil {
			log.Printf("Error parsing ICMP message: %s", err)
			continue
		}

		// Check if the reply is for our process and probe
		switch rm.Type {
		case ipv4.ICMPTypeEchoReply:
			if rm.Body.(*icmp.Echo).ID != os.Getpid()&0xffff {
				continue // Not our packet
			}
			fmt.Printf("%d\t%v\t%v\n", ttl, peer, elapsed)
			fmt.Println("Destination reached.")
			return // We are done
		case ipv4.ICMPTypeTimeExceeded:
			// For simplicity, we assume any TimeExceeded message is for our probe.
			// A robust implementation would parse the body of the message
			// to verify the ID of the original packet.
			fmt.Printf("%d\t%v\t%v\n", ttl, peer, elapsed)
		default:
			// This could be Destination Unreachable or other types. We'll ignore them for this simple tool.
			fmt.Printf("%d\t%v\t%v (type %d)\n", ttl, peer, elapsed, rm.Type)
		}
	}
}
