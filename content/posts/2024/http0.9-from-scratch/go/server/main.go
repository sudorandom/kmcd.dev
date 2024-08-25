package main

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type Server struct {
	Addr    string
	Handler http.Handler
}

func (s *Server) ListenAndServe() error {
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

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, _, err := reader.ReadLine()
	if err != nil {
		return
	}

	fields := strings.Fields(string(line))
	if len(fields) < 2 {
		return
	}
	r := &http.Request{
		Method:     fields[0],
		URL:        &url.URL{Scheme: "http", Path: fields[1]},
		Proto:      "HTTP/0.9",
		ProtoMajor: 0,
		ProtoMinor: 9,
		RemoteAddr: conn.RemoteAddr().String(),
	}

	s.Handler.ServeHTTP(newWriter(conn), r)
}

type responseBodyWriter struct {
	conn net.Conn
}

func (r *responseBodyWriter) Header() http.Header {
	// unsupported with HTTP/0.9
	return nil
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	return r.conn.Write(b)
}

func (r *responseBodyWriter) WriteHeader(statusCode int) {
	// unsupported with HTTP/0.9
}

func newWriter(c net.Conn) http.ResponseWriter {
	return &responseBodyWriter{
		conn: c,
	}
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
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
