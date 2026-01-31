package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	client := NewClient()

	req, err := http.NewRequest("GET", "https://kmcd.dev/", nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Protocol: %s\n", resp.Proto)

	fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read body: %v", err)
	}
	fmt.Printf("Response Body: %s\n", string(body))
}
