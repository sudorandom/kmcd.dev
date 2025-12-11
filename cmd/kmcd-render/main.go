package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"

	"github.com/playwright-community/playwright-go"
)

// setts from the assets directory can be referenced
const d2WorkingDirectory = "assets"

func handleSVGToPNG(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Println("INFO: Received request for /svg-to-png")
	svgData, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: Could not read request body: %v", err)
		http.Error(w, "Could not read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	pw, err := playwright.Run()
	if err != nil {
		log.Printf("ERROR: Could not start playwright: %v", err)
		http.Error(w, "Could not start playwright", http.StatusInternalServerError)
		return
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		log.Printf("ERROR: Could not launch browser: %v", err)
		http.Error(w, "Could not launch browser", http.StatusInternalServerError)
		return
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		log.Printf("ERROR: Could not create page: %v", err)
		http.Error(w, "Could not create page", http.StatusInternalServerError)
		return
	}

	if err = page.SetContent(string(svgData)); err != nil {
		log.Printf("ERROR: Could not set page content: %v", err)
		http.Error(w, "Could not set page content", http.StatusInternalServerError)
		return
	}

	svgElement, err := page.QuerySelector("svg")
	if err != nil {
		log.Printf("ERROR: Could not find SVG element on page: %v", err)
		http.Error(w, "Could not find SVG element", http.StatusInternalServerError)
		return
	}

	screenshotBytes, err := svgElement.Screenshot(playwright.ElementHandleScreenshotOptions{
		Type: playwright.ScreenshotTypePng,
	})
	if err != nil {
		log.Printf("ERROR: Could not take screenshot: %v", err)
		http.Error(w, "Could not take screenshot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(screenshotBytes)
}

func handleRenderRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Println("INFO: Received request for /render-d2")
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Could not read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	output, err := renderD2(string(requestBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write([]byte(output))
}

func renderD2(content string) (string, error) {
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
	http.HandleFunc("GET /render", handleRenderRequest)
	http.HandleFunc("POST /render", handleRenderRequest)
	http.HandleFunc("POST /svg-to-png", handleSVGToPNG)
	log.Printf("Starting server on: http://%s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("http: %s", err)
	}
}
