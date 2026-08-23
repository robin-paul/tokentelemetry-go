package api

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
	"github.com/robin-paul/tokentelemetry-go/internal/store"
)

// ListSessions handles GET /api/sessions and GET /sessions.
func (s *Server) ListSessions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit, _ = strconv.Atoi(q.Get("page_size"))
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var startDate, endDate time.Time
	if fromStr := q.Get("from"); fromStr != "" {
		startDate, _ = time.Parse("2006-01-02", fromStr)
		if startDate.IsZero() {
			startDate, _ = time.Parse(time.RFC3339, fromStr)
		}
	}
	if toStr := q.Get("to"); toStr != "" {
		endDate, _ = time.Parse("2006-01-02", toStr)
		if endDate.IsZero() {
			endDate, _ = time.Parse(time.RFC3339, toStr)
		}
	}

	params := models.FilterParams{
		Page:      page,
		Limit:     limit,
		Agent:     q.Get("agent"),
		Project:   q.Get("project"),
		Model:     q.Get("model"),
		StartDate: startDate,
		EndDate:   endDate,
		Search:    q.Get("search"),
	}

	sessions, total, err := s.db.ListSessions(r.Context(), params)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Support both wrapped response with pagination and plain session array
	if q.Get("format") == "paginated" || q.Get("page_size") != "" {
		totalPages := int((total + int64(limit) - 1) / int64(limit))
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"sessions": sessions,
			"pagination": map[string]interface{}{
				"page":        page,
				"page_size":   limit,
				"total":       total,
				"total_pages": totalPages,
			},
		})
		return
	}

	if sessions == nil {
		sessions = []models.Session{}
	}
	respondJSON(w, http.StatusOK, sessions)
}

// GetSession handles GET /api/sessions/{id} and GET /sessions/{id}.
func (s *Server) GetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		id = chi.URLParam(r, "session_id")
	}
	if decoded, err := url.PathUnescape(id); err == nil && decoded != "" {
		id = decoded
	}
	if id == "" {
		respondError(w, http.StatusBadRequest, "Missing session ID")
		return
	}

	sess, err := s.db.GetSessionDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "Session not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, sess)
}

// DeleteSession handles DELETE /api/sessions/{id} and DELETE /sessions/{id}.
func (s *Server) DeleteSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		id = chi.URLParam(r, "session_id")
	}
	if decoded, err := url.PathUnescape(id); err == nil && decoded != "" {
		id = decoded
	}
	if id == "" {
		respondError(w, http.StatusBadRequest, "Missing session ID")
		return
	}

	err := s.db.DeleteSession(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "Session not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

// GetRecentSessions handles GET /api/recent and GET /recent.
func (s *Server) GetRecentSessions(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	sessions, err := s.db.GetRecentSessions(r.Context(), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sessions == nil {
		sessions = []models.Session{}
	}
	respondJSON(w, http.StatusOK, sessions)
}

// GetSubagentTrace handles GET /sessions/{id}/subagents/{subagent_id}/trace.
func (s *Server) GetSubagentTrace(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	subagentID := chi.URLParam(r, "subagent_id")

	// Look up subagent details from DB
	subagents, err := s.db.GetSubagentRuns(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var matched *models.SubagentRun
	for _, sub := range subagents {
		if sub.ChildSessionID == subagentID || sub.ID == subagentID {
			matched = &sub
			break
		}
	}

	if matched == nil {
		respondJSON(w, http.StatusOK, []interface{}{})
		return
	}

	// If child session is tracked in sessions table, return its turns
	childSess, err := s.db.GetSessionDetail(r.Context(), matched.ChildSessionID)
	if err == nil && childSess != nil {
		respondJSON(w, http.StatusOK, childSess.Turns)
		return
	}

	respondJSON(w, http.StatusOK, []interface{}{})
}

// GetDelegation handles GET /sessions/{id}/delegation.
func (s *Server) GetDelegation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	subagents, err := s.db.GetSubagentRuns(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var totalTokens int64
	var totalCost float64
	for _, sub := range subagents {
		totalTokens += sub.Tokens
		totalCost += sub.CostUSD
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"supported":       true,
		"tokens_recorded": true,
		"spawn_count":     len(subagents),
		"subagents":       subagents,
		"totals": map[string]interface{}{
			"tokens": totalTokens,
			"cost":   totalCost,
		},
		"cost": totalCost,
	})
}

// GetGrokForensics handles GET /sessions/{id}/grok-forensics.
func (s *Server) GetGrokForensics(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := s.db.GetSessionDetail(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Session not found")
		return
	}

	toolsUsed := []string{}
	for _, t := range sess.Turns {
		toolsUsed = append(toolsUsed, t.ToolsInvoked...)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"session_id": sess.SessionID,
		"signals": map[string]interface{}{
			"context_tokens_used":      sess.InputTokens + sess.OutputTokens,
			"context_window_tokens":    256000,
			"context_window_usage_pct": 0.0,
			"tool_call_count":          len(toolsUsed),
			"tools_used":               toolsUsed,
			"models_used":              []string{sess.ModelResolved},
			"session_duration_seconds": sess.DurationSeconds,
			"turn_count":               len(sess.Turns),
		},
		"tool_events":      []interface{}{},
		"token_progression": []interface{}{},
	})
}
