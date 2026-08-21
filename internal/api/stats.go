package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// GetStats handles GET /api/stats and GET /stats.
func (s *Server) GetStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	agent := q.Get("agent")
	project := q.Get("project")

	overview, err := s.db.GetStatsOverview(r.Context(), from, to, agent, project)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, overview)
}

// GetDailyStats handles GET /api/stats/daily and GET /stats/daily.
func (s *Server) GetDailyStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	agent := q.Get("agent")
	project := q.Get("project")
	model := q.Get("model")

	summaries, err := s.db.QueryDailySummaries(r.Context(), from, to, agent, project, model)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if summaries == nil {
		summaries = []models.DailySummary{}
	}

	respondJSON(w, http.StatusOK, summaries)
}

// GetLeaderboard handles GET /api/leaderboard and GET /leaderboard.
func (s *Server) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	from := q.Get("from")
	to := q.Get("to")

	modelsList, agentsList, err := s.db.GetLeaderboard(r.Context(), limit, from, to)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if modelsList == nil {
		modelsList = []models.LeaderboardEntry{}
	}
	if agentsList == nil {
		agentsList = []models.LeaderboardEntry{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"models": modelsList,
		"agents": agentsList,
	})
}

// GetAnalytics handles GET /analytics.
func (s *Server) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	granularity := q.Get("granularity")
	if granularity == "" {
		granularity = "day"
	}

	// 1. Fetch daily summaries
	summaries, err := s.db.QueryDailySummaries(r.Context(), from, to, "", "", "")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	byAgent := make(map[string]map[string]interface{})
	byModel := make(map[string]map[string]interface{})
	byDay := []map[string]interface{}{}
	dayAgg := make(map[string]map[string]interface{})

	var totalInput, totalOutput, totalCached, totalCacheReads int64
	var totalCost float64

	for _, sm := range summaries {
		// By Agent
		if _, exists := byAgent[sm.AgentName]; !exists {
			byAgent[sm.AgentName] = map[string]interface{}{
				"input":         int64(0),
				"output":        int64(0),
				"cached":        int64(0),
				"cache_reads":   int64(0),
				"total":         int64(0),
				"cost":          float64(0),
				"energy_wh":     float64(0),
				"savings_usd":   float64(0),
				"co2_g":         float64(0),
				"session_count": int64(0),
				"cache_hit_pct": float64(0),
			}
		}
		agentData := byAgent[sm.AgentName]
		agentData["input"] = agentData["input"].(int64) + sm.TotalInputTokens
		agentData["output"] = agentData["output"].(int64) + sm.TotalOutputTokens
		agentData["cached"] = agentData["cached"].(int64) + sm.TotalCacheCreationTokens
		agentData["cache_reads"] = agentData["cache_reads"].(int64) + sm.TotalCacheReadTokens
		agentData["total"] = agentData["total"].(int64) + sm.TotalInputTokens + sm.TotalOutputTokens + sm.TotalCacheReadTokens + sm.TotalCacheCreationTokens
		agentData["cost"] = agentData["cost"].(float64) + sm.TotalCostUSD
		agentData["session_count"] = agentData["session_count"].(int64) + sm.TotalSessions

		// By Model
		if sm.ModelName != "" {
			if _, exists := byModel[sm.ModelName]; !exists {
				byModel[sm.ModelName] = map[string]interface{}{
					"input":         int64(0),
					"output":        int64(0),
					"cached":        int64(0),
					"total":         int64(0),
					"cost":          float64(0),
					"session_count": int64(0),
				}
			}
			modelData := byModel[sm.ModelName]
			modelData["input"] = modelData["input"].(int64) + sm.TotalInputTokens
			modelData["output"] = modelData["output"].(int64) + sm.TotalOutputTokens
			modelData["cached"] = modelData["cached"].(int64) + sm.TotalCacheReadTokens + sm.TotalCacheCreationTokens
			modelData["total"] = modelData["total"].(int64) + sm.TotalInputTokens + sm.TotalOutputTokens + sm.TotalCacheReadTokens + sm.TotalCacheCreationTokens
			modelData["cost"] = modelData["cost"].(float64) + sm.TotalCostUSD
			modelData["session_count"] = modelData["session_count"].(int64) + sm.TotalSessions
		}

		// By Day
		if _, exists := dayAgg[sm.Date]; !exists {
			dayAgg[sm.Date] = map[string]interface{}{
				"date":        sm.Date,
				"input":       int64(0),
				"output":      int64(0),
				"cached":      int64(0),
				"total":       int64(0),
				"cost":        float64(0),
				"energy_wh":   float64(0),
				"savings_usd": float64(0),
				"co2_g":       float64(0),
			}
		}
		dData := dayAgg[sm.Date]
		dData["input"] = dData["input"].(int64) + sm.TotalInputTokens
		dData["output"] = dData["output"].(int64) + sm.TotalOutputTokens
		dData["cached"] = dData["cached"].(int64) + sm.TotalCacheReadTokens + sm.TotalCacheCreationTokens
		dData["total"] = dData["total"].(int64) + sm.TotalInputTokens + sm.TotalOutputTokens + sm.TotalCacheReadTokens + sm.TotalCacheCreationTokens
		dData["cost"] = dData["cost"].(float64) + sm.TotalCostUSD

		totalInput += sm.TotalInputTokens
		totalOutput += sm.TotalOutputTokens
		totalCached += sm.TotalCacheCreationTokens
		totalCacheReads += sm.TotalCacheReadTokens
		totalCost += sm.TotalCostUSD
	}

	for _, dData := range dayAgg {
		byDay = append(byDay, dData)
	}

	totalTokens := totalInput + totalOutput + totalCached + totalCacheReads
	cacheHitPct := float64(0)
	if totalInput+totalCacheReads > 0 {
		cacheHitPct = (float64(totalCacheReads) / float64(totalInput+totalCacheReads)) * 100.0
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"by_agent":         byAgent,
		"by_day":           byDay,
		"by_model":         byModel,
		"by_skill":         map[string]interface{}{},
		"by_mcp_server":    map[string]interface{}{},
		"by_subagent_type": map[string]interface{}{},
		"by_loop":          map[string]interface{}{},
		"loops":            map[string]interface{}{},
		"delegation":       map[string]interface{}{},
		"total": map[string]interface{}{
			"input":         totalInput,
			"output":        totalOutput,
			"cached":        totalCached,
			"cache_reads":   totalCacheReads,
			"total":         totalTokens,
			"cost":          totalCost,
			"energy_wh":     0.0,
			"savings_usd":   0.0,
			"co2_g":         0.0,
			"cache_hit_pct": cacheHitPct,
		},
		"coverage":        map[string]interface{}{},
		"granularity":     granularity,
		"pricing_updated": time.Now().Format("2006-01-02"),
	})
}
