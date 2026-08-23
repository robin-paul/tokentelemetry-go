package api

import (
	"fmt"
	"net/http"
)

// Healthz returns the health check response.
func (s *Server) Healthz(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": s.cfg.Version,
	})
}

// Root returns API running status.
func (s *Server) Root(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "TokenTelemetry API is running",
	})
}

// Version returns version, commit hash, and update information.
func (s *Server) Version(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"current":        s.cfg.Commit,
		"latest":         s.cfg.Version,
		"behind":         false,
		"releases":       []interface{}{},
		"latest_release": fmt.Sprintf("v%s", s.cfg.Version),
		"release_url":    "https://github.com/robin-paul/token-analyzer",
		"source":         "github",
		"repo":           "robin-paul/token-analyzer",
	})
}

// Agents enumerates supported/detected coding agents.
func (s *Server) Agents(w http.ResponseWriter, r *http.Request) {
	agents := []string{
		"claude", "codex", "gemini", "antigravity", "qwen",
		"cursor", "copilot", "opencode", "grok",
		"pi", "cline", "muse", "prime", "dsh", "smallcode",
		"windsurf", "vibe", "ollama",
	}
	respondJSON(w, http.StatusOK, agents)
}

// RemoteAccess returns pairing information for loopback clients only.
func (s *Server) RemoteAccess(w http.ResponseWriter, r *http.Request) {
	if !isLoopback(r.RemoteAddr) {
		respondError(w, http.StatusForbidden, "Remote access endpoint is only accessible via loopback")
		return
	}

	if s.cfg.AuthToken == "" {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"enabled": true,
		"url":     fmt.Sprintf("http://localhost:8000/?token=%s", s.cfg.AuthToken),
		"token":   s.cfg.AuthToken,
	})
}
