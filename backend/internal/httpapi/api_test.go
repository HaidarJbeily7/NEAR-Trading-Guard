package httpapi

import (
	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/intents"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLiveBoundary(t *testing.T) {
	h := Handler(intents.Client{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/positions", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"connected":false`) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	for _, path := range []string{"/", "/app.js", "/styles.css"} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("asset %s: %d", path, w.Code)
		}
	}
}
