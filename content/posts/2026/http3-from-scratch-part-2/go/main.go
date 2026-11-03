package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "Welcome to HTTP/3 from scratch! You requested %s %s\n", r.Method, r.URL.Path)
	})
	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(r.Header)
	})
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.Copy(w, r.Body)
	})

	tlsConf, err := generateTLSConfig("h3")
	if err != nil {
		log.Fatal(err)
	}

	listener, err := ListenQUIC("127.0.0.1:0", tlsConf)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	srv := &Server{
		Handler: mux,
	}
	go func() {
		_ = srv.ServeListener(ctx, listener)
	}()

	addr := listener.Addr().String()
	fmt.Printf("HTTP/3 server listening on UDP %s\n\n", addr)

	client := NewClient()

	// 1. Test GET /
	req1, _ := http.NewRequestWithContext(ctx, "GET", "https://"+addr+"/", nil)
	req1.Header.Set("User-Agent", "kmcd-h3-client/1.0")
	resp1, err := client.Do(req1)
	if err != nil {
		log.Fatal(err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	fmt.Printf("[GET /] Status: %s | Body: %s", resp1.Status, string(body1))

	// 2. Test GET /headers
	req2, _ := http.NewRequestWithContext(ctx, "GET", "https://"+addr+"/headers", nil)
	req2.Header.Set("X-Custom-Header", "true")
	resp2, err := client.Do(req2)
	if err != nil {
		log.Fatal(err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	fmt.Printf("[GET /headers] Status: %s | Body: %s", resp2.Status, string(body2))

	// 3. Test POST /echo
	req3, _ := http.NewRequestWithContext(ctx, "POST", "https://"+addr+"/echo", strings.NewReader("Echoing over HTTP/3 DATA frames!"))
	resp3, err := client.Do(req3)
	if err != nil {
		log.Fatal(err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	fmt.Printf("[POST /echo] Status: %s | Body: %s\n", resp3.Status, string(body3))
}
