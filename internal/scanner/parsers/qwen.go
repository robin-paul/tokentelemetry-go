package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type QwenParser struct{}

func NewQwenParser() *QwenParser {
	return &QwenParser{}
}

func (p *QwenParser) AgentName() string {
	return "qwen"
}

func (p *QwenParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, ".qwen/projects") && strings.HasSuffix(lower, ".jsonl")
}

func (p *QwenParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:   "qwen",
		Model:       "qwen-2.5-coder-32b-instruct",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	var firstTime, lastTime time.Time
	turnIndex := 0

	endOffset, err := ReadLines(r, startOffset, func(line []byte, offset int64) error {
		var item struct {
			Role      string `json:"role"`
			Model     string `json:"model"`
			Timestamp string `json:"timestamp"`
			Usage     *struct {
				InputTokens              int64 `json:"input_tokens"`
				OutputTokens             int64 `json:"output_tokens"`
				CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
				CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
			} `json:"usage"`
			ToolCalls []struct {
				Name string `json:"name"`
			} `json:"tool_calls"`
		}

		if err := json.Unmarshal(line, &item); err != nil {
			return nil
		}

		var ts time.Time
		if item.Timestamp != "" {
			parsed, err := time.Parse(time.RFC3339, item.Timestamp)
			if err == nil {
				ts = parsed
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

		if item.Model != "" {
			session.Model = item.Model
		}

		turnIndex++
		role := item.Role
		if role == "" {
			role = "assistant"
		}

		turn := Turn{
			Index:     turnIndex,
			Timestamp: ts,
			Role:      role,
			Model:     session.Model,
			Tools:     make([]string, 0),
		}

		for _, tc := range item.ToolCalls {
			if tc.Name != "" {
				turn.Tools = append(turn.Tools, tc.Name)
			}
		}

		if item.Usage != nil {
			u := item.Usage
			turn.Usage.InputTokens = u.InputTokens
			turn.Usage.OutputTokens = u.OutputTokens
			turn.Usage.CacheReadTokens = u.CacheReadInputTokens
			turn.Usage.CacheCreationTokens = u.CacheCreationInputTokens

			session.TotalUsage.InputTokens += u.InputTokens
			session.TotalUsage.OutputTokens += u.OutputTokens
			session.TotalUsage.CacheReadTokens += u.CacheReadInputTokens
			session.TotalUsage.CacheCreationTokens += u.CacheCreationInputTokens
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
	session.ID = "qwen:" + session.SessionID
	session.StartTime = firstTime
	session.EndTime = lastTime
	if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
		session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
	}

	return session, endOffset, nil
}
