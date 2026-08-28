package api

import (
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/robin-paul/tokentelemetry-go/internal/scanner/parsers"
	"github.com/robin-paul/tokentelemetry-go/internal/store"
)

// GetDSHLifecycle handles GET /dsh/lifecycle and GET /api/dsh/lifecycle.
func (s *Server) GetDSHLifecycle(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sessionID := q.Get("session_id")
	if sessionID == "" {
		sessionID = q.Get("sessionId")
	}

	limit := 500
	if lStr := q.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	var since, until *int64
	if sStr := q.Get("since_ms"); sStr != "" {
		if val, err := strconv.ParseInt(sStr, 10, 64); err == nil {
			since = &val
		}
	} else if sStr := q.Get("since"); sStr != "" {
		if val, err := strconv.ParseInt(sStr, 10, 64); err == nil {
			since = &val
		}
	}

	if uStr := q.Get("until_ms"); uStr != "" {
		if val, err := strconv.ParseInt(uStr, 10, 64); err == nil {
			until = &val
		}
	} else if uStr := q.Get("until"); uStr != "" {
		if val, err := strconv.ParseInt(uStr, 10, 64); err == nil {
			until = &val
		}
	}

	correlation := "none"
	if sessionID != "" {
		sess, err := s.db.GetSessionDetail(r.Context(), sessionID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
				return
			}
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if !sess.StartTime.IsZero() {
			st := sess.StartTime.UnixMilli()
			since = &st
		}
		if !sess.EndTime.IsZero() {
			et := sess.EndTime.UnixMilli()
			until = &et
		}
		correlation = "time-window"
	}

	lifecyclePath := parsers.DefaultDSHLifecycleFilePath()
	installed := false
	if lifecyclePath != "" {
		if _, err := os.Stat(lifecyclePath); err == nil {
			installed = true
		}
	}

	events := parsers.ReadDSHLifecycleEvents(lifecyclePath, since, until, limit)
	summary := parsers.SummarizeDSHLifecycleEvents(events)
	summary.Installed = installed
	summary.Correlation = correlation

	respondJSON(w, http.StatusOK, summary)
}
