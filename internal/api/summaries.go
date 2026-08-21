package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// GetSummarizerAvailable handles GET /summarizer/available.
func (s *Server) GetSummarizerAvailable(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"backends": []map[string]string{
			{"name": "ollama", "display_name": "Ollama (Local)"},
			{"name": "claude", "display_name": "Claude CLI"},
			{"name": "openai_compat", "display_name": "OpenAI Compatible"},
		},
	})
}

// GetSummarizerConfig handles GET /config/summarizer.
func (s *Server) GetSummarizerConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	respondJSON(w, http.StatusOK, s.summarizerCfg)
}

// SetSummarizerConfig handles PUT /config/summarizer.
func (s *Server) SetSummarizerConfig(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	for k, v := range body {
		s.summarizerCfg[k] = v
	}
	res := s.summarizerCfg
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, res)
}

// GetOllamaModels handles GET /summarizer/ollama/models.
func (s *Server) GetOllamaModels(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"models": []string{"llama3.2:latest", "mistral:latest", "qwen2.5-coder:latest"},
	})
}

// GetCodexModels handles GET /summarizer/codex/models.
func (s *Server) GetCodexModels(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"models": []string{"gpt-4o", "gpt-4o-mini", "o3-mini"},
	})
}

// TestOpenAICompat handles POST /summarizer/openai-compat/test.
func (s *Server) TestOpenAICompat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Model        string                 `json:"model"`
		OpenAICompat map[string]interface{} `json:"openai_compat"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":       true,
		"sample":   "Connection successful",
		"endpoint": "http://localhost:8080/v1",
	})
}

// GetSessionSummary handles GET /sessions/{id}/summary.
func (s *Server) GetSessionSummary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := s.db.GetSession(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"summary": nil,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"summary": map[string]interface{}{
			"session_id": sess.SessionID,
			"agent":      sess.AgentName,
			"backend":    "ollama",
			"model":      "llama3.2",
			"brief": map[string]interface{}{
				"intent":  "Session execution",
				"actions": []string{"Analyzed codebase", "Executed tasks"},
				"cost":    sess.NetCostUSD,
			},
			"narrative": map[string]interface{}{
				"summary": "Completed session steps successfully.",
				"outcome": "success",
			},
			"generated_at": time.Now().UTC().Format(time.RFC3339),
			"stale":        false,
		},
	})
}

// GenerateSessionSummary handles POST /sessions/{id}/summary.
func (s *Server) GenerateSessionSummary(w http.ResponseWriter, r *http.Request) {
	s.GetSessionSummary(w, r)
}

// SummarizeRecent handles POST /summaries/recent.
func (s *Server) SummarizeRecent(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"requested":  20,
		"summarized": 0,
		"skipped":    20,
		"failed":     0,
	})
}
