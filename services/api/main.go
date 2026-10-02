package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"parental-monitor-cli/services/api/internal/factory"
	"parental-monitor-cli/services/api/internal/handlers"
	"parental-monitor-cli/services/api/internal/storage"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	var h *handlers.Handler

	useFirebase := os.Getenv("USE_FIREBASE") == "true" || os.Getenv("FIREBASE_PROJECT_ID") != ""
	if useFirebase {
		repos, err := factory.NewFirebaseFactory(context.Background(), os.Getenv("FIREBASE_PROJECT_ID"), os.Getenv("FIREBASE_CREDENTIALS_PATH"))
		if err != nil {
			log.Printf("firebase init failed: %v; falling back to memory", err)
			store := storage.NewMemoryStore()
			h = handlers.NewHandler(store)
		} else {
			h = handlers.NewHandlerFactory(repos)
			log.Println("using Firebase-backed repositories")
		}
	} else {
		store := storage.NewMemoryStore()
		h = handlers.NewHandler(store)
		log.Println("using in-memory repositories")
	}

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
