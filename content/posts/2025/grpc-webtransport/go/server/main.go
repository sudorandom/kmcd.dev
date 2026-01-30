package main

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net/http"

	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

func main() {
	cert, err := tls.LoadX509KeyPair("localhost+1.pem", "localhost+1-key.pem")
	if err != nil {
		log.Fatal(err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	mux := http.NewServeMux()

	server := &webtransport.Server{
		H3: http3.Server{
			Addr:      ":4433",
			Handler:   mux,
			TLSConfig: tlsConfig,
		},
		ReorderingTimeout: 0,
		CheckOrigin: func(r *http.Request) bool {
			log.Printf("checking origin: %s", r.Header.Get("Origin"))
			return true
		},
	}

	mux.HandleFunc("/webtransport", func(rw http.ResponseWriter, r *http.Request) {
		conn, err := server.Upgrade(rw, r)
		if err != nil {
			log.Printf("upgrading failed: %s", err)
			rw.WriteHeader(500)
			return
		}

		go func() {
			log.Printf("accepted session: %s", conn.RemoteAddr())
			for {
				stream, err := conn.AcceptStream(r.Context())
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						log.Printf("accepting stream failed: %s", err)
					}
					return
				}
				log.Printf("accepted stream: %d", stream.StreamID())

				go func() {
					for {
						buf := make([]byte, 1024)
						n, err := stream.Read(buf)
						if err != nil {
							log.Printf("read finished with error: %s", err)
							return
						}
						log.Printf("read %d bytes: %s", n, buf[:n])

						_, err = stream.Write(buf[:n])
						if err != nil {
							log.Printf("write finished with error: %s", err)
							return
						}
						log.Printf("wrote %d bytes: %s", n, buf[:n])
					}
				}()
			}
		}()
	})

	mux.Handle("/", http.FileServer(http.Dir("client")))

	log.Println("Starting server on :4433")

	// Start the HTTP/3 server
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// Start the HTTP/2 server
	h2Srv := &http.Server{
		Addr:      ":4433",
		Handler:   mux,
		TLSConfig: tlsConfig,
	}
	log.Println("Starting HTTP/2 server on :4433")
	if err := h2Srv.ListenAndServeTLS("", ""); err != nil {
		log.Fatal(err)
	}
}
