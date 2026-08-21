package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SmallCodeParser struct{}

func NewSmallCodeParser() *SmallCodeParser {
	return &SmallCodeParser{}
}

func (p *SmallCodeParser) AgentName() string {
	return "smallcode"
}

func (p *SmallCodeParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, ".smallcode/traces") && strings.HasSuffix(lower, ".json")
}

func (p *SmallCodeParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, startOffset, err
	}
	endOffset := startOffset + int64(len(data))

	session := &ParsedSession{
		AgentName:   "smallcode",
		Model:       "nemotron-3-nano:4b",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	var doc struct {
		ID        string `json:"id"`
		Model     string `json:"model"`
		Prompt    string `json:"prompt"`
		StartedAt string `json:"startedAt"`
		EndedAt   string `json:"endedAt"`
		Tokens    *struct {
			Prompt     int64 `json:"prompt"`
			Completion int64 `json:"completion"`
		} `json:"tokens"`
		Steps []struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"steps"`
	}

	if err := json.Unmarshal(data, &doc); err == nil {
		if doc.ID != "" {
			session.SessionID = doc.ID
		}
		if doc.Model != "" {
			session.Model = doc.Model
		}

		if doc.StartedAt != "" {
			if t, err := time.Parse(time.RFC3339, doc.StartedAt); err == nil {
				session.StartTime = t
			}
		}
		if doc.EndedAt != "" {
			if t, err := time.Parse(time.RFC3339, doc.EndedAt); err == nil {
				session.EndTime = t
			}
		}
		if !session.StartTime.IsZero() && !session.EndTime.IsZero() && session.EndTime.After(session.StartTime) {
			session.DurationSeconds = session.EndTime.Sub(session.StartTime).Seconds()
		}

		var tools []string
		for _, s := range doc.Steps {
			if s.Name != "" {
				tools = append(tools, s.Name)
			}
		}

		if doc.Tokens != nil {
			session.TotalUsage.InputTokens = doc.Tokens.Prompt
			session.TotalUsage.OutputTokens = doc.Tokens.Completion

			session.Turns = append(session.Turns, Turn{
				Index:     1,
				Timestamp: session.StartTime,
				Role:      "assistant",
				Model:     session.Model,
				Usage: TokenUsage{
					InputTokens:  doc.Tokens.Prompt,
					OutputTokens: doc.Tokens.Completion,
				},
				Tools: tools,
			})
		}
	}

	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "smallcode:" + session.SessionID

	return session, endOffset, nil
}
