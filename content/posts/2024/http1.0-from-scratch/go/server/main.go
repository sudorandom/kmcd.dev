package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
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

func (s *Server) ServeAndListen() error {
	if s.Handler == nil {
		panic("http server started without a handler")
	}
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
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

	reader := textproto.NewReader(bufio.NewReader(io.LimitReader(conn, 1024*1014*1)))
	reqLine, err := reader.ReadLine()
	if err != nil {
		return fmt.Errorf("read request line error: %w", err)
	}

	req := new(http.Request)
	var found bool
	req.Method, reqLine, found = strings.Cut(reqLine, " ")
	if !found {
		return errors.New("invalid method")
	}
	req.RequestURI, reqLine, found = strings.Cut(reqLine, " ")
	if !found {
		return errors.New("invalid path")
	}
	req.Proto, _, _ = strings.Cut(reqLine, " ")
	if len(req.Proto) == 0 {
		req.Proto = "HTTP/0.9"
	}
	req.ProtoMajor, req.ProtoMinor, found = parseProtocol(req.Proto)
	if !found {
		return errors.New("invalid proto")
	}

	if req.URL, err = url.ParseRequestURI(req.RequestURI); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	req.RemoteAddr = conn.RemoteAddr().String()

	mimeHeader, err := reader.ReadMIMEHeader()
	if err != nil {
		return err
	}
	req.Header = http.Header(mimeHeader)
	// TODO: set up body, if it exists

	s.Handler.ServeHTTP(&responseBodyWriter{
		proto:   req.Proto,
		conn:    conn,
		headers: make(http.Header),
	}, req)
	return nil
}

func parseProtocol(proto string) (int, int, bool) {
	switch proto {
	case "HTTP/0.9":
		return 0, 9, true
	case "HTTP/1.0":
		return 1, 0, true
	case "HTTP/1.1":
		return 1, 1, true
	}
	return 0, 0, false
}

type responseBodyWriter struct {
	proto       string
	conn        net.Conn
	sentHeaders bool
	headers     http.Header
}

func (r *responseBodyWriter) Header() http.Header {
	return r.headers
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	if !r.sentHeaders {
		r.sendHeaders(http.StatusOK)
	}
	return r.conn.Write(b)
}

func (r *responseBodyWriter) WriteHeader(statusCode int) {
	if r.sentHeaders {
		slog.Warn(fmt.Sprintf("WriteHeader called twice, second time with: %d", statusCode))
		return
	}
	r.sentHeaders = true
	r.sendHeaders(statusCode)
}

func (r *responseBodyWriter) sendHeaders(statusCode int) {
	io.WriteString(r.conn, r.proto)
	r.conn.Write([]byte{' '})
	io.WriteString(r.conn, strconv.FormatInt(int64(statusCode), 10))
	r.conn.Write([]byte{' '})
	io.WriteString(r.conn, http.StatusText(statusCode))
	r.conn.Write([]byte{'\r', '\n'})
	for k, vals := range r.headers {
		for _, val := range vals {
			io.WriteString(r.conn, k)
			r.conn.Write([]byte{':'})
			io.WriteString(r.conn, val)
			r.conn.Write([]byte{'\r', '\n'})
		}
	}
	r.conn.Write([]byte{'\r', '\n'})
}

func main() {
	addr := "127.0.0.1:9000"
	s := Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World!"))
		}),
	}
	log.Printf("Listening on %s", addr)
	if err := s.ServeAndListen(); err != nil {
		log.Fatal(err)
	}
}
