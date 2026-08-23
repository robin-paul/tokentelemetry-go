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

	// Prevent directory traversal outside allowed roots (user home, cwd, temp dirs)
	if !isPathAllowed(cleanPath) {
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

// isPathAllowed checks if the target path resides within an allowed base directory.
func isPathAllowed(targetPath string) bool {
	homeDir, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	tempDir := os.TempDir()

	allowedRoots := []string{
		homeDir,
		cwd,
		tempDir,
		"/tmp",
		"/var/tmp",
		"/private/tmp",
		"/private/var/tmp",
		"/var/folders",
		"/private/var/folders",
	}

	cleanTarget := filepath.Clean(targetPath)
	evalTarget, evalTargetErr := filepath.EvalSymlinks(cleanTarget)

	for _, root := range allowedRoots {
		if root == "" {
			continue
		}
		cleanRoot := filepath.Clean(root)

		if isSubDir(cleanRoot, cleanTarget) {
			return true
		}

		if evalTargetErr == nil {
			evalRoot, evalRootErr := filepath.EvalSymlinks(cleanRoot)
			if evalRootErr == nil && isSubDir(evalRoot, evalTarget) {
				return true
			}
		}
	}
	return false
}

func isSubDir(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}
