package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/quic-go/quic-go"
)

// generateTLSConfig creates an in-memory self-signed certificate for our QUIC demos.
// QUIC mandates TLS 1.3; there is no unencrypted "h3c" equivalent.
func generateTLSConfig(alpn string) (*tls.Config, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		DNSNames:     []string{"localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{certDER},
			PrivateKey:  priv,
		}},
		NextProtos: []string{alpn},
	}, nil
}

// runEchoServer starts a minimal QUIC listener that accepts one connection and
// echoes back whatever arrives on its first bidirectional stream.
func runEchoServer(ctx context.Context, listener *quic.Listener) error {
	conn, err := listener.Accept(ctx)
	if err != nil {
		return fmt.Errorf("accepting connection: %w", err)
	}

	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		return fmt.Errorf("accepting stream: %w", err)
	}

	data, err := io.ReadAll(stream)
	if err != nil {
		return fmt.Errorf("reading stream: %w", err)
	}
	fmt.Printf("[Echo Server] Received on stream %d: %q\n", stream.StreamID(), string(data))

	if _, err := stream.Write(append([]byte("echo: "), data...)); err != nil {
		return err
	}
	// Close the stream's write-side to send a FIN frame, then wait for the client to close the connection.
	if err := stream.Close(); err != nil {
		return err
	}
	<-conn.Context().Done()
	return nil
}

// runEchoClient dials the QUIC echo server, opens a bidirectional stream, sends
// a message, and reads back the echoed response.
func runEchoClient(ctx context.Context, addr string) error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo"},
	}
	conn, err := quic.DialAddr(ctx, addr, tlsConf, nil)
	if err != nil {
		return fmt.Errorf("dialing quic: %w", err)
	}
	defer conn.CloseWithError(0, "done")

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return fmt.Errorf("opening stream: %w", err)
	}

	msg := "Hello over QUIC!"
	fmt.Printf("[Echo Client] Sending on stream %d: %q\n", stream.StreamID(), msg)
	if _, err := stream.Write([]byte(msg)); err != nil {
		return err
	}
	// Closing the write side sends a FIN bit on the stream so io.ReadAll on the server finishes,
	// while still allowing us to read the server's response on the same stream!
	stream.Close()

	reply, err := io.ReadAll(stream)
	if err != nil {
		return err
	}
	fmt.Printf("[Echo Client] Got reply: %q\n", string(reply))
	return nil
}
