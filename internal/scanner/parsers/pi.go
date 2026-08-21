package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PiParser struct{}

func NewPiParser() *PiParser {
	return &PiParser{}
}

func (p *PiParser) AgentName() string {
	return "pi"
}

func (p *PiParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return (strings.Contains(lower, ".pi/agent") || strings.Contains(lower, ".pi/sessions")) && strings.HasSuffix(lower, ".jsonl")
}

func (p *PiParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:   "pi",
		Model:       "zai-glm-4.7",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	var firstTime, lastTime time.Time
	turnIndex := 0

	endOffset, err := ReadLines(r, startOffset, func(line []byte, offset int64) error {
		var item struct {
			Type      string `json:"type"`
			ID        string `json:"id"`
			Cwd       string `json:"cwd"`
			Timestamp string `json:"timestamp"`
			ModelID   string `json:"modelId"`
			Provider  string `json:"provider"`
			Message   *struct {
				Role     string `json:"role"`
				Provider string `json:"provider"`
				Model    string `json:"model"`
				Usage    *struct {
					Input       int64 `json:"input"`
					Output      int64 `json:"output"`
					CacheRead   int64 `json:"cacheRead"`
					CacheWrite  int64 `json:"cacheWrite"`
					Reasoning   int64 `json:"reasoning"`
					TotalTokens int64 `json:"totalTokens"`
				} `json:"usage"`
				Content []struct {
					Type      string `json:"type"`
					Thinking  string `json:"thinking"`
					Name      string `json:"name"`
					ID        string `json:"id"`
				} `json:"content"`
			} `json:"message"`
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

		if item.Type == "session" {
			if item.ID != "" {
				session.SessionID = item.ID
			}
		}

		if item.Type == "model_change" && item.ModelID != "" {
			session.Model = item.ModelID
		}

		if item.Type == "message" && item.Message != nil {
			turnIndex++
			m := item.Message
			modelName := m.Model
			if modelName == "" {
				modelName = session.Model
			} else {
				session.Model = modelName
			}

			turn := Turn{
				Index:     turnIndex,
				Timestamp: ts,
				Role:      m.Role,
				Model:     modelName,
				Tools:     make([]string, 0),
			}

			for _, c := range m.Content {
				if c.Name != "" {
					turn.Tools = append(turn.Tools, c.Name)
				}
			}

			if m.Usage != nil {
				turn.Usage.InputTokens = m.Usage.Input
				turn.Usage.OutputTokens = m.Usage.Output
				turn.Usage.CacheReadTokens = m.Usage.CacheRead
				turn.Usage.CacheCreationTokens = m.Usage.CacheWrite

				session.TotalUsage.InputTokens += m.Usage.Input
				session.TotalUsage.OutputTokens += m.Usage.Output
				session.TotalUsage.CacheReadTokens += m.Usage.CacheRead
				session.TotalUsage.CacheCreationTokens += m.Usage.CacheWrite
			}

			session.Turns = append(session.Turns, turn)
		}

		return nil
	})

	if err != nil {
		return nil, endOffset, err
	}

	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "pi:" + session.SessionID
	session.StartTime = firstTime
	session.EndTime = lastTime
	if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
		session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
	}

	return session, endOffset, nil
}
