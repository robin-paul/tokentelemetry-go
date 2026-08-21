package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

type AntigravityParser struct{}

func NewAntigravityParser() *AntigravityParser {
	return &AntigravityParser{}
}

func (p *AntigravityParser) AgentName() string {
	return "antigravity"
}

func (p *AntigravityParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return (strings.Contains(lower, ".gemini/antigravity") || strings.Contains(lower, "antigravity-cli") || strings.Contains(lower, "antigravity-ide")) &&
		(strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".json"))
}

type antigravityStep struct {
	StepIndex int    `json:"step_index"`
	Source    string `json:"source"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
	Content   string `json:"content"`
	Thinking  string `json:"thinking"`
	ToolCalls []struct {
		Name string          `json:"name"`
		Args json.RawMessage `json:"args"`
	} `json:"tool_calls"`
	Metrics *struct {
		InputTokens         int64 `json:"input_tokens"`
		OutputTokens        int64 `json:"output_tokens"`
		CacheReadTokens     int64 `json:"cache_read_tokens"`
		CacheCreationTokens int64 `json:"cache_creation_tokens"`
	} `json:"metrics"`
}

func (p *AntigravityParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:       "antigravity",
		Model:           "gemini-2.5-pro",
		Turns:           make([]Turn, 0),
		SubagentRuns:    make([]models.SubagentRun, 0),
		Status:          "completed",
		HardwareProfile: "default",
	}

	var firstTime, lastTime time.Time
	turnIndex := 0

	endOffset, err := ReadLines(r, startOffset, func(line []byte, offset int64) error {
		var step antigravityStep
		if err := json.Unmarshal(line, &step); err != nil {
			return nil
		}

		var ts time.Time
		if step.CreatedAt != "" {
			t, err := time.Parse(time.RFC3339, step.CreatedAt)
			if err == nil {
				ts = t
			}
		}
		if ts.IsZero() {
			ts = time.Now().UTC()
		}

		if firstTime.IsZero() || ts.Before(firstTime) {
			firstTime = ts
		}
		if lastTime.IsZero() || ts.After(lastTime) {
			lastTime = ts
		}

		turnIndex++
		role := "assistant"
		if step.Source == "USER_EXPLICIT" || step.Type == "USER_INPUT" {
			role = "user"
		}

		turn := Turn{
			Index:     turnIndex,
			Timestamp: ts,
			Role:      role,
			Model:     session.Model,
			Tools:     make([]string, 0),
		}

		for _, tc := range step.ToolCalls {
			if tc.Name != "" {
				turn.Tools = append(turn.Tools, tc.Name)
			}
		}

		// Token estimation or explicit metrics
		if step.Metrics != nil {
			turn.Usage.InputTokens = step.Metrics.InputTokens
			turn.Usage.OutputTokens = step.Metrics.OutputTokens
			turn.Usage.CacheReadTokens = step.Metrics.CacheReadTokens
			turn.Usage.CacheCreationTokens = step.Metrics.CacheCreationTokens
		} else {
			// Character-based token heuristics (len / 4)
			charCount := int64(len(step.Content) + len(step.Thinking))
			estTokens := charCount / 4
			if estTokens < 1 && len(step.Content) > 0 {
				estTokens = 1
			}

			if role == "assistant" {
				turn.Usage.OutputTokens = estTokens
			} else {
				turn.Usage.InputTokens = estTokens
			}
		}

		session.TotalUsage.InputTokens += turn.Usage.InputTokens
		session.TotalUsage.OutputTokens += turn.Usage.OutputTokens
		session.TotalUsage.CacheReadTokens += turn.Usage.CacheReadTokens
		session.TotalUsage.CacheCreationTokens += turn.Usage.CacheCreationTokens

		// Check for subagent invocation
		if step.Type == "INVOKE_SUBAGENT" || strings.Contains(step.Content, "Created subagents") {
			if strings.Contains(step.Content, "conversationId") {
				// Parse subagent ID
				idx := strings.Index(step.Content, "conversationId")
				rest := step.Content[idx:]
				parts := strings.Split(rest, "\"")
				if len(parts) >= 3 {
					childID := parts[2]
					session.SubagentRuns = append(session.SubagentRuns, models.SubagentRun{
						ID:              uuid.New().String(),
						ParentSessionID: session.ID,
						ChildSessionID:  childID,
						AgentType:       "subagent",
						CreatedAt:       ts,
					})
				}
			}
		}

		session.Turns = append(session.Turns, turn)
		return nil
	})

	if err != nil {
		return nil, endOffset, err
	}

	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "antigravity:" + session.SessionID
	session.StartTime = firstTime
	session.EndTime = lastTime
	if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
		session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
	}

	return session, endOffset, nil
}
