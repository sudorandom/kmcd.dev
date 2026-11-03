package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Server implements an HTTP/3 server on top of our QUIC Conn/Listener abstraction.
type Server struct {
	Addr      string
	Handler   http.Handler
	TLSConfig *tls.Config
}

// ListenAndServeQUIC starts listening on UDP for QUIC connections and serves HTTP/3.
func (s *Server) ListenAndServeQUIC(ctx context.Context) error {
	tlsConf := s.TLSConfig
	if tlsConf == nil {
		var err error
		tlsConf, err = generateTLSConfig("h3")
		if err != nil {
			return err
		}
	}
	listener, err := ListenQUIC(s.Addr, tlsConf)
	if err != nil {
		return err
	}
	defer listener.Close()
	return s.ServeListener(ctx, listener)
}

// ServeListener accepts connections from any Listener implementation (quic-go or our own QUIC!).
func (s *Server) ServeListener(ctx context.Context, listener Listener) error {
	for {
		conn, err := listener.Accept(ctx)
		if err != nil {
			return err
		}
		go func(c Conn) {
			if err := s.handleConnection(ctx, c); err != nil && !errors.Is(err, context.Canceled) {
				slog.Debug("connection closed", "err", err)
			}
		}(conn)
	}
}

func (s *Server) handleConnection(ctx context.Context, conn Conn) error {
	// 1. Open the mandatory server-initiated unidirectional Control Stream (0x00)
	// and immediately send our SETTINGS frame (RFC 9114 Section 6.2.1 & 7.2.4).
	if err := sendControlStreamSettings(ctx, conn); err != nil {
		return fmt.Errorf("sending server settings: %w", err)
	}

	// 2. Drain incoming client-initiated unidirectional streams (Control, QPACK, Grease)
	// in the background so we don't block or violate the spec.
	go drainPeerUniStreams(ctx, conn)

	// 3. Accept bidirectional request streams in a loop!
	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return err
		}
		go func(st Stream) {
			if err := s.handleRequestStream(ctx, conn, st); err != nil {
				slog.Error("http3 stream error", "stream", st.StreamID(), "err", err)
			}
		}(stream)
	}
}

// sendControlStreamSettings opens a unidirectional stream, writes stream type 0x00
// (Control Stream), and writes a SETTINGS frame (0x04) advertising QPACK static-only mode.
func sendControlStreamSettings(ctx context.Context, conn Conn) error {
	uni, err := conn.OpenUniStreamSync(ctx)
	if err != nil {
		return err
	}
	// Do NOT close the control stream! RFC 9114 Section 6.2.1 states that closing
	// the control stream MUST be treated as a connection error of type H3_CLOSED_CRITICAL_STREAM.
	var buf []byte
	buf = AppendVarint(buf, StreamTypeControl)

	// Build SETTINGS payload: pairs of (SettingID, Value) varints
	var settingsPayload []byte
	settingsPayload = AppendVarint(settingsPayload, SettingsQPACKMaxTableCapacity)
	settingsPayload = AppendVarint(settingsPayload, 0)
	settingsPayload = AppendVarint(settingsPayload, SettingsQPACKBlockedStreams)
	settingsPayload = AppendVarint(settingsPayload, 0)

	buf = AppendVarint(buf, FrameSettings)
	buf = AppendVarint(buf, uint64(len(settingsPayload)))
	buf = append(buf, settingsPayload...)

	_, err = uni.Write(buf)
	return err
}

// drainPeerUniStreams accepts incoming unidirectional streams from the peer, reads their
// stream type varint, and parses control frames (like SETTINGS).
func drainPeerUniStreams(ctx context.Context, conn Conn) {
	for {
		uni, err := conn.AcceptUniStream(ctx)
		if err != nil {
			return
		}
		go func(r ReceiveStream) {
			streamType, err := ReadVarint(r)
			if err != nil {
				return
			}
			if streamType == StreamTypeControl {
				// Read the peer's initial SETTINGS frame
				frame, err := ReadFrame(r)
				if err == nil && frame.Type == FrameSettings {
					slog.Debug("received peer HTTP/3 SETTINGS", "bytes", len(frame.Payload))
				}
			}
			// Keep draining any future frames on this unidirectional stream
			_, _ = io.Copy(io.Discard, r)
		}(uni)
	}
}

func (s *Server) handleRequestStream(ctx context.Context, conn Conn, stream Stream) error {
	defer stream.Close()

	// The first frame on a request stream MUST be a HEADERS frame (RFC 9114 Section 4.1).
	firstFrame, err := ReadFrame(stream)
	if err != nil {
		return fmt.Errorf("reading initial request frame: %w", err)
	}
	if firstFrame.Type != FrameHeaders {
		return fmt.Errorf("expected HEADERS frame (0x01), got 0x%x", firstFrame.Type)
	}

	dec := NewQPACKDecoder()
	headerFields, err := dec.Decode(firstFrame.Payload)
	if err != nil {
		return fmt.Errorf("qpack decode error: %w", err)
	}

	req := &http.Request{
		Proto:      "HTTP/3.0",
		ProtoMajor: 3,
		ProtoMinor: 0,
		Header:     make(http.Header),
		Body:       &h3BodyReader{stream: stream},
		RemoteAddr: conn.RemoteAddr().String(),
	}

	var scheme, authority, path string
	for _, hf := range headerFields {
		switch hf.Name {
		case ":method":
			req.Method = hf.Value
		case ":scheme":
			scheme = hf.Value
		case ":authority":
			authority = hf.Value
			req.Host = hf.Value
		case ":path":
			path = hf.Value
			req.RequestURI = hf.Value
		default:
			req.Header.Add(hf.Name, hf.Value)
		}
	}
	if cl := req.Header.Get("Content-Length"); cl != "" {
		if n, err := strconv.ParseInt(cl, 10, 64); err == nil {
			req.ContentLength = n
		}
	}
	if u, err := url.Parse(fmt.Sprintf("%s://%s%s", scheme, authority, path)); err == nil {
		req.URL = u
	}

	w := &h3ResponseWriter{
		stream:  stream,
		headers: make(http.Header),
	}

	handler := s.Handler
	if handler == nil {
		handler = http.DefaultServeMux
	}
	handler.ServeHTTP(w, req.WithContext(ctx))

	if !w.sentHeaders {
		w.WriteHeader(http.StatusOK)
	}
	return nil
}

// h3BodyReader streams DATA frame payloads on demand when the handler reads req.Body
// or the client reads resp.Body.
type h3BodyReader struct {
	stream io.Reader
	buf    []byte
	closer io.Closer
}

func (b *h3BodyReader) Read(p []byte) (int, error) {
	for len(b.buf) == 0 {
		frame, err := ReadFrame(b.stream)
		if err != nil {
			return 0, err // io.EOF when the peer closes the QUIC stream with FIN
		}
		if frame.Type == FrameData {
			b.buf = frame.Payload
		}
		// Ignore unknown/grease frames or trailing HEADERS frames
	}
	n := copy(p, b.buf)
	b.buf = b.buf[n:]
	return n, nil
}

func (b *h3BodyReader) Close() error {
	if b.closer != nil {
		return b.closer.Close()
	}
	return nil
}

// h3ResponseWriter implements http.ResponseWriter over an HTTP/3 request stream.
type h3ResponseWriter struct {
	stream      io.Writer
	headers     http.Header
	sentHeaders bool
}

func (w *h3ResponseWriter) Header() http.Header {
	return w.headers
}

func (w *h3ResponseWriter) WriteHeader(statusCode int) {
	if w.sentHeaders {
		return
	}
	w.sentHeaders = true

	hfs := []HeaderField{
		{Name: ":status", Value: strconv.Itoa(statusCode)},
	}
	for k, vals := range w.headers {
		for _, v := range vals {
			hfs = append(hfs, HeaderField{Name: strings.ToLower(k), Value: v})
		}
	}
	enc := NewQPACKEncoder()
	_ = WriteFrame(w.stream, FrameHeaders, enc.Encode(hfs))
}

func (w *h3ResponseWriter) Write(p []byte) (int, error) {
	if !w.sentHeaders {
		if w.headers.Get("Content-Type") == "" {
			w.headers.Set("Content-Type", http.DetectContentType(p))
		}
		w.WriteHeader(http.StatusOK)
	}
	if len(p) == 0 {
		return 0, nil
	}
	if err := WriteFrame(w.stream, FrameData, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

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
