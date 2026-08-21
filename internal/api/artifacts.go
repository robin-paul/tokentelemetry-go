package api

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// GetArtifact handles GET /artifacts and HEAD /artifacts.
func (s *Server) GetArtifact(w http.ResponseWriter, r *http.Request) {
	artifactPath := r.URL.Query().Get("path")
	if artifactPath == "" {
		respondError(w, http.StatusBadRequest, "Missing path query parameter")
		return
	}

	cleanPath := filepath.Clean(artifactPath)
	if !filepath.IsAbs(cleanPath) {
		respondError(w, http.StatusBadRequest, "Artifact path must be absolute")
		return
	}

	// Guardrail: verify file exists
	fi, err := os.Stat(cleanPath)
	if err != nil || fi.IsDir() {
		respondError(w, http.StatusNotFound, "Artifact file not found")
		return
	}

	// Prevent directory traversal outside user home and current working dir
	homeDir, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()

	allowed := false
	if homeDir != "" && strings.HasPrefix(cleanPath, homeDir) {
		allowed = true
	} else if cwd != "" && strings.HasPrefix(cleanPath, cwd) {
		allowed = true
	} else if strings.HasPrefix(cleanPath, "/tmp") || strings.HasPrefix(cleanPath, "/var/tmp") {
		allowed = true
	}

	if !allowed {
		respondError(w, http.StatusForbidden, "Access to artifact path is forbidden")
		return
	}

	// Content-Type detection
	ext := filepath.Ext(cleanPath)
	ctype := mime.TypeByExtension(ext)
	if ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}

	http.ServeFile(w, r, cleanPath)
}
