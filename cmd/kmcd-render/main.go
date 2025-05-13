package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os/exec"
)

func handleRenderRequest(w http.ResponseWriter, r *http.Request) {
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	output, err := renderText(string(requestBody))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(output))
}

func renderText(content string) (string, error) {
	// d2 -, will render the text received in stdin
	command := exec.Command(
		"d2",
		"--sketch",
		"--theme", "201",
		"--pad", "20",
		"-",
	)
	command.Stdin = bytes.NewBuffer([]byte(content))
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func main() {
	const addr = "127.0.0.1:7001"
	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	http.HandleFunc("POST /render", handleRenderRequest)
	log.Printf("Starting server on: http://%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("http: %s", err)
	}
}
