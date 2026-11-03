package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	StreamTypeControl             uint64 = 0x00
	SettingsQPACKMaxTableCapacity uint64 = 0x01
	SettingsQPACKBlockedStreams   uint64 = 0x07
)

func main() {
	targetURL := "https://kmcd.dev/"
	if len(os.Args) > 1 {
		targetURL = os.Args[1]
		if !strings.Contains(targetURL, "://") {
			targetURL = "https://" + targetURL
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := NewClient()

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "kmcd-h3-scratch/1.0")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	preview := body
	if len(preview) > 200 {
		preview = preview[:200]
	}

	fmt.Printf("URL:          %s\n", targetURL)
	fmt.Printf("Proto:        %s\n", resp.Proto)
	fmt.Printf("Status:       %s\n", resp.Status)
	fmt.Printf("Server:       %s\n", resp.Header.Get("Server"))
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("Alt-Svc:      %s\n", resp.Header.Get("Alt-Svc"))
	fmt.Printf("Body Length:  %d bytes\n", len(body))
	fmt.Printf("Body Preview: %s\n", string(preview))
}
