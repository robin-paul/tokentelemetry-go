package api

import "net/http"

// GetDSHLifecycle handles GET /dsh/lifecycle.
func (s *Server) GetDSHLifecycle(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"installed":   true,
		"correlation": "none",
		"events":      []interface{}{},
	})
}
