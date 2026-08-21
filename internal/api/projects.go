package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// GetProjects handles GET /api/projects and GET /projects.
func (s *Server) GetProjects(w http.ResponseWriter, r *http.Request) {
	includeHidden := r.URL.Query().Get("include_hidden") == "true"

	projects, err := s.db.GetProjects(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.mu.RLock()
	var filtered []models.ProjectSummary
	for _, p := range projects {
		if !includeHidden && s.hiddenProjects[p.ProjectName] {
			continue
		}
		filtered = append(filtered, p)
	}
	s.mu.RUnlock()

	if filtered == nil {
		filtered = []models.ProjectSummary{}
	}
	respondJSON(w, http.StatusOK, filtered)
}

// GetProjectDetail handles GET /api/projects/{name} and GET /projects/{name}.
func (s *Server) GetProjectDetail(w http.ResponseWriter, r *http.Request) {
	projectName := chi.URLParam(r, "*")
	if projectName == "" {
		projectName = chi.URLParam(r, "path")
	}
	if projectName == "" {
		projectName = chi.URLParam(r, "name")
	}
	projectName = strings.TrimPrefix(projectName, "/")

	summary, sessions, err := s.db.GetProjectDetail(r.Context(), projectName)
	if err != nil {
		respondError(w, http.StatusNotFound, "Project not found")
		return
	}

	if sessions == nil {
		sessions = []models.Session{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"project":  summary,
		"sessions": sessions,
	})
}

// GetHiddenProjects handles GET /config/hidden.
func (s *Server) GetHiddenProjects(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var hidden []string
	for p := range s.hiddenProjects {
		hidden = append(hidden, p)
	}
	if hidden == nil {
		hidden = []string{}
	}
	respondJSON(w, http.StatusOK, hidden)
}

// HideProject handles POST /config/hide.
func (s *Server) HideProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	s.hiddenProjects[body.Path] = true
	var hidden []string
	for p := range s.hiddenProjects {
		hidden = append(hidden, p)
	}
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":     true,
		"hidden": hidden,
	})
}

// UnhideProject handles POST /config/unhide.
func (s *Server) UnhideProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	delete(s.hiddenProjects, body.Path)
	var hidden []string
	for p := range s.hiddenProjects {
		hidden = append(hidden, p)
	}
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":     true,
		"hidden": hidden,
	})
}

// GetAliases handles GET /config/aliases.
func (s *Server) GetAliases(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"aliases": s.aliases,
	})
}

// SetAliases handles POST /config/aliases.
func (s *Server) SetAliases(w http.ResponseWriter, r *http.Request) {
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	for k, v := range updates {
		s.aliases[k] = v
	}
	aliases := s.aliases
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"aliases": aliases,
	})
}
