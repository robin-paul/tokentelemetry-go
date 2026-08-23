package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// GetPricing handles GET /api/pricing and GET /pricing.
func (s *Server) GetPricing(w http.ResponseWriter, r *http.Request) {
	overrides, err := s.db.GetPricingOverrides(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	modelMap := make(map[string]interface{})
	if s.pricingEngine != nil && s.pricingEngine.Dataset != nil {
		for pattern, m := range s.pricingEngine.Dataset.GetAllRates() {
			modelMap[pattern] = map[string]interface{}{
				"in":           m.InputCostPerM,
				"out":          m.OutputCostPerM,
				"cached_read":  m.CacheReadCostPerM,
				"cached_write": m.CacheWriteCostPerM,
				"source":       m.Source,
			}
		}
	}

	if overrides == nil {
		overrides = []models.PricingOverride{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"updated":   time.Now().Format("2006-01-02"),
		"models":    modelMap,
		"overrides": overrides,
	})
}

// UpsertPricingOverride handles POST /api/pricing/override and PUT /api/pricing/override.
func (s *Server) UpsertPricingOverride(w http.ResponseWriter, r *http.Request) {
	var override models.PricingOverride
	if err := json.NewDecoder(r.Body).Decode(&override); err != nil || override.ModelPattern == "" {
		respondError(w, http.StatusBadRequest, "Invalid pricing override payload")
		return
	}

	if override.Source == "" {
		override.Source = "user_override"
	}
	override.UpdatedAt = time.Now().UTC()

	if err := s.db.UpsertPricingOverride(r.Context(), &override); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":       true,
		"override": override,
	})
}

// DeletePricingOverride handles DELETE /api/pricing/override/{pattern}.
func (s *Server) DeletePricingOverride(w http.ResponseWriter, r *http.Request) {
	pattern := chi.URLParam(r, "pattern")
	if pattern == "" {
		pattern = chi.URLParam(r, "*")
	}
	if decoded, err := url.PathUnescape(pattern); err == nil && decoded != "" {
		pattern = decoded
	}
	if pattern == "" {
		respondError(w, http.StatusBadRequest, "Missing model pattern")
		return
	}

	if err := s.db.DeletePricingOverride(r.Context(), pattern); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}
