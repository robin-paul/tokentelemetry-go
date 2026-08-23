package models

import "time"

// DailySummary represents pre-aggregated token and cost metrics grouped by date, agent, project, and model.
type DailySummary struct {
	Date                     string  `json:"date"`
	AgentName                string  `json:"agent_name"`
	ProjectName              string  `json:"project_name"`
	ModelName                string  `json:"model_name"`
	TotalSessions            int64   `json:"total_sessions"`
	TotalInputTokens         int64   `json:"total_input_tokens"`
	TotalOutputTokens        int64   `json:"total_output_tokens"`
	TotalCacheReadTokens     int64   `json:"total_cache_read_tokens"`
	TotalCacheCreationTokens int64   `json:"total_cache_creation_tokens"`
	TotalCostUSD             float64 `json:"total_cost_usd"`
	TotalDurationSeconds     float64 `json:"total_duration_seconds"`
}

// LeaderboardEntry represents ranking metrics for models or agents.
type LeaderboardEntry struct {
	Name         string  `json:"name"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	SessionCount int64   `json:"session_count"`
}

// StatsOverview represents high-level dashboard aggregate metrics.
type StatsOverview struct {
	TotalSessions    int64   `json:"total_sessions"`
	TotalInputTokens int64   `json:"total_input_tokens"`
	TotalOutputTokens int64  `json:"total_output_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	GrossCostUSD     float64 `json:"gross_cost_usd"`
	NetCostUSD       float64 `json:"net_cost_usd"`
	ActiveAgents     int     `json:"active_agents"`
	ActiveProjects   int     `json:"active_projects"`
}

// ProjectSummary represents aggregated project statistics.
type ProjectSummary struct {
	ProjectName  string    `json:"project_name"`
	SessionCount int64     `json:"session_count"`
	TotalTokens  int64     `json:"total_tokens"`
	TotalCostUSD float64   `json:"total_cost_usd"`
	LastActive   time.Time `json:"last_active"`
}

// FilterParams encapsulates query filter parameters for API endpoints.
type FilterParams struct {
	Page      int       `json:"page"`
	Limit     int       `json:"limit"`
	Agent     string    `json:"agent"`
	Project   string    `json:"project"`
	Model     string    `json:"model"`
	MachineID string    `json:"machine_id"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Search    string    `json:"search"`
}
