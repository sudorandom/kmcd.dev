package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func main() {
	queryPayload := `{"filters": {"status": "active", "category": "electronics"}, "fields": ["name", "price"]}`
	client := &http.Client{}

	// Create a new request with the custom "QUERY" method
	req, err := http.NewRequest("QUERY", "http://localhost:8080/data", strings.NewReader(queryPayload))
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json") // Important for body processing

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}
	fmt.Println("Response Body:", string(body))
}
