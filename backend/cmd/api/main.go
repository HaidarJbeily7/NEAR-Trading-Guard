package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/risk"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "mode": "simulation"})
	})
	mux.HandleFunc("/api/demo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", 405)
			return
		}
		plan, err := risk.Evaluate(risk.Position{CollateralUSD: 1400, DebtUSD: 1000, LiquidationThreshold: 0.8}, risk.Policy{Trigger: 1.2, Target: 1.5, BudgetUSD: 500})
		if err != nil {
			http.Error(w, "evaluation failed", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(plan)
	})
	server := &http.Server{Addr: "127.0.0.1:8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("simulation API listening on http://%s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
