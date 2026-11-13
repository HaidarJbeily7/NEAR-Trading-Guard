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

func TestPositionAPIRoundTrip(t *testing.T) {
	h := Handler(intents.Client{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/api/positions/action", strings.NewReader(`{"id":"near-1","action":"crash","revision":0,"drop_percent":25}`)))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Position rescued") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/api/positions/action", strings.NewReader(`{"id":"near-1","action":"crash","revision":0,"drop_percent":25}`)))
	if w.Code != 409 {
		t.Fatalf("stale action accepted: %d", w.Code)
	}
	for _, path := range []string{"/", "/app.js", "/styles.css"} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("asset %s failed: %d", path, w.Code)
		}
		if w.Header().Get("Content-Security-Policy") == "" {
			t.Fatal("missing CSP")
		}
	}
}
