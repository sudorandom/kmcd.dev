package main

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cert, err := tls.LoadX509KeyPair("localhost.pem", "localhost-key.pem")
	if err != nil {
		return err
	}
	h3TLSConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{http3.NextProtoH3},
	}
	httpsTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2", "http/1.1"},
	}

	mux := http.NewServeMux()

	server := &webtransport.Server{
		H3: &http3.Server{
			Addr:            ":4433",
			Port:            4433,
			Handler:         mux,
			TLSConfig:       h3TLSConfig,
			EnableDatagrams: true,
		},
		ReorderingTimeout: 0,
		CheckOrigin: func(r *http.Request) bool {
			log.Printf("checking origin: %s", r.Header.Get("Origin"))
			return true
		},
	}
	webtransport.ConfigureHTTP3Server(server.H3)

	withH3Headers := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.Header().Add("Alt-Svc", `h3=":4433"; ma=2592000`)
			next.ServeHTTP(rw, r)
		})
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
				log.Printf("accepted stream")

				go func() {
					buf := make([]byte, 1024)
					for {
						n, err := stream.Read(buf)
						if err != nil {
							if !errors.Is(err, io.EOF) {
								log.Printf("read finished with error: %s", err)
							}
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

	browserHandler := http.FileServer(http.Dir("client"))
	mux.Handle("/", browserHandler)

	h2Srv := &http.Server{
		Addr:      ":4433",
		Handler:   withH3Headers(mux),
		TLSConfig: httpsTLSConfig,
	}
	browserSrv := &http.Server{
		Addr:    ":8080",
		Handler: browserHandler,
	}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Println("Starting WebTransport HTTP/3 server on :4433")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		log.Println("Starting HTTPS file server on :4433")
		if err := h2Srv.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		log.Println("Starting HTTP browser client on :8080")
		if err := browserSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-gctx.Done()
		if err := server.Close(); err != nil {
			log.Printf("closing WebTransport server failed: %s", err)
		}
		if err := h2Srv.Shutdown(context.Background()); err != nil {
			log.Printf("shutting down HTTPS server failed: %s", err)
		}
		if err := browserSrv.Shutdown(context.Background()); err != nil {
			log.Printf("shutting down browser server failed: %s", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}
