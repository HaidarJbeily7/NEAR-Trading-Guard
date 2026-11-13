// Package httpapi exposes live Intents previews and the position-adapter boundary.
package httpapi

import (
	"encoding/json"
	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/intents"
	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/web"
	"io"
	"net/http"
)

func Handler(client intents.Client) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", web.Handler())
	mux.HandleFunc("/api/positions", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, http.MethodGet) {
			return
		}
		write(w, map[string]any{"connected": false, "positions": []any{}, "message": "No lending protocol adapter is configured. Connect a supported account to load live positions."})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, http.MethodGet) {
			return
		}
		write(w, map[string]any{"status": "ok", "intents_network": "mainnet", "lending_mode": "unconfigured", "server_signing": false})
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
			http.Error(w, "invalid request", 400)
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
