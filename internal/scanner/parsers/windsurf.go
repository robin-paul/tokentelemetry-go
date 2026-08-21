package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type WindsurfParser struct{}

func NewWindsurfParser() *WindsurfParser {
	return &WindsurfParser{}
}

func (p *WindsurfParser) AgentName() string {
	return "windsurf"
}

func (p *WindsurfParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return (strings.Contains(lower, "windsurf") || strings.Contains(lower, "codeium") || strings.Contains(lower, "cascade")) &&
		(strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".log"))
}

func (p *WindsurfParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, startOffset, err
	}
	endOffset := startOffset + int64(len(data))

	session := &ParsedSession{
		AgentName:   "windsurf",
		Model:       "claude-3-5-sonnet",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	var doc struct {
		SessionID string `json:"sessionId"`
		Model     string `json:"model"`
		Turns     []struct {
			Role         string   `json:"role"`
			Model        string   `json:"model"`
			InputTokens  int64    `json:"input_tokens"`
			OutputTokens int64    `json:"output_tokens"`
			Tools        []string `json:"tools"`
		} `json:"turns"`
	}

	if err := json.Unmarshal(data, &doc); err == nil && len(doc.Turns) > 0 {
		if doc.SessionID != "" {
			session.SessionID = doc.SessionID
		}
		if doc.Model != "" {
			session.Model = doc.Model
		}

		for i, t := range doc.Turns {
			modelName := t.Model
			if modelName == "" {
				modelName = session.Model
			}

			turn := Turn{
				Index:     i + 1,
				Timestamp: time.Now().UTC(),
				Role:      t.Role,
				Model:     modelName,
				Usage: TokenUsage{
					InputTokens:  t.InputTokens,
					OutputTokens: t.OutputTokens,
				},
				Tools: t.Tools,
			}

			session.TotalUsage.InputTokens += t.InputTokens
			session.TotalUsage.OutputTokens += t.OutputTokens
			session.Turns = append(session.Turns, turn)
		}
	} else {
		// Fallback to line-by-line
		turnIndex := 0
		_, _ = ReadLines(strings.NewReader(string(data)), 0, func(line []byte, offset int64) error {
			var item struct {
				Role         string `json:"role"`
				Model        string `json:"model"`
				InputTokens  int64  `json:"input_tokens"`
				OutputTokens int64  `json:"output_tokens"`
			}
			if err := json.Unmarshal(line, &item); err == nil && (item.InputTokens > 0 || item.OutputTokens > 0) {
				turnIndex++
				if item.Model != "" {
					session.Model = item.Model
				}
				session.TotalUsage.InputTokens += item.InputTokens
				session.TotalUsage.OutputTokens += item.OutputTokens
				session.Turns = append(session.Turns, Turn{
					Index:     turnIndex,
					Timestamp: time.Now().UTC(),
					Role:      item.Role,
					Model:     session.Model,
					Usage: TokenUsage{
						InputTokens:  item.InputTokens,
						OutputTokens: item.OutputTokens,
					},
				})
			}
			return nil
		})
	}

	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "windsurf:" + session.SessionID

	return session, endOffset, nil
}
