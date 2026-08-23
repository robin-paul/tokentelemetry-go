package models

import "time"

// TokenUsage holds raw and cache token counts.
type TokenUsage struct {
	InputTokens         int64 `json:"input_tokens"`
	OutputTokens        int64 `json:"output_tokens"`
	CacheReadTokens     int64 `json:"cache_read_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_tokens"`
}

// MessageTurn represents a single conversational turn with token metrics and tool invocations.
type MessageTurn struct {
	ID                  string     `json:"id"`
	SessionID           string     `json:"session_id"`
	TurnIndex           int        `json:"turn_index"`
	Timestamp           time.Time  `json:"timestamp"`
	Role                string     `json:"role"`
	ModelName           string     `json:"model_name"`
	InputTokens         int64      `json:"input_tokens"`
	OutputTokens        int64      `json:"output_tokens"`
	CacheReadTokens     int64      `json:"cache_read_tokens"`
	CacheCreationTokens int64      `json:"cache_creation_tokens"`
	CostUSD             float64    `json:"cost_usd"`
	ToolsInvokedJSON    string     `json:"tools_invoked_json"`
	ToolsInvoked        []string   `json:"tools_invoked,omitempty"`
}

// SubagentRun represents a subagent session spawned by a parent orchestrator.
type SubagentRun struct {
	ID              string    `json:"id"`
	ParentSessionID string    `json:"parent_session_id"`
	ChildSessionID  string    `json:"child_session_id"`
	AgentType       string    `json:"agent_type"`
	Tokens          int64     `json:"tokens"`
	CostUSD         float64   `json:"cost_usd"`
	CreatedAt       time.Time `json:"created_at"`
}

// Session represents a complete conversation or execution session across an agent ecosystem.
type Session struct {
	ID                  string        `json:"id"`
	SessionID           string        `json:"session_id"`
	AgentName           string        `json:"agent_name"`
	ProjectName         string        `json:"project_name"`
	FilePath            string        `json:"file_path"`
	MachineID           string        `json:"machine_id,omitempty"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
	StartTime           time.Time     `json:"start_time"`
	EndTime             time.Time     `json:"end_time"`
	DurationSeconds     float64       `json:"duration_seconds"`
	ModelRaw            string        `json:"model_raw"`
	ModelResolved       string        `json:"model_resolved"`
	InputTokens         int64         `json:"input_tokens"`
	OutputTokens        int64         `json:"output_tokens"`
	CacheReadTokens     int64         `json:"cache_read_tokens"`
	CacheCreationTokens int64         `json:"cache_creation_tokens"`
	GrossCostUSD        float64       `json:"gross_cost_usd"`
	NetCostUSD          float64       `json:"net_cost_usd"`
	ElectricityCostUSD  float64       `json:"electricity_cost_usd"`
	HardwareProfile     string        `json:"hardware_profile"`
	Status              string        `json:"status"`
	GitBranch           string        `json:"git_branch"`
	IsSubagent          bool          `json:"is_subagent"`
	ParentSessionID     string        `json:"parent_session_id"`
	SubagentType        string        `json:"subagent_type"`
	Turns               []MessageTurn `json:"turns,omitempty"`
	SubagentRuns        []SubagentRun `json:"subagent_runs,omitempty"`
}
