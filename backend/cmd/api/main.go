package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/httpapi"
	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/intents"
)

func main() {
	server := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           httpapi.Handler(intents.Client{JWT: os.Getenv("INTENTS_JWT")}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("API on http://%s; Intents mainnet previews enabled; lending simulated; signing disabled", server.Addr)
	log.Fatal(server.ListenAndServe())
}
