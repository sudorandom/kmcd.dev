package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOutOfOrderStreamReassembly(t *testing.T) {
	conn := &toyQUICConn{}
	st := newToyQUICStream(0, conn)

	// Push chunk 2 (offset 6, "world!") BEFORE chunk 1 (offset 0, "hello ")
	// to simulate out-of-order UDP packet arrival!
	st.pushFrame(6, []byte("world!"), true)
	st.pushFrame(0, []byte("hello "), false)

	got, err := io.ReadAll(st)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello world!" {
		t.Fatalf("got %q, want %q", string(got), "hello world!")
	}
}

func TestEndToEndHTTP3OverToyQUIC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		w.Header().Set("X-From-Scratch", "100%")
		_, _ = w.Write([]byte("echo: " + string(b)))
	})

	listener, err := ListenToyQUIC("127.0.0.1:0", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	srv := &Server{Handler: mux}
	go func() {
		_ = srv.ServeListener(ctx, listener)
	}()

	client := NewClient()
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://"+listener.Addr().String()+"/test", strings.NewReader("digging downward"))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "echo: digging downward" {
		t.Fatalf("got %q, want %q", string(body), "echo: digging downward")
	}
	if got := resp.Header.Get("X-From-Scratch"); got != "100%" {
		t.Fatalf("got header %q, want 100%%", got)
	}
}
