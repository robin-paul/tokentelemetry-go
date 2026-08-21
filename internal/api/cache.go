package api

import (
	"context"
	"net/http"

	"github.com/robin-paul/tokentelemetry-go/internal/scanner"
)

// GetCacheStatus handles GET /cache/status.
func (s *Server) GetCacheStatus(w http.ResponseWriter, r *http.Request) {
	overview, _ := s.db.GetStatsOverview(r.Context(), "", "", "", "")
	entries := int64(0)
	if overview != nil {
		entries = overview.TotalSessions
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"cached":     true,
		"age_sec":    0.0,
		"ttl_sec":    30.0,
		"entries":    entries,
		"building":   false,
		"last_error": nil,
	})
}

// InvalidateCache handles POST /cache/invalidate.
func (s *Server) InvalidateCache(w http.ResponseWriter, r *http.Request) {
	if s.scannerEngine != nil {
		go func() {
			roots := scanner.DiscoverDefaultRoots()
			_ = s.scannerEngine.ScanRoots(context.Background(), roots)
		}()
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}
