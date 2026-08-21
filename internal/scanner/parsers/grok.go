package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type GrokParser struct{}

func NewGrokParser() *GrokParser {
	return &GrokParser{}
}

func (p *GrokParser) AgentName() string {
	return "grok"
}

func (p *GrokParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, ".grok/sessions") && (strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".jsonl"))
}

func (p *GrokParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, startOffset, err
	}
	endOffset := startOffset + int64(len(data))

	session := &ParsedSession{
		AgentName:   "grok",
		Model:       "grok-build",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	// 1. Check if summary.json or signals.json shape
	var summaryDoc struct {
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
		GeneratedTitle string `json:"generated_title"`
		CurrentModelID string `json:"current_model_id"`
		NumMessages    int    `json:"num_messages"`
		AgentName      string `json:"agent_name"`
		Info           *struct {
			Cwd string `json:"cwd"`
		} `json:"info"`
	}

	var signalsDoc struct {
		ContextTokensUsed int64    `json:"contextTokensUsed"`
		ToolsUsed         []string `json:"toolsUsed"`
		ModelsUsed        []string `json:"modelsUsed"`
	}

	if err := json.Unmarshal(data, &summaryDoc); err == nil && summaryDoc.CreatedAt != "" {
		if summaryDoc.CurrentModelID != "" {
			session.Model = summaryDoc.CurrentModelID
		}
		if t, err := time.Parse(time.RFC3339, summaryDoc.CreatedAt); err == nil {
			session.StartTime = t
		}
		if t, err := time.Parse(time.RFC3339, summaryDoc.UpdatedAt); err == nil {
			session.EndTime = t
		}
		if !session.StartTime.IsZero() && !session.EndTime.IsZero() && session.EndTime.After(session.StartTime) {
			session.DurationSeconds = session.EndTime.Sub(session.StartTime).Seconds()
		}
	} else if err := json.Unmarshal(data, &signalsDoc); err == nil && (signalsDoc.ContextTokensUsed > 0 || len(signalsDoc.ToolsUsed) > 0) {
		session.TotalUsage.InputTokens = signalsDoc.ContextTokensUsed
		if len(signalsDoc.ModelsUsed) > 0 {
			session.Model = signalsDoc.ModelsUsed[0]
		}
		session.Turns = append(session.Turns, Turn{
			Index:     1,
			Timestamp: time.Now().UTC(),
			Role:      "assistant",
			Model:     session.Model,
			Usage: TokenUsage{
				InputTokens: signalsDoc.ContextTokensUsed,
			},
			Tools: signalsDoc.ToolsUsed,
		})
	} else {
		// Try line-by-line updates.jsonl
		_, _ = ReadLines(strings.NewReader(string(data)), 0, func(line []byte, offset int64) error {
			var update struct {
				Params struct {
					Meta struct {
						TotalTokens int64 `json:"totalTokens"`
					} `json:"_meta"`
				} `json:"params"`
			}
			if err := json.Unmarshal(line, &update); err == nil && update.Params.Meta.TotalTokens > session.TotalUsage.InputTokens {
				session.TotalUsage.InputTokens = update.Params.Meta.TotalTokens
			}
			return nil
		})
	}

	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "grok:" + session.SessionID

	return session, endOffset, nil
}
