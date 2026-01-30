package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var nlcf = []byte{0x0d, 0x0a}

// Server is a simple HTTP/1.1 server.
type Server struct {
	Addr    string
	Handler http.Handler
}

// ListenAndServe starts the server.
func (s *Server) ListenAndServe() error {
	handler := s.Handler
	if handler == nil {
		handler = http.DefaultServeMux
	}
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go func() {
			if err := s.handleConnection(conn); err != nil {
				slog.Error(fmt.Sprintf("http error: %s", err))
			}
		}()
	}
}

func (s *Server) handleConnection(conn net.Conn) error {
	defer conn.Close()
	for {
		shouldClose, err := s.handleRequest(conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if shouldClose {
			return nil
		}
	}
}

func (s *Server) handleRequest(conn net.Conn) (bool, error) {
	// Limit headers to 1MB
	limitReader := io.LimitReader(conn, 1*1024*1024).(*io.LimitedReader)
	reader := bufio.NewReader(limitReader)

	reqLineBytes, _, err := reader.ReadLine()
	if err != nil {
		return true, fmt.Errorf("read request line error: %w", err)
	}
	reqLine := string(reqLineBytes)

	req := new(http.Request)
	var found bool

	req.Method, reqLine, found = strings.Cut(reqLine, " ")
	if !found {
		return true, errors.New("invalid method")
	}
	if !methodValid(req.Method) {
		return true, errors.New("invalid method")
	}

	req.RequestURI, reqLine, found = strings.Cut(reqLine, " ")
	if !found {
		return true, errors.New("invalid path")
	}
	if req.URL, err = url.ParseRequestURI(req.RequestURI); err != nil {
		return true, fmt.Errorf("invalid path: %w", err)
	}

	req.Proto = reqLine
	req.ProtoMajor, req.ProtoMinor, found = parseProtocol(req.Proto)
	if !found {
		return true, errors.New("invalid protocol")
	}

	req.Header = make(http.Header)
	for {
		line, _, err := reader.ReadLine()
		if err != nil && err != io.EOF {
			return true, err
		} else if err != nil {
			break
		}
		if len(line) == 0 {
			break
		}

		k, v, ok := bytes.Cut(line, []byte{':'})
		if !ok {
			return true, errors.New("invalid header")
		}
		req.Header.Add(strings.ToLower(string(k)), strings.TrimLeft(string(v), " "))
	}

	if _, ok := req.Header["Host"]; !ok {
		return true, errors.New("required 'Host' header not found")
	}

	switch strings.ToLower(req.Header.Get("Connection")) {
	case "keep-alive", "":
		req.Close = false
	case "close":
		req.Close = true
	}

	limitReader.N = math.MaxInt64

	ctx := context.Background()
	ctx = context.WithValue(ctx, http.LocalAddrContextKey, conn.LocalAddr())
	ctx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()
	contentLength, err := parseContentLength(req.Header.Get("Content-Length"))
	if err != nil {
		return true, err
	}
	req.ContentLength = contentLength
	isChunked := req.Header.Get("Transfer-Encoding") == "chunked"
	if req.ContentLength == 0 && !isChunked {
		req.Body = noBody{}
	} else {
		if isChunked {
			req.Body = &chunkedBodyReader{
				reader: reader,
			}
		} else {
			req.Body = &bodyReader{
				reader: io.LimitReader(reader, req.ContentLength),
			}
		}
	}

	req.RemoteAddr = conn.RemoteAddr().String()

	w := &responseBodyWriter{
		req:     req,
		conn:    conn,
		headers: make(http.Header),
	}

	s.Handler.ServeHTTP(w, req.WithContext(ctx))
	if err := w.flush(); err != nil {
		return true, nil
	}
	return req.Close, nil
}

type noBody struct{}

func (noBody) Read([]byte) (int, error) { return 0, io.EOF }
func (noBody) Close() error             { return nil }

func parseContentLength(headerval string) (int64, error) {
	if headerval == "" {
		return 0, nil
	}
	return strconv.ParseInt(headerval, 10, 64)
}

func parseProtocol(proto string) (int, int, bool) {
	switch proto {
	case "HTTP/1.0":
		return 1, 0, true
	case "HTTP/1.1":
		return 1, 1, true
	}
	return 0, 0, false
}

func methodValid(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
		return true
	}
	return false
}

type bodyReader struct {
	reader io.Reader
}

func (r *bodyReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r *bodyReader) Close() error {
	_, err := io.Copy(io.Discard, r.reader)
	return err
}

type chunkedBodyReader struct {
	reader *bufio.Reader
	n      int64 // bytes left in current chunk
	err    error
}

func (r *chunkedBodyReader) Read(p []byte) (n int, err error) {
	if r.err != nil {
		return 0, r.err
	}
	if r.n == 0 {
		r.n, r.err = r.readChunkSize()
		if r.err != nil {
			return 0, r.err
		}
	}
	if r.n == 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > r.n {
		p = p[0:r.n]
	}
	n, err = r.reader.Read(p)
	r.n -= int64(n)
	if r.n == 0 && err == nil {
		// Read trailing \r\n
		b, err := r.reader.ReadByte()
		if err != nil {
			r.err = err
			return n, err
		}
		if b != '\r' {
			r.err = errors.New("missing \r after chunk")
			return n, r.err
		}
		b, err = r.reader.ReadByte()
		if err != nil {
			r.err = err
			return n, err
		}
		if b != '\n' {
			r.err = errors.New("missing \n after chunk")
			return n, r.err
		}
	}
	r.err = err
	return n, err
}

func (r *chunkedBodyReader) readChunkSize() (int64, error) {
	line, err := r.readLine()
	if err != nil {
		return 0, err
	}
	// chunkSize is hex
	n, err := strconv.ParseInt(strings.TrimSpace(string(line)), 16, 64)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		// Read trailers
		for {
			line, err := r.readLine()
			if err != nil {
				return 0, err
			}
			if len(line) == 0 {
				break
			}
		}
	}
	return n, nil
}

func (r *chunkedBodyReader) readLine() (string, error) {
	var line []byte
	for {
		b, err := r.reader.ReadByte()
		if err != nil {
			return "", err
		}
		if b == '\n' {
			break
		}
		line = append(line, b)
	}
	return strings.TrimRight(string(line), "\r"), nil
}

func (r *chunkedBodyReader) Close() error {
	_, err := io.Copy(io.Discard, r)
	return err
}

type responseBodyWriter struct {
	req             *http.Request
	conn            net.Conn
	sentHeaders     bool
	headers         http.Header
	chunkedEncoding bool
	bodyBuffer      *bytes.Buffer
}

func (r *responseBodyWriter) Header() http.Header {
	return r.headers
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	if !r.sentHeaders {
		if r.headers.Get("Content-Type") == "" {
			r.headers.Set("Content-Type", http.DetectContentType(b))
		}
		r.WriteHeader(http.StatusOK)
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
	if !r.sentHeaders {
		r.WriteHeader(http.StatusOK)
	}
	if flusher, ok := r.conn.(interface{ Flush() error }); ok {
		flusher.Flush()
	}
}

func (r *responseBodyWriter) flush() error {
	if r.chunkedEncoding {
		if _, err := r.conn.Write([]byte("0\r\n\r\n")); err != nil {
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
	if r.bodyBuffer != nil {
		_, err := r.conn.Write(r.bodyBuffer.Bytes())
		if err != nil {
			slog.Error("Error writing buffered body", "err", err)
		}
		r.bodyBuffer = nil
	}
}

func (r *responseBodyWriter) writeHeader(conn io.Writer, proto string, headers http.Header, statusCode int) error {
	_, clSet := r.headers["Content-Length"]
	_, teSet := r.headers["Transfer-Encoding"]
	if !clSet && !teSet {
		r.chunkedEncoding = true
		r.headers.Set("Transfer-Encoding", "chunked")
	}

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

func main() {
	addr := "127.0.0.1:9000"
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(".")))
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		b, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		w.Write(b)
	})
	mux.HandleFunc("/echo/chunked", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		io.Copy(w, r.Body)
	})
	mux.HandleFunc("/status/{status}", func(w http.ResponseWriter, r *http.Request) {
		status, err := strconv.ParseInt(r.PathValue("status"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, fmt.Sprintf("error: %s", err))
			return
		}
		w.WriteHeader(int(status))
	})
	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		json.NewEncoder(w).Encode(r.Header)
	})
	mux.HandleFunc("/nothing", func(w http.ResponseWriter, r *http.Request) {})
	s := Server{
		Addr:    addr,
		Handler: mux,
	}
	log.Printf("Starting web server: http://%s", addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
