// Package httpapi exposes previews and read-only swap tracking. Deposit-address
// preparation is deliberately available only through the local CLI.
package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/intents"
	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/positions"
	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/risk"
	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/web"
)

func Handler(client intents.Client) http.Handler {
	mux := http.NewServeMux()
	store := positions.New()
	mux.Handle("/", web.Handler())
	mux.HandleFunc("/api/positions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			write(w, store.Snapshot())
		case http.MethodPost:
			var input positions.Create
			if !decode(w, r, &input) {
				return
			}
			result, err := store.Add(input)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			write(w, result)
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", 405)
		}
	})
	mux.HandleFunc("/api/positions/action", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, http.MethodPost) {
			return
		}
		var input positions.Action
		if !decode(w, r, &input) {
			return
		}
		result, err := store.Apply(input)
		if err != nil {
			code := 400
			if errors.Is(err, positions.ErrConflict) {
				code = 409
			}
			http.Error(w, err.Error(), code)
			return
		}
		write(w, result)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, http.MethodGet) {
			return
		}
		write(w, map[string]any{"status": "ok", "intents_network": "mainnet", "lending_mode": "simulation", "server_signing": false})
	})
	mux.HandleFunc("/api/demo", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, http.MethodGet) {
			return
		}
		plan, err := risk.Evaluate(risk.Position{CollateralUSD: 1400, DebtUSD: 1000, LiquidationThreshold: 0.8}, risk.Policy{Trigger: 1.2, Target: 1.5, BudgetUSD: 500})
		if err != nil {
			http.Error(w, "evaluation failed", 500)
			return
		}
		write(w, plan)
	})
	mux.HandleFunc("/api/intents/tokens", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, http.MethodGet) {
			return
		}
		tokens, err := client.Tokens(r.Context())
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		// Only list routes this release can validate.
		filtered := make([]intents.Token, 0)
		for _, token := range tokens {
			if token.Blockchain == "near" {
				filtered = append(filtered, token)
			}
		}
		write(w, filtered)
	})
	mux.HandleFunc("/api/intents/quote", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, http.MethodPost) {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var request intents.Request
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "invalid quote request", 400)
			return
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			http.Error(w, "expected one JSON object", 400)
			return
		}
		result, err := client.Quote(r.Context(), request)
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		write(w, result)
	})
	mux.HandleFunc("/api/intents/status", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, http.MethodGet) {
			return
		}
		result, err := client.Status(r.Context(), r.URL.Query().Get("depositAddress"), r.URL.Query().Get("depositMemo"))
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		write(w, result)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	})
}

func method(w http.ResponseWriter, r *http.Request, want string) bool {
	if r.Method == want {
		return true
	}
	w.Header().Set("Allow", want)
	http.Error(w, "method not allowed", 405)
	return false
}
func write(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		http.Error(w, "invalid request", 400)
		return false
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		http.Error(w, "expected one JSON object", 400)
		return false
	}
	return true
}
