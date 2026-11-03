package main

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestVarintRoundTrip(t *testing.T) {
	values := []uint64{0, 1, 63, 64, 16383, 16384, 1073741823, 1073741824, 4611686018427387903}
	for _, want := range values {
		buf := AppendVarint(nil, want)
		got, err := ReadVarint(bytes.NewReader(buf))
		if err != nil {
			t.Fatalf("ReadVarint(%d) error: %v", want, err)
		}
		if got != want {
			t.Fatalf("ReadVarint(%d) = %d, want %d", want, got, want)
		}
	}
}

func TestRequestResponseRoundTrip(t *testing.T) {
	var wire bytes.Buffer
	req, err := http.NewRequest("POST", "https://kmcd.dev/api/echo", strings.NewReader("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Custom-Header", "hello-h3")

	if err := WriteRequest(&wire, req); err != nil {
		t.Fatal(err)
	}

	parsedReq, err := ReadRequest(&wire)
	if err != nil {
		t.Fatal(err)
	}
	if parsedReq.Method != "POST" || parsedReq.URL.Path != "/api/echo" || parsedReq.Host != "kmcd.dev" {
		t.Fatalf("unexpected request metadata: %+v", parsedReq)
	}
	if got := parsedReq.Header.Get("X-Custom-Header"); got != "hello-h3" {
		t.Fatalf("got X-Custom-Header %q, want hello-h3", got)
	}
	body, _ := io.ReadAll(parsedReq.Body)
	if string(body) != "hello world" {
		t.Fatalf("got body %q, want hello world", string(body))
	}
}
