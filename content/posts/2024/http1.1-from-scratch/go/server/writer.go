package main

import (
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
}

func (r *responseBodyWriter) Header() http.Header {
	return r.headers
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	// https://stackoverflow.com/questions/26769626/send-a-chunked-http-response-from-a-go-server

	if !r.sentHeaders {
		if err := sendHeaders(r.conn, r.req.Proto, r.headers, http.StatusOK); err != nil {
			return 0, err
		}
		r.sentHeaders = true
	}
	return r.conn.Write(b)
}

func (r *responseBodyWriter) Flush() {
	r.chunkedEncoding = true
	r.flush()
}

func (r *responseBodyWriter) flush() error {
	if !r.sentHeaders {
		if err := sendHeaders(r.conn, r.req.Proto, r.headers, http.StatusOK); err != nil {
			return err
		}
		r.sentHeaders = true
	}
	return nil
}

func (r *responseBodyWriter) WriteHeader(statusCode int) {
	if r.sentHeaders {
		slog.Warn(fmt.Sprintf("WriteHeader called twice, second time with: %d", statusCode))
		return
	}
	if r.chunkedEncoding {
		r.headers.Set("Transfer-Encoding", "chunked")
	}
	sendHeaders(r.conn, r.req.Proto, r.headers, statusCode)
	r.sentHeaders = true
}

func sendHeaders(conn io.Writer, proto string, headers http.Header, statusCode int) error {
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
