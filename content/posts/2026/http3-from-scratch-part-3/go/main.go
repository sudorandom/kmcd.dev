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

// ListenQUIC and DialQUIC now point to our own UDP-based QUIC implementation!
var (
	ListenQUIC = ListenToyQUIC
	DialQUIC   = DialToyQUIC
)

// HTTP/3 Unidirectional Stream Types & Settings Constants (from Part 2)
const (
	StreamTypeControl             uint64 = 0x00
	SettingsQPACKMaxTableCapacity uint64 = 0x01
	SettingsQPACKBlockedStreams   uint64 = 0x07
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Transport", "toy-quic-udp")
		fmt.Fprintf(w, "HTTP/3 on homemade QUIC! Method=%s Path=%s\n", r.Method, r.URL.Path)
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

	listener, err := ListenQUIC("127.0.0.1:0", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	srv := &Server{Handler: mux}
	go func() {
		_ = srv.ServeListener(ctx, listener)
	}()

	addr := listener.Addr().String()
	fmt.Printf("Homemade QUIC + HTTP/3 server listening on UDP %s\n\n", addr)

	client := NewClient()

	// 1. GET /hello over our own HTTP/3 + QPACK + QUIC + UDP stack!
	req1, _ := http.NewRequestWithContext(ctx, "GET", "https://"+addr+"/hello", nil)
	req1.Header.Set("User-Agent", "kmcd-full-scratch/1.0")
	resp1, err := client.Do(req1)
	if err != nil {
		log.Fatal(err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	fmt.Printf("[GET /hello] Status: %s | X-Transport: %s | Body: %s",
		resp1.Status, resp1.Header.Get("X-Transport"), string(body1))

	// 2. POST /echo over our own HTTP/3 + QPACK + QUIC + UDP stack!
	req2, _ := http.NewRequestWithContext(ctx, "POST", "https://"+addr+"/echo",
		strings.NewReader("Zero quic-go imports! Pure UDP -> QUIC -> QPACK -> HTTP/3!"))
	resp2, err := client.Do(req2)
	if err != nil {
		log.Fatal(err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	fmt.Printf("[POST /echo] Status: %s | Body: %s\n", resp2.Status, string(body2))
}
