package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

type CursorParser struct{}

func NewCursorParser() *CursorParser {
	return &CursorParser{}
}

func (p *CursorParser) AgentName() string {
	return "cursor"
}

func (p *CursorParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, ".cursor/projects") && (strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".json"))
}

type cursorLine struct {
	Role      string `json:"role"`
	Timestamp string `json:"timestamp"`
	Text      string `json:"text"`
	Content   string `json:"content"`
	Message   *struct {
		Model string `json:"model"`
		Usage *struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
		Content []struct {
			Type  string          `json:"type"`
			Name  string          `json:"name"`
			Text  string          `json:"text"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	} `json:"message"`
}

func (p *CursorParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:   "cursor",
		Model:       "claude-3-5-sonnet",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	var firstTime, lastTime time.Time
	turnIndex := 0

	endOffset, err := ReadLines(r, startOffset, func(line []byte, offset int64) error {
		var item cursorLine
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

		turnIndex++
		role := item.Role
		if role == "" {
			role = "assistant"
		}

		turnContent := item.Content
		if turnContent == "" {
			turnContent = item.Text
		}

		turn := Turn{
			Index:          turnIndex,
			Timestamp:      ts,
			Role:           role,
			Model:          session.Model,
			Content:        turnContent,
			Tools:          make([]string, 0),
			ToolCalls:      make([]models.ToolCall, 0),
			RawPayloadJSON: string(line),
		}

		if item.Message != nil {
			if item.Message.Model != "" {
				session.Model = item.Message.Model
				turn.Model = item.Message.Model
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

			var textParts []string
			for _, c := range item.Message.Content {
				if c.Text != "" {
					textParts = append(textParts, c.Text)
				}
				if c.Name != "" {
					turn.Tools = append(turn.Tools, c.Name)
					var argsMap map[string]interface{}
					if len(c.Input) > 0 {
						_ = json.Unmarshal(c.Input, &argsMap)
					}
					turn.ToolCalls = append(turn.ToolCalls, models.ToolCall{
						Name:     c.Name,
						Args:     argsMap,
						ArgsJSON: string(c.Input),
					})
				}
			}
			if len(textParts) > 0 && turn.Content == "" {
				turn.Content = strings.Join(textParts, "\n\n")
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
	session.ID = "cursor:" + session.SessionID
	session.StartTime = firstTime
	session.EndTime = lastTime
	if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
		session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
	}

	return session, endOffset, nil
}
