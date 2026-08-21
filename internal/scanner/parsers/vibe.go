package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type VibeParser struct{}

func NewVibeParser() *VibeParser {
	return &VibeParser{}
}

func (p *VibeParser) AgentName() string {
	return "vibe"
}

func (p *VibeParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return (strings.Contains(lower, ".vibe/logs") || strings.Contains(lower, "vibe/logs")) && strings.HasSuffix(lower, ".json")
}

func (p *VibeParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, startOffset, err
	}
	endOffset := startOffset + int64(len(data))

	session := &ParsedSession{
		AgentName:   "vibe",
		Model:       "vibe-coder",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	var doc struct {
		ID       string `json:"id"`
		Model    string `json:"model"`
		Metadata struct {
			Model string `json:"model"`
			Stats *struct {
				SessionPromptTokens     int64 `json:"session_prompt_tokens"`
				SessionCompletionTokens int64 `json:"session_completion_tokens"`
				ContextTokens           int64 `json:"context_tokens"`
				SessionTotalLLMTokens   int64 `json:"session_total_llm_tokens"`
			} `json:"stats"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(data, &doc); err == nil {
		if doc.ID != "" {
			session.SessionID = doc.ID
		}
		if doc.Model != "" {
			session.Model = doc.Model
		} else if doc.Metadata.Model != "" {
			session.Model = doc.Metadata.Model
		}

		if doc.Metadata.Stats != nil {
			st := doc.Metadata.Stats
			session.TotalUsage.InputTokens = st.SessionPromptTokens
			session.TotalUsage.OutputTokens = st.SessionCompletionTokens
			session.TotalUsage.CacheReadTokens = st.ContextTokens

			session.Turns = append(session.Turns, Turn{
				Index:     1,
				Timestamp: time.Now().UTC(),
				Role:      "assistant",
				Model:     session.Model,
				Usage: TokenUsage{
					InputTokens:     st.SessionPromptTokens,
					OutputTokens:    st.SessionCompletionTokens,
					CacheReadTokens: st.ContextTokens,
				},
			})
		}
	}

	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "vibe:" + session.SessionID

	return session, endOffset, nil
}
