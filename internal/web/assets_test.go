package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebHandler(t *testing.T) {
	handler := Handler()

	// 1. Root index.html
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for root, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "TokenTelemetry") && !strings.Contains(body, "<!DOCTYPE html>") {
		t.Errorf("unexpected body content for root: %s", body)
	}

	// 2. Static route /analytics/
	req = httptest.NewRequest("GET", "/analytics/", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /analytics/, got %d", w.Code)
	}

	// 3. Dynamic route fallback /sessions/custom-uuid-123
	req = httptest.NewRequest("GET", "/sessions/custom-uuid-123", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /sessions/custom-uuid-123 fallback, got %d", w.Code)
	}

	// 4. Dynamic route fallback /projects/my-repo/subpath
	req = httptest.NewRequest("GET", "/projects/my-repo/subpath", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /projects/my-repo/subpath fallback, got %d", w.Code)
	}
}
