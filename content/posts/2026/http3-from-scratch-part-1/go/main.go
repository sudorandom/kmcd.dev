package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/quic-go/quic-go"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Step 1: Raw QUIC Echo Demo ===")
	echoTLS, err := generateTLSConfig("quic-echo")
	if err != nil {
		log.Fatal(err)
	}
	echoListener, err := quic.ListenAddr("127.0.0.1:0", echoTLS, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer echoListener.Close()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- runEchoServer(ctx, echoListener)
	}()

	if err := runEchoClient(ctx, echoListener.Addr().String()); err != nil {
		log.Fatal(err)
	}
	if err := <-serverErr; err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n=== Step 2: HTTP/3 Framing & QPACK Over a QUIC Stream ===")
	h3TLS, err := generateTLSConfig("h3")
	if err != nil {
		log.Fatal(err)
	}
	h3Listener, err := quic.ListenAddr("127.0.0.1:0", h3TLS, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer h3Listener.Close()

	go func() {
		conn, err := h3Listener.Accept(ctx)
		if err != nil {
			return
		}

		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return
		}

		req, err := ReadRequest(stream)
		if err != nil {
			log.Printf("server ReadRequest error: %v", err)
			return
		}
		reqBody, _ := io.ReadAll(req.Body)
		fmt.Printf("[H3 Server] Decoded %s %s (Host: %s, User-Agent: %s, Body: %q)\n",
			req.Method, req.URL.Path, req.Host, req.Header.Get("User-Agent"), string(reqBody))

		respHeaders := make(http.Header)
		respHeaders.Set("Content-Type", "text/plain; charset=utf-8")
		respHeaders.Set("Server", "kmcd-h3-scratch")
		respBody := []byte(fmt.Sprintf("Hello over HTTP/3! You sent %d bytes.", len(reqBody)))

		if err := WriteResponse(stream, http.StatusOK, respHeaders, respBody); err != nil {
			log.Printf("server WriteResponse error: %v", err)
		}
		stream.Close()
		<-conn.Context().Done()
	}()

	clientConn, err := quic.DialAddr(ctx, h3Listener.Addr().String(), &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h3"},
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer clientConn.CloseWithError(0, "done")

	reqStream, err := clientConn.OpenStreamSync(ctx)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", "https://kmcd.dev/hello", strings.NewReader("ping from h3"))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "kmcd-h3/0.1")

	if err := WriteRequest(reqStream, req); err != nil {
		log.Fatal(err)
	}
	reqStream.Close() // Send FIN on the request stream

	resp, err := ReadResponse(reqStream)
	if err != nil {
		log.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("[H3 Client] Status: %s\n", resp.Status)
	fmt.Printf("[H3 Client] Content-Type: %s, Server: %s\n",
		resp.Header.Get("Content-Type"), resp.Header.Get("Server"))
	fmt.Printf("[H3 Client] Body: %s\n", string(body))
}
