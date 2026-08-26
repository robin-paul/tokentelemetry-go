package parsers

import (
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

var reAntigravityUserSettings = regexp.MustCompile(`(?i)<USER_SETTINGS_CHANGE>[\s\S]*?Model Selection[^\n\r]*?(?:from\s+\S+\s+to|to)\s+([^\n\r<]+?)(?:\.\s|\.\n|\.\r|\.<\/|\n|\r|<\/|$)`)

func extractAntigravityModel(content string) string {
	if matches := reAntigravityUserSettings.FindStringSubmatch(content); len(matches) > 1 {
		raw := strings.TrimSpace(matches[1])
		return normalizeAntigravityModelName(raw)
	}
	return ""
}

func normalizeAntigravityModelName(raw string) string {
	if idx := strings.Index(raw, "("); idx != -1 {
		raw = raw[:idx]
	}
	raw = strings.TrimSpace(raw)
	lower := strings.ToLower(raw)

	switch {
	case strings.Contains(lower, "3.7") && strings.Contains(lower, "flash"):
		return "gemini-3.7-flash"
	case strings.Contains(lower, "3.6") && strings.Contains(lower, "flash"):
		return "gemini-3.6-flash"
	case strings.Contains(lower, "3.7") && strings.Contains(lower, "pro"):
		return "gemini-3.7-pro"
	case strings.Contains(lower, "3.1") && strings.Contains(lower, "flash"):
		return "gemini-3.1-flash"
	case strings.Contains(lower, "3.1") && strings.Contains(lower, "pro"):
		return "gemini-3.1-pro"
	case strings.Contains(lower, "3") && strings.Contains(lower, "flash"):
		return "gemini-3-flash"
	case strings.Contains(lower, "3") && strings.Contains(lower, "pro"):
		return "gemini-3-pro"
	case strings.Contains(lower, "2.5") && strings.Contains(lower, "pro"):
		return "gemini-2.5-pro"
	case strings.Contains(lower, "2.5") && strings.Contains(lower, "flash"):
		return "gemini-2.5-flash"
	case strings.Contains(lower, "2.0") && strings.Contains(lower, "flash"):
		return "gemini-2.0-flash"
	case strings.Contains(lower, "claude") && strings.Contains(lower, "3.7") && strings.Contains(lower, "sonnet"):
		return "claude-3-7-sonnet"
	case strings.Contains(lower, "claude") && strings.Contains(lower, "3.5") && strings.Contains(lower, "sonnet"):
		return "claude-3-5-sonnet"
	case strings.Contains(lower, "claude") && strings.Contains(lower, "3.5") && strings.Contains(lower, "haiku"):
		return "claude-3-5-haiku"
	case strings.Contains(lower, "gpt-4o-mini"):
		return "gpt-4o-mini"
	case strings.Contains(lower, "gpt-4o"):
		return "gpt-4o"
	default:
		cleaned := strings.ReplaceAll(lower, " ", "-")
		if cleaned != "" {
			return cleaned
		}
		return "gemini-3.7-flash"
	}
}

type AntigravityParser struct{}

func NewAntigravityParser() *AntigravityParser {
	return &AntigravityParser{}
}

func (p *AntigravityParser) AgentName() string {
	return "antigravity"
}

func (p *AntigravityParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	if !strings.Contains(lower, ".gemini/antigravity") && !strings.Contains(lower, "antigravity-cli") && !strings.Contains(lower, "antigravity-ide") {
		return false
	}
	// Avoid duplicate or internal files: chunks, messages, tasks, caches, metadata, config
	if strings.Contains(lower, "transcript_full.jsonl") ||
		strings.Contains(lower, "/chunks/") ||
		strings.Contains(lower, "\\chunks\\") ||
		strings.Contains(lower, "/messages/") ||
		strings.Contains(lower, "\\messages\\") ||
		strings.Contains(lower, "/tasks/") ||
		strings.Contains(lower, "\\tasks\\") ||
		strings.HasSuffix(lower, "tokens_cache.json") ||
		strings.HasSuffix(lower, ".metadata.json") ||
		strings.HasSuffix(lower, "mcp_config.json") {
		return false
	}
	return strings.HasSuffix(lower, "transcript.jsonl") || strings.HasSuffix(lower, ".jsonl")
}

type antigravityUsage struct {
	InputTokens         int64 `json:"input_tokens"`
	OutputTokens        int64 `json:"output_tokens"`
	CacheReadTokens     int64 `json:"cache_read_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_tokens"`
	PromptTokens        int64 `json:"prompt_tokens"`
	CompletionTokens    int64 `json:"completion_tokens"`
	CachedTokens        int64 `json:"cached_tokens"`
}

type antigravityStep struct {
	StepIndex  int               `json:"step_index"`
	Source     string            `json:"source"`
	Type       string            `json:"type"`
	CreatedAt  string            `json:"created_at"`
	Content    string            `json:"content"`
	Thinking   string            `json:"thinking"`
	ToolCalls  []struct {
		Name string          `json:"name"`
		Args json.RawMessage `json:"args"`
	} `json:"tool_calls"`
	Metrics    *antigravityUsage `json:"metrics"`
	TokenUsage *antigravityUsage `json:"token_usage"`
	Usage      *antigravityUsage `json:"usage"`
}

func (p *AntigravityParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:       "antigravity",
		Model:           "gemini-3.7-flash",
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

		if detected := extractAntigravityModel(step.Content); detected != "" {
			session.Model = detected
			for i := range session.Turns {
				session.Turns[i].Model = detected
			}
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
			Index:          turnIndex,
			Timestamp:      ts,
			Role:           role,
			Model:          session.Model,
			Content:        step.Content,
			Thinking:       step.Thinking,
			Tools:          make([]string, 0),
			ToolCalls:      make([]models.ToolCall, 0),
			RawPayloadJSON: string(line),
		}

		if step.Thinking != "" {
			turn.ReasoningEffort = "high"
		}

		for _, tc := range step.ToolCalls {
			if tc.Name != "" {
				turn.Tools = append(turn.Tools, tc.Name)
			}
			var argsMap map[string]interface{}
			if len(tc.Args) > 0 {
				_ = json.Unmarshal(tc.Args, &argsMap)
			}
			turn.ToolCalls = append(turn.ToolCalls, models.ToolCall{
				Name:     tc.Name,
				Args:     argsMap,
				ArgsJSON: string(tc.Args),
			})
		}

		// Token estimation or explicit metrics
		u := step.Metrics
		if u == nil {
			u = step.TokenUsage
		}
		if u == nil {
			u = step.Usage
		}

		if u != nil {
			in := u.InputTokens
			if in == 0 {
				in = u.PromptTokens
			}
			out := u.OutputTokens
			if out == 0 {
				out = u.CompletionTokens
			}
			cached := u.CacheReadTokens
			if cached == 0 {
				cached = u.CachedTokens
			}
			turn.Usage.InputTokens = in
			turn.Usage.OutputTokens = out
			turn.Usage.CacheReadTokens = cached
			turn.Usage.CacheCreationTokens = u.CacheCreationTokens
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
