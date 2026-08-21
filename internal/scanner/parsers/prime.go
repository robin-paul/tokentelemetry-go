package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PrimeParser struct{}

func NewPrimeParser() *PrimeParser {
	return &PrimeParser{}
}

func (p *PrimeParser) AgentName() string {
	return "prime"
}

func (p *PrimeParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, ".prime/sessions") && strings.HasSuffix(lower, ".jsonl")
}

func (p *PrimeParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:   "prime",
		Model:       "claude-3-5-sonnet",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	var firstTime, lastTime time.Time
	turnIndex := 0

	endOffset, err := ReadLines(r, startOffset, func(line []byte, offset int64) error {
		var item struct {
			ID             string `json:"id"`
			ParentID       string `json:"parentId"`
			Type           string `json:"type"`
			Model          string `json:"model"`
			Timestamp      string `json:"timestamp"`
			AggregateUsage *struct {
				InputTokens  int64 `json:"input_tokens"`
				OutputTokens int64 `json:"output_tokens"`
			} `json:"aggregateUsage"`
			Usage *struct {
				InputTokens         int64 `json:"input_tokens"`
				OutputTokens        int64 `json:"output_tokens"`
				CacheReadTokens     int64 `json:"cache_read_tokens"`
				CacheCreationTokens int64 `json:"cache_creation_tokens"`
			} `json:"usage"`
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

		if item.ID != "" && session.SessionID == "" {
			session.SessionID = item.ID
		}
		if item.ParentID != "" {
			session.IsSubagent = true
			session.ParentSessionID = item.ParentID
		}
		if item.Model != "" {
			session.Model = item.Model
		}

		if item.Usage != nil {
			turnIndex++
			u := item.Usage

			turn := Turn{
				Index:     turnIndex,
				Timestamp: ts,
				Role:      "assistant",
				Model:     session.Model,
				Usage: TokenUsage{
					InputTokens:         u.InputTokens,
					OutputTokens:        u.OutputTokens,
					CacheReadTokens:     u.CacheReadTokens,
					CacheCreationTokens: u.CacheCreationTokens,
				},
			}

			session.TotalUsage.InputTokens += u.InputTokens
			session.TotalUsage.OutputTokens += u.OutputTokens
			session.TotalUsage.CacheReadTokens += u.CacheReadTokens
			session.TotalUsage.CacheCreationTokens += u.CacheCreationTokens

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
	session.ID = "prime:" + session.SessionID
	session.StartTime = firstTime
	session.EndTime = lastTime
	if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
		session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
	}

	return session, endOffset, nil
}
