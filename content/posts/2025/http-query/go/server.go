package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func queryHandler(w http.ResponseWriter, r *http.Request) {
	// The new ServeMux handles method checking based on registration.

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// In a real application, you'd parse the query from the body
	// (e.g., JSON) and fetch data accordingly.
	fmt.Fprintf(w, "Received your QUERY request with body: %s\n", string(body))
	log.Printf("Handled QUERY request with body: %s", string(body))
}

func main() {
	mux := http.NewServeMux()
	// Register handler specifically for QUERY method on /data path
	mux.HandleFunc("QUERY /data", queryHandler)

	log.Println("Server starting on port 8080, handling custom QUERY method...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
