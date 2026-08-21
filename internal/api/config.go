package api

import (
	"encoding/json"
	"net/http"
)

// GetConfig handles GET /config.
func (s *Server) GetConfig(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"project":       project,
		"project_valid": true,
		"skills": []map[string]interface{}{
			{
				"name":        "research",
				"description": "Investigate facts from primary sources",
				"scope":       "user",
				"agent":       "claude",
				"source":      "~/.claude/skills/research/SKILL.md",
			},
		},
		"mcps":      []interface{}{},
		"memory":    []interface{}{},
		"commands":  []interface{}{},
		"subagents": []interface{}{},
		"plugins":   []interface{}{},
		"counts": map[string]interface{}{
			"skills":       1,
			"mcps":         0,
			"memory_files": 0,
			"commands":     0,
			"subagents":    0,
			"plugins":      0,
		},
	})
}

// GetUpdateCheck handles GET /config/update-check.
func (s *Server) GetUpdateCheck(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	respondJSON(w, http.StatusOK, s.updateCheckCfg)
}

// SetUpdateCheck handles POST /config/update-check.
func (s *Server) SetUpdateCheck(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	for k, v := range body {
		s.updateCheckCfg[k] = v
	}
	res := s.updateCheckCfg
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, res)
}

// GetTelemetryConfig handles GET /config/telemetry.
func (s *Server) GetTelemetryConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	respondJSON(w, http.StatusOK, s.telemetryCfg)
}

// SetTelemetryConfig handles POST /config/telemetry.
func (s *Server) SetTelemetryConfig(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	for k, v := range body {
		s.telemetryCfg[k] = v
	}
	res := s.telemetryCfg
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, res)
}

// AckTelemetry handles POST /config/telemetry/ack.
func (s *Server) AckTelemetry(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.telemetryCfg["notice_ack"] = true
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"notice_ack": true,
	})
}

// GetTelemetryPreview handles GET /config/telemetry/preview.
func (s *Server) GetTelemetryPreview(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"payload_sample": map[string]interface{}{
			"event": "app.launched",
		},
		"privacy": "Only anonymized event identifiers and metric aggregates are sent.",
	})
}

// PostTelemetryEvent handles POST /telemetry/event.
func (s *Server) PostTelemetryEvent(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

// GetRetentionConfig handles GET /config/retention.
func (s *Server) GetRetentionConfig(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"agents": map[string]interface{}{
			"claude": map[string]interface{}{
				"label":             "Claude Code",
				"default_days":      30,
				"effective_days":    30,
				"detected_override": nil,
				"configurable":      true,
				"archivable":        true,
				"archive_enabled":   false,
			},
		},
		"storage": map[string]interface{}{
			"sessions_rows":    100,
			"transcripts_rows": 0,
			"summaries_rows":   10,
			"db_bytes":         65536,
			"transcript_bytes": 0,
		},
		"coverage": map[string]interface{}{},
	})
}

// SetRetentionConfig handles POST /config/retention.
func (s *Server) SetRetentionConfig(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"archive": map[string]interface{}{"claude": true},
	})
}

// DeleteTranscripts handles DELETE /history/transcripts.
func (s *Server) DeleteTranscripts(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"deleted": 0,
		"storage": map[string]interface{}{},
	})
}

// GetPowerConfig handles GET /config/power.
func (s *Server) GetPowerConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	respondJSON(w, http.StatusOK, s.powerCfg)
}

// SetPowerConfig handles PUT /config/power.
func (s *Server) SetPowerConfig(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	for k, v := range body {
		s.powerCfg[k] = v
	}
	s.powerCfg["configured"] = true
	res := s.powerCfg
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, res)
}

// GetPowerMeter handles GET /config/power/meter.
func (s *Server) GetPowerMeter(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"capability": map[string]interface{}{
			"available": true,
			"source":    "apple-silicon-default",
			"reason":    nil,
		},
		"reading": 18.5,
	})
}

// CalibratePower handles POST /config/power/calibrate.
func (s *Server) CalibratePower(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"measured": 22.4,
		"source":   "calibrated-baseline",
		"samples":  []float64{21.8, 23.0, 22.4},
	})
}

// GetBillingConfig handles GET /config/billing.
func (s *Server) GetBillingConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	respondJSON(w, http.StatusOK, s.billingCfg)
}

// SetBillingConfig handles PUT /config/billing.
func (s *Server) SetBillingConfig(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	for k, v := range body {
		s.billingCfg[k] = v
	}
	res := s.billingCfg
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, res)
}

// GetBillingRouteConfig handles GET /config/billing-route.
func (s *Server) GetBillingRouteConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	respondJSON(w, http.StatusOK, s.billingRoute)
}

// SetBillingRouteConfig handles PUT /config/billing-route.
func (s *Server) SetBillingRouteConfig(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	for k, v := range body {
		s.billingRoute[k] = v
	}
	res := s.billingRoute
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, res)
}

// GetAgentFeatures handles GET /config/agent-features.
func (s *Server) GetAgentFeatures(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"agents": []map[string]interface{}{
			{
				"agent":    "claude",
				"detected": true,
				"source":   "~/.claude/settings.json",
				"flags":    []map[string]interface{}{},
			},
		},
		"not_detectable": []interface{}{},
	})
}
