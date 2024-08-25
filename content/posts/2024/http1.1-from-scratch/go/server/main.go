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
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
)

type Server struct {
	Addr    string
	Handler http.Handler
}

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
	headerReader := textproto.NewReader(bufio.NewReader(reader))

	// Read the request line: GET /path/to/index.html HTTP/1.0
	reqLine, err := headerReader.ReadLine()
	if err != nil {
		return true, fmt.Errorf("read request line error: %w", err)
	}

	req := new(http.Request)
	var found bool

	// Parse Method: GET/POST/PUT/DELETE/etc
	req.Method, reqLine, found = strings.Cut(reqLine, " ")
	if !found {
		return true, errors.New("invalid method")
	}
	if !methodValid(req.Method) {
		return true, errors.New("invalid method")
	}

	// Parse Request URI
	req.RequestURI, reqLine, found = strings.Cut(reqLine, " ")
	if !found {
		return true, errors.New("invalid path")
	}
	if req.URL, err = url.ParseRequestURI(req.RequestURI); err != nil {
		return true, fmt.Errorf("invalid path: %w", err)
	}

	// Parse protocol version "HTTP/1.0"
	req.Proto = reqLine
	req.ProtoMajor, req.ProtoMinor, found = parseProtocol(req.Proto)
	if !found {
		return true, errors.New("invalid protocol")
	}

	// Parse headers
	req.Header = make(http.Header)
	for {
		line, err := headerReader.ReadLineBytes()
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

	// Unbound the limit after we've read the headers since the body can be any size
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
	if req.ContentLength == 0 {
		req.Body = noBody{}
	} else {
		req.Body = &bodyReader{reader: io.LimitReader(reader, req.ContentLength)}
	}

	req.RemoteAddr = conn.RemoteAddr().String()

	w := &responseBodyWriter{
		req:     req,
		conn:    conn,
		headers: make(http.Header),
	}

	// Finally, call our http.Handler!
	s.Handler.ServeHTTP(w, req.WithContext(ctx))
	if err := w.flush(); err != nil {
		return true, nil
	}
	return req.Close, nil
}

type noBody struct{}

func (noBody) Read([]byte) (int, error) { return 0, io.EOF }
func (noBody) Close() error             { return nil }

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

func main() {
	addr := "127.0.0.1:9000"
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("public")))
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		io.Copy(w, r.Body)
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
