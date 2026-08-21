package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type GeminiParser struct{}

func NewGeminiParser() *GeminiParser {
	return &GeminiParser{}
}

func (p *GeminiParser) AgentName() string {
	return "gemini_cli"
}

func (p *GeminiParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	if strings.Contains(lower, "antigravity") {
		return false
	}
	return (strings.Contains(lower, ".gemini") || strings.Contains(lower, "gemini/")) &&
		(strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".jsonl"))
}

type geminiDoc struct {
	SessionID string `json:"sessionId"`
	Model     string `json:"model"`
	Turns     []struct {
		Role          string `json:"role"`
		Model         string `json:"model"`
		Timestamp     string `json:"timestamp"`
		UsageMetadata *struct {
			PromptTokenCount        int64 `json:"promptTokenCount"`
			CandidatesTokenCount    int64 `json:"candidatesTokenCount"`
			CachedContentTokenCount int64 `json:"cachedContentTokenCount"`
		} `json:"usageMetadata"`
		Parts []struct {
			Text         string `json:"text"`
			FunctionCall *struct {
				Name string `json:"name"`
			} `json:"functionCall"`
		} `json:"parts"`
	} `json:"turns"`
}

func (p *GeminiParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, startOffset, err
	}
	endOffset := startOffset + int64(len(data))

	session := &ParsedSession{
		AgentName:   "gemini_cli",
		Model:       "gemini-2.0-flash",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	var doc geminiDoc
	if err := json.Unmarshal(data, &doc); err == nil && len(doc.Turns) > 0 {
		if doc.SessionID != "" {
			session.SessionID = doc.SessionID
		}
		if doc.Model != "" {
			session.Model = doc.Model
		}

		var firstTime, lastTime time.Time
		for i, t := range doc.Turns {
			var ts time.Time
			if t.Timestamp != "" {
				parsed, err := time.Parse(time.RFC3339, t.Timestamp)
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

			modelName := t.Model
			if modelName == "" {
				modelName = session.Model
			}

			turn := Turn{
				Index:     i + 1,
				Timestamp: ts,
				Role:      t.Role,
				Model:     modelName,
				Tools:     make([]string, 0),
			}

			for _, pt := range t.Parts {
				if pt.FunctionCall != nil && pt.FunctionCall.Name != "" {
					turn.Tools = append(turn.Tools, pt.FunctionCall.Name)
				}
			}

			if t.UsageMetadata != nil {
				turn.Usage.InputTokens = t.UsageMetadata.PromptTokenCount
				turn.Usage.OutputTokens = t.UsageMetadata.CandidatesTokenCount
				turn.Usage.CacheReadTokens = t.UsageMetadata.CachedContentTokenCount

				session.TotalUsage.InputTokens += t.UsageMetadata.PromptTokenCount
				session.TotalUsage.OutputTokens += t.UsageMetadata.CandidatesTokenCount
				session.TotalUsage.CacheReadTokens += t.UsageMetadata.CachedContentTokenCount
			}

			session.Turns = append(session.Turns, turn)
		}

		session.StartTime = firstTime
		session.EndTime = lastTime
		if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
			session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
		}
	} else {
		// Fallback to line-by-line JSONL
		turnIndex := 0
		var firstTime, lastTime time.Time

		_, _ = ReadLines(strings.NewReader(string(data)), 0, func(line []byte, offset int64) error {
			var item struct {
				Role          string `json:"role"`
				Model         string `json:"model"`
				Timestamp     string `json:"timestamp"`
				UsageMetadata *struct {
					PromptTokenCount        int64 `json:"promptTokenCount"`
					CandidatesTokenCount    int64 `json:"candidatesTokenCount"`
					CachedContentTokenCount int64 `json:"cachedContentTokenCount"`
				} `json:"usageMetadata"`
			}
			if err := json.Unmarshal(line, &item); err != nil {
				return nil
			}

			turnIndex++
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

			turn := Turn{
				Index:     turnIndex,
				Timestamp: ts,
				Role:      item.Role,
				Model:     item.Model,
			}
			if item.Model != "" {
				session.Model = item.Model
			}
			if item.UsageMetadata != nil {
				turn.Usage.InputTokens = item.UsageMetadata.PromptTokenCount
				turn.Usage.OutputTokens = item.UsageMetadata.CandidatesTokenCount
				turn.Usage.CacheReadTokens = item.UsageMetadata.CachedContentTokenCount

				session.TotalUsage.InputTokens += item.UsageMetadata.PromptTokenCount
				session.TotalUsage.OutputTokens += item.UsageMetadata.CandidatesTokenCount
				session.TotalUsage.CacheReadTokens += item.UsageMetadata.CachedContentTokenCount
			}
			session.Turns = append(session.Turns, turn)
			return nil
		})

		session.StartTime = firstTime
		session.EndTime = lastTime
		if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
			session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
		}
	}

	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "gemini:" + session.SessionID

	return session, endOffset, nil
}
