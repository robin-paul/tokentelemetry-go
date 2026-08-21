package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OllamaParser struct{}

func NewOllamaParser() *OllamaParser {
	return &OllamaParser{}
}

func (p *OllamaParser) AgentName() string {
	return "ollama"
}

func (p *OllamaParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return (strings.Contains(lower, "ollama") || strings.Contains(lower, "openai_compat") || strings.Contains(lower, "local-models")) &&
		(strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".log"))
}

func (p *OllamaParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, startOffset, err
	}
	endOffset := startOffset + int64(len(data))

	session := &ParsedSession{
		AgentName:       "ollama",
		Model:           "llama3:8b",
		Turns:           make([]Turn, 0),
		Status:          "completed",
		HardwareProfile: "m4_max",
		Endpoint:        "http://localhost:11434",
		TotalUsage:      TokenUsage{},
	}

	var doc struct {
		ID        string `json:"id"`
		Model     string `json:"model"`
		CreatedAt string `json:"created_at"`
		Usage     *struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
		PromptEvalCount int64 `json:"prompt_eval_count"`
		EvalCount       int64 `json:"eval_count"`
	}

	if err := json.Unmarshal(data, &doc); err == nil {
		if doc.ID != "" {
			session.SessionID = doc.ID
		}
		if doc.Model != "" {
			session.Model = doc.Model
		}
		if doc.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, doc.CreatedAt); err == nil {
				session.StartTime = t
				session.EndTime = t
			}
		}

		promptTokens := doc.PromptEvalCount
		evalTokens := doc.EvalCount
		if doc.Usage != nil {
			promptTokens = doc.Usage.PromptTokens
			evalTokens = doc.Usage.CompletionTokens
		}

		session.TotalUsage.InputTokens = promptTokens
		session.TotalUsage.OutputTokens = evalTokens

		session.Turns = append(session.Turns, Turn{
			Index:     1,
			Timestamp: time.Now().UTC(),
			Role:      "assistant",
			Model:     session.Model,
			Usage: TokenUsage{
				InputTokens:  promptTokens,
				OutputTokens: evalTokens,
			},
		})
	}

	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "ollama:" + session.SessionID

	return session, endOffset, nil
}
