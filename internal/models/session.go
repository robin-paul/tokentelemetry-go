package models

import "time"

// TokenUsage holds raw and cache token counts.
type TokenUsage struct {
	InputTokens         int64 `json:"input_tokens"`
	OutputTokens        int64 `json:"output_tokens"`
	CacheReadTokens     int64 `json:"cache_read_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_tokens"`
}

// ToolCall represents a single tool invocation within a message turn.
type ToolCall struct {
	ID       string                 `json:"id,omitempty"`
	Name     string                 `json:"name"`
	Args     map[string]interface{} `json:"args,omitempty"`
	ArgsJSON string                 `json:"args_json,omitempty"`
}

// ToolResult represents the output from a tool execution.
type ToolResult struct {
	ID      string      `json:"id,omitempty"`
	Name    string      `json:"name,omitempty"`
	Content interface{} `json:"content,omitempty"`
	IsError bool        `json:"is_error,omitempty"`
}

// MessageTurn represents a single conversational turn with token metrics, rich content, thoughts, and tool invocations.
type MessageTurn struct {
	ID                  string       `json:"id"`
	SessionID           string       `json:"session_id"`
	TurnIndex           int          `json:"turn_index"`
	Timestamp           time.Time    `json:"timestamp"`
	Role                string       `json:"role"`
	ModelName           string       `json:"model_name"`
	Content             string       `json:"content,omitempty"`
	Thinking            string       `json:"thinking,omitempty"`
	ReasoningEffort     string       `json:"reasoning_effort,omitempty"`
	InputTokens         int64        `json:"input_tokens"`
	OutputTokens        int64        `json:"output_tokens"`
	CacheReadTokens     int64        `json:"cache_read_tokens"`
	CacheCreationTokens int64        `json:"cache_creation_tokens"`
	CostUSD             float64      `json:"cost_usd"`
	ToolsInvokedJSON    string       `json:"tools_invoked_json,omitempty"`
	ToolsInvoked        []string     `json:"tools_invoked,omitempty"`
	ToolCallsJSON       string       `json:"tool_calls_json,omitempty"`
	ToolCalls           []ToolCall   `json:"tool_calls,omitempty"`
	ToolResultsJSON     string       `json:"tool_results_json,omitempty"`
	ToolResults         []ToolResult `json:"tool_results,omitempty"`
	RawPayloadJSON      string       `json:"raw_payload_json,omitempty"`
}

// SubagentRun represents a subagent session spawned by a parent orchestrator.
type SubagentRun struct {
	ID              string                 `json:"id"`
	ParentSessionID string                 `json:"parent_session_id"`
	ChildSessionID  string                 `json:"child_session_id"`
	AgentType       string                 `json:"agent_type"`
	Tokens          int64                  `json:"tokens"`
	CostUSD         float64                `json:"cost_usd"`
	CreatedAt       time.Time              `json:"created_at"`
	Sandbox         map[string]interface{} `json:"sandbox,omitempty"`
}

// DSHMetrics holds latency and throughput breakdown derived from DSH transcripts.
type DSHMetrics struct {
	Turns           int      `json:"turns"`
	Steps           int      `json:"steps"`
	LLMMs           *float64 `json:"llm_ms,omitempty"`
	ToolMs          *float64 `json:"tool_ms,omitempty"`
	TTFTMsAvg       *float64 `json:"ttft_ms_avg,omitempty"`
	OutputTokPerSec *float64 `json:"output_tok_per_sec,omitempty"`
	CacheHitPct     *float64 `json:"cache_hit_pct,omitempty"`
}

// DSHLifecycleEvent represents a single plugin lifecycle transition record from dsh_lifecycle.jsonl.
type DSHLifecycleEvent struct {
	TS      int64   `json:"ts"`
	Plugin  string  `json:"plugin"`
	EntryID *string `json:"entry_id,omitempty"`
	UID     *int64  `json:"uid,omitempty"`
	From    string  `json:"from,omitempty"`
	To      string  `json:"to,omitempty"`
	Error   string  `json:"error,omitempty"`
}

// DSHPluginSummary holds aggregated transition metrics for a specific plugin.
type DSHPluginSummary struct {
	Plugin      string `json:"plugin"`
	Transitions int    `json:"transitions"`
	Failed      int    `json:"failed"`
	FinalState  string `json:"final_state,omitempty"`
}

// DSHLifecycleSummary holds rolled-up lifecycle transitions.
type DSHLifecycleSummary struct {
	Installed   bool               `json:"installed"`
	Correlation string             `json:"correlation"`
	Transitions int                `json:"transitions"`
	Failed      int                `json:"failed"`
	Reloads     int                `json:"reloads"`
	Unloads     int                `json:"unloads"`
	FirstTS     *int64             `json:"first_ts,omitempty"`
	LastTS      *int64             `json:"last_ts,omitempty"`
	Plugins     []DSHPluginSummary `json:"plugins,omitempty"`
	Events      []DSHLifecycleEvent `json:"events,omitempty"`
}

// DSHContext holds DSH-specific metadata, posture, metrics, and plugin lifecycle.
type DSHContext struct {
	Metrics        *DSHMetrics            `json:"metrics,omitempty"`
	AgentPreset    string                 `json:"agent_preset,omitempty"`
	PresetChain    []string               `json:"preset_chain,omitempty"`
	Sandbox        map[string]interface{} `json:"sandbox,omitempty"`
	Lifecycle      *DSHLifecycleSummary   `json:"lifecycle,omitempty"`
	ModelsUsed     []string               `json:"models_used,omitempty"`
	ProvidersUsed  []string               `json:"providers_used,omitempty"`
	SkillsCatalog  []string               `json:"skills_catalog,omitempty"`
	ToolsAvailable []string               `json:"tools_available,omitempty"`
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
	DSH                 *DSHContext   `json:"dsh,omitempty"`
}
