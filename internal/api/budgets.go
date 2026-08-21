package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// Budget represents spend or token limits.
type Budget struct {
	ID                string                 `json:"id"`
	Filters           map[string]string      `json:"filters"`
	Period            string                 `json:"period"`
	LimitType         string                 `json:"limit_type"`
	LimitValue        float64                `json:"limit_value"`
	Thresholds        []float64              `json:"thresholds"`
	Enabled           bool                   `json:"enabled"`
	Used              float64                `json:"used"`
	Fraction          float64                `json:"fraction"`
	AlertLevel        float64                `json:"alert_level"`
	SessionsInWindow  int                    `json:"sessions_in_window"`
	WindowStart       string                 `json:"window_start"`
	PeriodKey         string                 `json:"period_key"`
	ResetAt           string                 `json:"reset_at"`
	BreakdownByAgent  map[string]interface{} `json:"breakdown_by_agent"`
}

// GetBudgets handles GET /budgets.
func (s *Server) GetBudgets(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var budgets []Budget
	if len(s.budgetsJSON) > 0 {
		var wrapper struct {
			Budgets []Budget `json:"budgets"`
		}
		if err := json.Unmarshal(s.budgetsJSON, &wrapper); err == nil && len(wrapper.Budgets) > 0 {
			budgets = wrapper.Budgets
		} else {
			_ = json.Unmarshal(s.budgetsJSON, &budgets)
		}
	}

	if budgets == nil {
		budgets = []Budget{}
	}

	// Calculate overview metrics for each budget
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextMonth := startOfMonth.AddDate(0, 1, 0)

	for i := range budgets {
		if budgets[i].WindowStart == "" {
			budgets[i].WindowStart = startOfMonth.Format(time.RFC3339)
		}
		if budgets[i].ResetAt == "" {
			budgets[i].ResetAt = nextMonth.Format(time.RFC3339)
		}
		if budgets[i].PeriodKey == "" {
			budgets[i].PeriodKey = startOfMonth.Format("2006-01-02")
		}
		if budgets[i].BreakdownByAgent == nil {
			budgets[i].BreakdownByAgent = make(map[string]interface{})
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"budgets": budgets,
	})
}

// SetBudgets handles PUT /budgets.
func (s *Server) SetBudgets(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var parsed interface{}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	s.mu.Lock()
	s.budgetsJSON = bodyBytes
	s.mu.Unlock()

	s.GetBudgets(w, r)
}
