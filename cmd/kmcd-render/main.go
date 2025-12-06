package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
)

// setts from the assets directory can be referenced
const d2WorkingDirectory = "assets"

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

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write([]byte(output))
}

func renderText(content string) (string, error) {
	// d2 -, will render the text received in stdin
	command := exec.Command(
		"d2",
		"--sketch",
		"--theme",
		"201",
		"--pad",
		"20",
		"-",
	)
	command.Stdin = bytes.NewBuffer([]byte(content))
	command.Dir = d2WorkingDirectory
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("d2 execution failed: %s\n%s", err, stderr.String())
		}
		return "", fmt.Errorf("d2 command failed: %w", err)
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
