package api

import (
	"net/http"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// GetHermesOverview handles GET /hermes/overview.
func (s *Server) GetHermesOverview(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"installed": true,
		"gateway": map[string]interface{}{
			"running": true,
			"pid":     0,
			"port":    8080,
		},
		"cron_jobs": []interface{}{},
	})
}

// GetHermesTelemetry handles GET /hermes/telemetry.
func (s *Server) GetHermesTelemetry(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"outcomes": map[string]interface{}{
			"success": 0,
			"failed":  0,
		},
		"costs": map[string]interface{}{
			"total_usd": 0.0,
		},
		"latency": map[string]interface{}{
			"p50_ms": 0.0,
			"p95_ms": 0.0,
		},
		"tool_failures": map[string]interface{}{},
	})
}

// GetHermesSessions handles GET /hermes/sessions.
func (s *Server) GetHermesSessions(w http.ResponseWriter, r *http.Request) {
	sessions, total, err := s.db.ListSessions(r.Context(), models.FilterParams{
		Agent: "hermes",
		Limit: 50,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if sessions == nil {
		sessions = []models.Session{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"pagination": map[string]interface{}{
			"page":        1,
			"page_size":   50,
			"total":       total,
			"total_pages": (total + 49) / 50,
		},
	})
}

// GetHermesSkills handles GET /hermes/skills.
func (s *Server) GetHermesSkills(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"snapshot_loaded": 0,
		"skills":          []interface{}{},
		"categories":      map[string]interface{}{},
	})
}

// GetHermesMemory handles GET /hermes/memory.
func (s *Server) GetHermesMemory(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"memory": map[string]interface{}{
			"entries":    []interface{}{},
			"char_count": 0,
			"exists":     true,
		},
		"user": map[string]interface{}{
			"entries":    []interface{}{},
			"char_count": 0,
		},
		"memory_char_limit": 2200,
		"user_char_limit":   1375,
	})
}

// GetHermesSoul handles GET /hermes/soul.
func (s *Server) GetHermesSoul(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"content": "Hermes autonomous assistant persona definition",
		"exists":  true,
	})
}

// GetHermesProfiles handles GET /hermes/profiles.
func (s *Server) GetHermesProfiles(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"profiles":       []string{"default"},
		"active_profile": "default",
	})
}

// GetHermesKanban handles GET /api/hermes/kanban and GET /hermes/kanban.
func (s *Server) GetHermesKanban(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"installed": true,
		"boards": []map[string]interface{}{
			{
				"id":    "tasks",
				"title": "Hermes Autonomous Tasks",
				"columns": []map[string]interface{}{
					{"id": "todo", "title": "To Do", "cards": []interface{}{}},
					{"id": "in_progress", "title": "In Progress", "cards": []interface{}{}},
					{"id": "done", "title": "Done", "cards": []interface{}{}},
				},
			},
		},
	})
}

// GetHermesTools handles GET /hermes/tools.
func (s *Server) GetHermesTools(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"enabled_tools": []string{"cli", "filesystem"},
	})
}
