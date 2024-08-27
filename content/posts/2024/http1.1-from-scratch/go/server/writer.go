package main

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
)

var nlcf = []byte{'\r', '\n'}

type responseBodyWriter struct {
	req             *http.Request
	conn            net.Conn
	sentHeaders     bool
	headers         http.Header
	chunkedEncoding bool
	bodyBuffer      *bytes.Buffer // Buffer to store the request body
}

func (r *responseBodyWriter) Header() http.Header {
	return r.headers
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	if !r.sentHeaders {
		// Buffer the body until headers are sent
		if r.bodyBuffer == nil {
			r.bodyBuffer = &bytes.Buffer{}
		}
		r.bodyBuffer.Write(b)
		return len(b), nil
	}

	if r.chunkedEncoding {
		chunkSize := fmt.Sprintf("%x\r\n", len(b))
		if _, err := r.conn.Write([]byte(chunkSize)); err != nil {
			return 0, err
		}
	}

	n, err := r.conn.Write(b)
	if err != nil {
		return n, err
	}

	if r.chunkedEncoding {
		if _, err := r.conn.Write(nlcf); err != nil {
			return n, err
		}
	}

	return n, nil
}

func (r *responseBodyWriter) Flush() {
	r.chunkedEncoding = true
	r.flush()
}

func (r *responseBodyWriter) flush() error {
	if !r.sentHeaders {
		if err := r.writeHeader(r.conn, r.req.Proto, r.headers, http.StatusOK); err != nil {
			return err
		}
		r.sentHeaders = true
	}

	if r.chunkedEncoding {
		// Write the final "0" chunk to signal the end of the chunked response
		if _, err := r.conn.Write([]byte("\r\n\r\n")); err != nil {
			return err
		}
	}

	r.writeBufferedBody()

	return nil
}

func (r *responseBodyWriter) WriteHeader(statusCode int) {
	if r.sentHeaders {
		slog.Warn(fmt.Sprintf("WriteHeader called twice, second time with: %d", statusCode))
		return
	}

	r.writeHeader(r.conn, r.req.Proto, r.headers, statusCode)
	r.sentHeaders = true
	r.writeBufferedBody()
}

func (r *responseBodyWriter) writeBufferedBody() {
	// If body was buffered, write it now
	if r.bodyBuffer != nil {
		_, err := r.conn.Write(r.bodyBuffer.Bytes())
		if err != nil {
			slog.Error("Error writing buffered body", "err", err)
		}
		r.bodyBuffer = nil // Clear the buffer after writing
	}
}

func (r *responseBodyWriter) writeHeader(conn io.Writer, proto string, headers http.Header, statusCode int) error {
	// If not chunked encoding, calculate and set the Content-Length header
	if !r.chunkedEncoding {
		if r.bodyBuffer != nil {
			r.headers.Set("Content-Length", strconv.Itoa(r.bodyBuffer.Len()))
		} else {
			// If no body was written, set Content-Length to 0
			r.headers.Set("Content-Length", "0")
		}
	} else {
		r.headers.Set("Transfer-Encoding", "chunked")
	}

	// Set Connection header based on request's Close field
	if r.req.Close {
		r.headers.Set("Connection", "close")
	} else {
		r.headers.Set("Connection", "keep-alive")
	}

	if _, err := io.WriteString(conn, proto); err != nil {
		return err
	}
	if _, err := conn.Write([]byte{' '}); err != nil {
		return err
	}
	if _, err := io.WriteString(conn, strconv.FormatInt(int64(statusCode), 10)); err != nil {
		return err
	}
	if _, err := conn.Write([]byte{' '}); err != nil {
		return err
	}
	if _, err := io.WriteString(conn, http.StatusText(statusCode)); err != nil {
		return err
	}
	if _, err := conn.Write(nlcf); err != nil {
		return err
	}
	for k, vals := range headers {
		for _, val := range vals {
			if _, err := io.WriteString(conn, k); err != nil {
				return err
			}
			if _, err := conn.Write([]byte{':', ' '}); err != nil {
				return err
			}
			if _, err := io.WriteString(conn, val); err != nil {
				return err
			}
			if _, err := conn.Write(nlcf); err != nil {
				return err
			}
		}
	}
	if _, err := conn.Write(nlcf); err != nil {
		return err
	}
	return nil
}
