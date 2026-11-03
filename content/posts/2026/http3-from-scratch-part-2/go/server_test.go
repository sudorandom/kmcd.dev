package main

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/quic-go/quic-go/http3"
)

// TestInteropWithQuicGoHTTP3 verifies that our from-scratch HTTP/3 server and client
// speak RFC-compliant HTTP/3 + QPACK by testing our Server against quic-go's official
// http3.Transport, and our Client against our Server!
func TestInteropWithQuicGoHTTP3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Server-Impl", "from-scratch")
		_, _ = w.Write([]byte("got: " + string(body)))
	})

	tlsConf, err := generateTLSConfig("h3")
	if err != nil {
		t.Fatal(err)
	}
	listener, err := ListenQUIC("127.0.0.1:0", tlsConf)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	srv := &Server{Handler: mux}
	go func() {
		_ = srv.ServeListener(ctx, listener)
	}()

	// Use quic-go's official http3.Transport to call our from-scratch HTTP/3 server!
	officialTransport := &http3.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"h3"},
		},
	}
	defer officialTransport.Close()
	officialClient := &http.Client{Transport: officialTransport}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://"+listener.Addr().String()+"/hello", strings.NewReader("real browser/client test"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := officialClient.Do(req)
	if err != nil {
		t.Fatalf("official http3.Transport failed against our Server: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "got: real browser/client test" {
		t.Fatalf("got body %q, want %q", string(body), "got: real browser/client test")
	}
	if got := resp.Header.Get("X-Server-Impl"); got != "from-scratch" {
		t.Fatalf("got X-Server-Impl %q, want from-scratch", got)
	}
}

func TestClientAgainstOfficialServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tlsConf, err := generateTLSConfig("h3")
	if err != nil {
		t.Fatal(err)
	}
	qListener, err := ListenQUIC("127.0.0.1:0", tlsConf)
	if err != nil {
		t.Fatal(err)
	}
	defer qListener.Close()

	officialSrv := &http3.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Official", "quic-go")
			_, _ = w.Write([]byte("official server says hi to " + r.URL.Path))
		}),
	}
	go func() {
		_ = officialSrv.ServeListener(qListener.(*quicGoListener).Listener)
	}()

	client := NewClient()
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://"+qListener.Addr().String()+"/hello", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("our Client failed against official http3.Server: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "official server says hi to /hello" {
		t.Fatalf("got body %q", string(body))
	}
	if got := resp.Header.Get("X-Official"); got != "quic-go" {
		t.Fatalf("got X-Official %q, want quic-go", got)
	}
}
