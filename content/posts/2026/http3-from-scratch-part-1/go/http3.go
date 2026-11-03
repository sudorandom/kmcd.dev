package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// WriteRequest encodes an *http.Request as an HTTP/3 HEADERS frame followed by
// an optional DATA frame on the given QUIC stream.
func WriteRequest(w io.Writer, req *http.Request) error {
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
	if err := WriteFrame(w, FrameHeaders, enc.Encode(headers)); err != nil {
		return fmt.Errorf("writing request HEADERS frame: %w", err)
	}

	if req.Body != nil {
		defer req.Body.Close()
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("reading request body: %w", err)
		}
		if len(bodyBytes) > 0 {
			if err := WriteFrame(w, FrameData, bodyBytes); err != nil {
				return fmt.Errorf("writing request DATA frame: %w", err)
			}
		}
	}
	return nil
}

// ReadRequest reads HTTP/3 frames from a request stream and constructs an *http.Request.
func ReadRequest(r io.Reader) (*http.Request, error) {
	dec := NewQPACKDecoder()
	var headerFields []HeaderField
	var bodyBuf bytes.Buffer
	gotHeaders := false

	for {
		frame, err := ReadFrame(r)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		switch frame.Type {
		case FrameHeaders:
			if !gotHeaders {
				fields, err := dec.Decode(frame.Payload)
				if err != nil {
					return nil, fmt.Errorf("decoding request headers: %w", err)
				}
				headerFields = fields
				gotHeaders = true
			}
			// A second HEADERS frame after DATA would be HTTP trailers.
			// We are deliberately ignoring trailers here!
		case FrameData:
			bodyBuf.Write(frame.Payload)
		}
	}

	req := &http.Request{
		Proto:      "HTTP/3.0",
		ProtoMajor: 3,
		ProtoMinor: 0,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(bodyBuf.Bytes())),
		ContentLength: int64(bodyBuf.Len()),
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

	u, err := url.Parse(fmt.Sprintf("%s://%s%s", scheme, authority, path))
	if err == nil {
		req.URL = u
	}
	return req, nil
}

// WriteResponse writes an HTTP/3 response (a HEADERS frame followed by a DATA frame) to w.
func WriteResponse(w io.Writer, status int, headers http.Header, body []byte) error {
	hfs := []HeaderField{
		{Name: ":status", Value: strconv.Itoa(status)},
	}
	for k, vals := range headers {
		for _, v := range vals {
			hfs = append(hfs, HeaderField{Name: strings.ToLower(k), Value: v})
		}
	}

	enc := NewQPACKEncoder()
	if err := WriteFrame(w, FrameHeaders, enc.Encode(hfs)); err != nil {
		return fmt.Errorf("writing response HEADERS: %w", err)
	}
	if len(body) > 0 {
		if err := WriteFrame(w, FrameData, body); err != nil {
			return fmt.Errorf("writing response DATA: %w", err)
		}
	}
	return nil
}

// ReadResponse reads HTTP/3 frames from r and returns an *http.Response.
func ReadResponse(r io.Reader) (*http.Response, error) {
	dec := NewQPACKDecoder()
	var headerFields []HeaderField
	var bodyBuf bytes.Buffer
	gotHeaders := false

	for {
		frame, err := ReadFrame(r)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		switch frame.Type {
		case FrameHeaders:
			if !gotHeaders {
				fields, err := dec.Decode(frame.Payload)
				if err != nil {
					return nil, fmt.Errorf("decoding response headers: %w", err)
				}
				headerFields = fields
				gotHeaders = true
			}
		case FrameData:
			bodyBuf.Write(frame.Payload)
		}
	}

	resp := &http.Response{
		StatusCode:    200,
		Proto:         "HTTP/3.0",
		ProtoMajor:    3,
		ProtoMinor:    0,
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(bodyBuf.Bytes())),
		ContentLength: int64(bodyBuf.Len()),
	}

	for _, hf := range headerFields {
		if hf.Name == ":status" {
			resp.StatusCode, _ = strconv.Atoi(hf.Value)
		} else {
			resp.Header.Add(hf.Name, hf.Value)
		}
	}
	resp.Status = fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	return resp, nil
}
