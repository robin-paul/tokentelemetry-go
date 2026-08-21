package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ClaudeParser struct{}

func NewClaudeParser() *ClaudeParser {
	return &ClaudeParser{}
}

func (p *ClaudeParser) AgentName() string {
	return "claude_code"
}

func (p *ClaudeParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, ".claude/projects") && (strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".json"))
}

type claudeLine struct {
	Type              string `json:"type"`
	SessionID         string `json:"sessionId"`
	Timestamp         string `json:"timestamp"`
	IsSidechain       bool   `json:"isSidechain"`
	AgentID           string `json:"agentId"`
	AttributionAgent  string `json:"attributionAgent"`
	ParentSessionID   string `json:"parentSessionId"`
	Message           *struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Usage *struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
			CacheCreation            *struct {
				Ephemeral1hInputTokens int64 `json:"ephemeral_1h_input_tokens"`
			} `json:"cache_creation"`
		} `json:"usage"`
		Content []struct {
			Type  string          `json:"type"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
			Text  string          `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

func (p *ClaudeParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:   "claude_code",
		ProjectName: "",
		Turns:       make([]Turn, 0),
		Status:      "completed",
	}

	var firstTime, lastTime time.Time
	turnIndex := 0

	endOffset, err := ReadLines(r, startOffset, func(line []byte, offset int64) error {
		var item claudeLine
		if err := json.Unmarshal(line, &item); err != nil {
			return nil
		}

		var ts time.Time
		if item.Timestamp != "" {
			t, err := time.Parse(time.RFC3339, item.Timestamp)
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

		if item.SessionID != "" && session.SessionID == "" {
			session.SessionID = item.SessionID
		}

		if item.IsSidechain || item.AgentID != "" || item.AttributionAgent != "" {
			session.IsSubagent = true
			if item.AttributionAgent != "" {
				session.SubagentType = item.AttributionAgent
			} else if item.AgentID != "" {
				session.SubagentType = item.AgentID
			}
		}
		if item.ParentSessionID != "" {
			session.ParentSessionID = item.ParentSessionID
		}

		if item.Type == "assistant" && item.Message != nil {
			turnIndex++
			turn := Turn{
				Index:     turnIndex,
				Timestamp: ts,
				Role:      "assistant",
				Model:     item.Message.Model,
				Tools:     make([]string, 0),
			}

			if item.Message.Model != "" {
				session.Model = item.Message.Model
			}

			if item.Message.Usage != nil {
				u := item.Message.Usage
				turn.Usage.InputTokens = u.InputTokens
				turn.Usage.OutputTokens = u.OutputTokens
				turn.Usage.CacheReadTokens = u.CacheReadInputTokens
				turn.Usage.CacheCreationTokens = u.CacheCreationInputTokens

				session.TotalUsage.InputTokens += u.InputTokens
				session.TotalUsage.OutputTokens += u.OutputTokens
				session.TotalUsage.CacheReadTokens += u.CacheReadInputTokens
				session.TotalUsage.CacheCreationTokens += u.CacheCreationInputTokens
			}

			for _, c := range item.Message.Content {
				if c.Type == "tool_use" && c.Name != "" {
					turn.Tools = append(turn.Tools, c.Name)
				}
			}

			session.Turns = append(session.Turns, turn)
		} else if item.Type == "user" {
			turnIndex++
			session.Turns = append(session.Turns, Turn{
				Index:     turnIndex,
				Timestamp: ts,
				Role:      "user",
			})
		}

		return nil
	})

	if err != nil {
		return nil, endOffset, err
	}

	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "claude:" + session.SessionID
	session.StartTime = firstTime
	session.EndTime = lastTime
	if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
		session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
	}

	return session, endOffset, nil
}
