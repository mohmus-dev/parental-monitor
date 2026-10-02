package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			fmt.Println("\n======================================")
			fmt.Println("📡 RECEIVED DATA FROM MONITOR:")
			fmt.Println("======================================")
			// Pretty print
			var pretty interface{}
			json.Unmarshal(body, &pretty)
			formatted, _ := json.MarshalIndent(pretty, "", "  ")
			fmt.Println(string(formatted))
			fmt.Println("======================================")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"status":"ok"}`)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("🖥️  Test server running on http://localhost:8080")
	fmt.Println("Waiting for monitor data...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
