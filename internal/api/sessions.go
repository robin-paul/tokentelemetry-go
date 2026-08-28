package api

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
	"github.com/robin-paul/tokentelemetry-go/internal/scanner/parsers"
	"github.com/robin-paul/tokentelemetry-go/internal/store"
)

func parseMultiValue(q url.Values, keys ...string) []string {
	var result []string
	for _, key := range keys {
		values := q[key]
		for _, v := range values {
			for _, item := range strings.Split(v, ",") {
				trimmed := strings.TrimSpace(item)
				if trimmed != "" {
					result = append(result, trimmed)
				}
			}
		}
	}
	return result
}

func parseDateTime(val string) time.Time {
	if val == "" {
		return time.Time{}
	}
	if sec, err := strconv.ParseInt(val, 10, 64); err == nil {
		if sec > 1e11 {
			return time.UnixMilli(sec).UTC()
		}
		return time.Unix(sec, 0).UTC()
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, val); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func parseSessionFilterParams(r *http.Request) models.FilterParams {
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

	// Search
	search := q.Get("q")
	if search == "" {
		search = q.Get("search")
	}
	searchScope := q.Get("search_scope")

	// Multi-select dimensions
	agents := parseMultiValue(q, "agent")
	projects := parseMultiValue(q, "project")
	modelsList := parseMultiValue(q, "model")
	machineIDs := parseMultiValue(q, "machine_id")
	tools := parseMultiValue(q, "tool", "tool_name")
	subagentTypes := parseMultiValue(q, "subagent_type")

	// Dates
	startStr := q.Get("since")
	if startStr == "" {
		startStr = q.Get("from")
	}
	if startStr == "" {
		startStr = q.Get("start_date")
	}
	startDate := parseDateTime(startStr)

	endStr := q.Get("until")
	if endStr == "" {
		endStr = q.Get("to")
	}
	if endStr == "" {
		endStr = q.Get("end_date")
	}
	endDate := parseDateTime(endStr)

	// Status & Git Branch
	status := q.Get("status")
	gitBranch := q.Get("git_branch")
	parentSessionID := q.Get("parent_session_id")

	var isSubagent *bool
	if subStr := q.Get("is_subagent"); subStr != "" {
		if subStr == "true" || subStr == "1" {
			v := true
			isSubagent = &v
		} else if subStr == "false" || subStr == "0" {
			v := false
			isSubagent = &v
		}
	}

	// Numeric range bounds
	var minCost, maxCost *float64
	if v, err := strconv.ParseFloat(q.Get("min_cost"), 64); err == nil {
		minCost = &v
	}
	if v, err := strconv.ParseFloat(q.Get("max_cost"), 64); err == nil {
		maxCost = &v
	}

	var minTokens, maxTokens *int64
	if v, err := strconv.ParseInt(q.Get("min_tokens"), 10, 64); err == nil {
		minTokens = &v
	}
	if v, err := strconv.ParseInt(q.Get("max_tokens"), 10, 64); err == nil {
		maxTokens = &v
	}

	var minInputTokens, maxInputTokens *int64
	if v, err := strconv.ParseInt(q.Get("min_input_tokens"), 10, 64); err == nil {
		minInputTokens = &v
	}
	if v, err := strconv.ParseInt(q.Get("max_input_tokens"), 10, 64); err == nil {
		maxInputTokens = &v
	}

	var minOutputTokens, maxOutputTokens *int64
	if v, err := strconv.ParseInt(q.Get("min_output_tokens"), 10, 64); err == nil {
		minOutputTokens = &v
	}
	if v, err := strconv.ParseInt(q.Get("max_output_tokens"), 10, 64); err == nil {
		maxOutputTokens = &v
	}

	var minDuration, maxDuration *float64
	if v, err := strconv.ParseFloat(q.Get("min_duration"), 64); err == nil {
		minDuration = &v
	}
	if v, err := strconv.ParseFloat(q.Get("max_duration"), 64); err == nil {
		maxDuration = &v
	}

	// Sorting
	sortBy := models.SortField(q.Get("sort_by"))
	sortOrderStr := strings.ToLower(q.Get("sort_order"))
	if sortOrderStr == "" {
		sortOrderStr = strings.ToLower(q.Get("order"))
	}
	sortOrder := models.SortOrderDesc
	if sortOrderStr == "asc" {
		sortOrder = models.SortOrderAsc
	}

	return models.FilterParams{
		Page:            page,
		Limit:           limit,
		Search:          search,
		SearchScope:     searchScope,
		Agents:          agents,
		Projects:        projects,
		Models:          modelsList,
		MachineIDs:      machineIDs,
		Tools:           tools,
		SubagentTypes:   subagentTypes,
		StartDate:       startDate,
		EndDate:         endDate,
		Status:          status,
		GitBranch:       gitBranch,
		IsSubagent:      isSubagent,
		ParentSessionID: parentSessionID,
		MinCostUSD:      minCost,
		MaxCostUSD:      maxCost,
		MinTokens:       minTokens,
		MaxTokens:       maxTokens,
		MinInputTokens:  minInputTokens,
		MaxInputTokens:  maxInputTokens,
		MinOutputTokens: minOutputTokens,
		MaxOutputTokens: maxOutputTokens,
		MinDurationSec:  minDuration,
		MaxDurationSec:  maxDuration,
		SortBy:          sortBy,
		SortOrder:       sortOrder,
		Format:          q.Get("format"),
	}
}

// ListSessions handles GET /api/sessions and GET /sessions.
func (s *Server) ListSessions(w http.ResponseWriter, r *http.Request) {
	params := parseSessionFilterParams(r)

	sessions, total, err := s.db.ListSessions(r.Context(), params)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Support both wrapped response with pagination and plain session array
	if params.Format == "paginated" || r.URL.Query().Get("page_size") != "" {
		totalPages := int((total + int64(params.Limit) - 1) / int64(params.Limit))
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"sessions": sessions,
			"pagination": map[string]interface{}{
				"page":        params.Page,
				"page_size":   params.Limit,
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

	if sess.AgentName == "dsh" && sess.FilePath != "" && sess.DSH == nil {
		if f, err := os.Open(sess.FilePath); err == nil {
			p := parsers.NewDSHParser()
			if parsed, _, parseErr := p.Parse(f, 0); parseErr == nil && parsed != nil {
				sess.DSH = parsed.DSH
			}
			f.Close()
		}
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
