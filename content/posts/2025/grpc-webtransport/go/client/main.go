package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	dialer := &webtransport.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	_, conn, err := dialer.Dial(ctx, "https://localhost:4433/webtransport", nil)
	if err != nil {
		return err
	}
	defer conn.CloseWithError(0, "graceful shutdown")

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return err
	}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-gctx.Done():
				log.Println("shutting down writer")
				return gctx.Err()
			case t := <-ticker.C:
				msg := fmt.Sprintf("Hello! The time is now %v", t.Format(time.DateTime))
				_, err = stream.Write([]byte(msg))
				if err != nil {
					return err
				}
				log.Printf("Wrote: %s", msg)
			}
		}
	})

	g.Go(func() error {
		for {
			buf := make([]byte, 1024)
			n, err := stream.Read(buf)
			if err != nil {
				log.Printf("shutting down reader: %v", err)
				return err
			}
			log.Printf("Read: %s", buf[:n])
		}
	})

	go func() {
		<-gctx.Done()
		stream.CancelRead(0)
		stream.Close()
	}()

	log.Println("Running, press CTRL+C to stop...")
	defer log.Println("shutting down")

	return g.Wait()
}
