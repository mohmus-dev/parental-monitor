package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"parental-monitor-cli/services/api/internal/handlers"
	"parental-monitor-cli/services/api/internal/storage"
)

func main() {
	store := storage.NewMemoryStore()
	h := handlers.NewHandler(store)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      h,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	log.Printf("API running on http://localhost:%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
