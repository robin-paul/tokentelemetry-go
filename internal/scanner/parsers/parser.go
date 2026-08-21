package parsers

import (
	"io"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// TokenUsage alias for convenience
type TokenUsage = models.TokenUsage

// Turn represents an individual message turn parsed from a transcript.
type Turn struct {
	Index        int        `json:"index"`
	Timestamp    time.Time  `json:"timestamp"`
	Role         string     `json:"role"`
	Model        string     `json:"model"`
	Usage        TokenUsage `json:"usage"`
	Tools        []string   `json:"tools"`
	CostUSD      float64    `json:"cost_usd,omitempty"`
}

// ParsedSession encapsulates the full extracted telemetry for a single session.
type ParsedSession struct {
	ID              string               `json:"id"`
	SessionID       string               `json:"session_id"`
	AgentName       string               `json:"agent_name"`
	ProjectName     string               `json:"project_name"`
	FilePath        string               `json:"file_path"`
	StartTime       time.Time            `json:"start_time"`
	EndTime         time.Time            `json:"end_time"`
	DurationSeconds float64              `json:"duration_seconds"`
	Model           string               `json:"model"`
	Provider        string               `json:"provider,omitempty"`
	TotalUsage      TokenUsage           `json:"total_usage"`
	Turns           []Turn               `json:"turns"`
	SubagentRuns    []models.SubagentRun `json:"subagent_runs,omitempty"`
	IsSubagent      bool                 `json:"is_subagent"`
	ParentSessionID string               `json:"parent_session_id"`
	SubagentType    string               `json:"subagent_type"`
	GitBranch       string               `json:"git_branch"`
	Status          string               `json:"status"`
	HardwareProfile string               `json:"hardware_profile"`
	Endpoint        string               `json:"endpoint,omitempty"`
}

// AgentParser is the standard interface implemented by all 18+ agent parsers.
type AgentParser interface {
	AgentName() string
	Detect(filePath string) bool
	Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error)
}
