package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/intents"
)

func TestLocalRoutes(t *testing.T) {
	h := Handler(intents.Client{})
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/healthz", "", 200},
		{"GET", "/api/demo", "", 200},
		{"POST", "/api/demo", "", 405},
		{"POST", "/api/intents/quote", `{"privateKey":"no"}`, 400},
		{"POST", "/api/intents/quote", `{} {}`, 400},
		{"POST", "/api/intents/prepare", `{}`, 404},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
		if w.Code != tc.status {
			t.Fatalf("%s: got %d want %d", tc.path, w.Code, tc.status)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing cache protection")
		}
	}
}
