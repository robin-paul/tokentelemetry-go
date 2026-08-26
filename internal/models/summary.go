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

// WorktreeSummary represents an individual git worktree linked to a parent repository.
type WorktreeSummary struct {
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	SessionCount int64    `json:"session_count"`
	TotalTokens  int64    `json:"total_tokens"`
	TotalCostUSD float64  `json:"total_cost_usd"`
	Agents       []string `json:"agents"`
	Status       string   `json:"status"` // "active" | "missing"
}

// ProjectAggregate holds rolled-up metrics across a repository root and all its worktrees.
type ProjectAggregate struct {
	SessionCount            int64    `json:"session_count"`
	SubagentCount           int64    `json:"subagent_count"`
	PlanCount               int64    `json:"plan_count"`
	ConfiguredSubagentCount int64    `json:"configured_subagent_count"`
	TotalTokens             int64    `json:"total_tokens"`
	TotalCostUSD            float64  `json:"total_cost_usd"`
	Agents                  []string `json:"agents"`
	MCPTools                []string `json:"mcp_tools"`
	WorktreeCount           int      `json:"worktree_count"`
}

// ProjectSummary represents aggregated project statistics and git worktree relationships.
type ProjectSummary struct {
	ProjectName             string            `json:"project_name"`
	Name                    string            `json:"name,omitempty"`
	Path                    string            `json:"path"`
	SessionCount            int64             `json:"session_count"`
	TotalTokens             int64             `json:"total_tokens"`
	TotalCostUSD            float64           `json:"total_cost_usd"`
	LastActive              time.Time         `json:"last_active"`
	Agents                  []string          `json:"agents"`
	MCPTools                []string          `json:"mcp_tools"`
	SubagentCount           int64             `json:"subagent_count"`
	ConfiguredSubagentCount int64             `json:"configured_subagent_count"`
	PlanCount               int64             `json:"plan_count"`
	Status                  string            `json:"status"` // "active" | "missing"
	CanonicalRepo           string            `json:"canonical_repo,omitempty"`
	IsWorktree              bool              `json:"is_worktree"`
	WorktreeName            string            `json:"worktree_name,omitempty"`
	ParentPath              string            `json:"parent_path,omitempty"`
	ParentName              string            `json:"parent_name,omitempty"`
	IsRepoRoot              bool              `json:"is_repo_root"`
	Synthesized             bool              `json:"synthesized"`
	Worktrees               []WorktreeSummary `json:"worktrees,omitempty"`
	Aggregate               *ProjectAggregate `json:"aggregate,omitempty"`
}

// SortField defines the validated field to sort session queries by.
type SortField string

const (
	SortByStartTime    SortField = "start_time"
	SortByEndTime      SortField = "end_time"
	SortByUpdatedAt    SortField = "updated_at"
	SortByCost         SortField = "cost"
	SortByTokens       SortField = "tokens"
	SortByInputTokens  SortField = "input_tokens"
	SortByOutputTokens SortField = "output_tokens"
	SortByDuration     SortField = "duration"
	SortByRelevance    SortField = "relevance"
)

// SortOrder defines sort direction.
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// FilterParams encapsulates query filter parameters for API endpoints.
type FilterParams struct {
	Page            int       `json:"page"`
	Limit           int       `json:"limit"`
	Agent           string    `json:"agent,omitempty"`
	Agents          []string  `json:"agents,omitempty"`
	Project         string    `json:"project,omitempty"`
	Projects        []string  `json:"projects,omitempty"`
	Model           string    `json:"model,omitempty"`
	Models          []string  `json:"models,omitempty"`
	MachineID       string    `json:"machine_id,omitempty"`
	MachineIDs      []string  `json:"machine_ids,omitempty"`
	Status          string    `json:"status,omitempty"`
	GitBranch       string    `json:"git_branch,omitempty"`
	IsSubagent      *bool     `json:"is_subagent,omitempty"`
	ParentSessionID string    `json:"parent_session_id,omitempty"`
	SubagentTypes   []string  `json:"subagent_types,omitempty"`
	Tools           []string  `json:"tools,omitempty"`
	StartDate       time.Time `json:"start_date,omitempty"`
	EndDate         time.Time `json:"end_date,omitempty"`
	MinCostUSD      *float64  `json:"min_cost_usd,omitempty"`
	MaxCostUSD      *float64  `json:"max_cost_usd,omitempty"`
	MinTokens       *int64    `json:"min_tokens,omitempty"`
	MaxTokens       *int64    `json:"max_tokens,omitempty"`
	MinInputTokens  *int64    `json:"min_input_tokens,omitempty"`
	MaxInputTokens  *int64    `json:"max_input_tokens,omitempty"`
	MinOutputTokens *int64    `json:"min_output_tokens,omitempty"`
	MaxOutputTokens *int64    `json:"max_output_tokens,omitempty"`
	MinDurationSec  *float64  `json:"min_duration_sec,omitempty"`
	MaxDurationSec  *float64  `json:"max_duration_sec,omitempty"`
	Search          string    `json:"search,omitempty"`
	SearchScope     string    `json:"search_scope,omitempty"`
	SortBy          SortField `json:"sort_by,omitempty"`
	SortOrder       SortOrder `json:"sort_order,omitempty"`
	Format          string    `json:"format,omitempty"`
}
