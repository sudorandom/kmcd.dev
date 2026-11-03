package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Client is a from-scratch HTTP/3 client.
type Client struct {
	TLSConfig *tls.Config
	DialFunc  func(ctx context.Context, addr string, tlsConf *tls.Config) (Conn, error)
}

func NewClient() *Client {
	return &Client{
		TLSConfig: &tls.Config{
			NextProtos: []string{"h3"},
		},
		DialFunc: DialQUIC,
	}
}

// Do sends an HTTP/3 request and returns an *http.Response.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	port := req.URL.Port()
	if port == "" {
		port = "443"
	}
	addr := req.URL.Hostname() + ":" + port

	dial := c.DialFunc
	if dial == nil {
		dial = DialQUIC
	}
	conn, err := dial(ctx, addr, c.TLSConfig)
	if err != nil {
		return nil, fmt.Errorf("dialing QUIC %s: %w", addr, err)
	}

	// 1. Open our client Control Stream (0x00) and send initial SETTINGS
	if err := sendControlStreamSettings(ctx, conn); err != nil {
		conn.CloseWithError(0, "settings error")
		return nil, fmt.Errorf("sending client settings: %w", err)
	}
	go drainPeerUniStreams(ctx, conn)

	// 2. Open a bidirectional request stream
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		conn.CloseWithError(0, "stream error")
		return nil, fmt.Errorf("opening request stream: %w", err)
	}

	// 3. Encode and send HEADERS frame
	authority := req.URL.Host
	if authority == "" {
		authority = req.Host
	}
	scheme := req.URL.Scheme
	if scheme == "" {
		scheme = "https"
	}
	path := req.URL.RequestURI()
	if path == "" {
		path = "/"
	}

	headers := []HeaderField{
		{Name: ":method", Value: req.Method},
		{Name: ":scheme", Value: scheme},
		{Name: ":authority", Value: authority},
		{Name: ":path", Value: path},
	}
	for k, vals := range req.Header {
		for _, v := range vals {
			headers = append(headers, HeaderField{Name: strings.ToLower(k), Value: v})
		}
	}

	enc := NewQPACKEncoder()
	if err := WriteFrame(stream, FrameHeaders, enc.Encode(headers)); err != nil {
		conn.CloseWithError(0, "write headers error")
		return nil, err
	}

	// 4. Send DATA frame if request has a body, then close write-side of the stream (FIN)
	if req.Body != nil {
		defer req.Body.Close()
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			conn.CloseWithError(0, "read body error")
			return nil, err
		}
		if len(bodyBytes) > 0 {
			if err := WriteFrame(stream, FrameData, bodyBytes); err != nil {
				conn.CloseWithError(0, "write body error")
				return nil, err
			}
		}
	}
	if err := stream.Close(); err != nil {
		return nil, err
	}

	// 5. Read the response HEADERS frame
	var respHeaderFields []HeaderField
	for {
		frame, err := ReadFrame(stream)
		if err != nil {
			conn.CloseWithError(0, "read response error")
			return nil, fmt.Errorf("reading response headers: %w", err)
		}
		if frame.Type == FrameHeaders {
			dec := NewQPACKDecoder()
			respHeaderFields, err = dec.Decode(frame.Payload)
			if err != nil {
				conn.CloseWithError(0, "qpack error")
				return nil, err
			}
			break
		}
	}

	resp := &http.Response{
		StatusCode: 200,
		Proto:      "HTTP/3.0",
		ProtoMajor: 3,
		ProtoMinor: 0,
		Header:     make(http.Header),
		Request:    req,
		Body: &h3BodyReader{
			stream: stream,
			closer: connCloserFunc(func() error {
				return conn.CloseWithError(0, "done")
			}),
		},
	}

	for _, hf := range respHeaderFields {
		if hf.Name == ":status" {
			resp.StatusCode, _ = strconv.Atoi(hf.Value)
		} else {
			resp.Header.Add(hf.Name, hf.Value)
		}
	}
	resp.Status = fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	return resp, nil
}

type connCloserFunc func() error

func (f connCloserFunc) Close() error { return f() }
