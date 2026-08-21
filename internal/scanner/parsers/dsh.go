package parsers

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type DSHParser struct{}

func NewDSHParser() *DSHParser {
	return &DSHParser{}
}

func (p *DSHParser) AgentName() string {
	return "dsh"
}

func (p *DSHParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, ".dsh/sessions") &&
		(strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".jsonl.zstd") || strings.HasSuffix(lower, ".json"))
}

func (p *DSHParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:   "dsh",
		Model:       "zai-glm-4.7",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	seenTurnStep := make(map[string]bool)
	var firstTime, lastTime time.Time
	turnIndex := 0

	endOffset, err := ReadLines(r, startOffset, func(line []byte, offset int64) error {
		var event struct {
			Type          string `json:"type"`
			ID            string `json:"id"`
			CreatedAt     int64  `json:"createdAt"`
			Time          int64  `json:"time"`
			Origin        string `json:"origin"`
			ParentSession string `json:"parentSession"`
			Data          struct {
				Turn     int    `json:"turn"`
				Step     int    `json:"step"`
				Provider string `json:"provider"`
				Model    string `json:"model"`
				Chunk    *struct {
					Type  string `json:"type"`
					Usage *struct {
						InputTokens     int64 `json:"inputTokens"`
						OutputTokens    int64 `json:"outputTokens"`
						CacheReadTokens int64 `json:"cacheReadTokens"`
					} `json:"usage"`
				} `json:"chunk"`
				Usage *struct {
					InputTokens     int64 `json:"inputTokens"`
					OutputTokens    int64 `json:"outputTokens"`
					CacheReadTokens int64 `json:"cacheReadTokens"`
				} `json:"usage"`
				CallID    string `json:"callId"`
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"data"`
		}

		if err := json.Unmarshal(line, &event); err != nil {
			return nil
		}

		var ts time.Time
		if event.Time > 0 {
			ts = time.UnixMilli(event.Time).UTC()
		} else if event.CreatedAt > 0 {
			ts = time.UnixMilli(event.CreatedAt).UTC()
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

		if event.Type == "session" {
			if event.ID != "" {
				session.SessionID = event.ID
			}
			if event.Origin == "subagent" || event.ParentSession != "" {
				session.IsSubagent = true
				session.ParentSessionID = event.ParentSession
			}
		}

		if event.Type == "request/context" && event.Data.Model != "" {
			session.Model = event.Data.Model
			if event.Data.Provider != "" {
				session.Provider = event.Data.Provider
			}
		}

		// Handle chunk / message usage with deduplication by (turn, step)
		var usageIn, usageOut, usageCache int64
		hasUsage := false

		if event.Data.Chunk != nil && event.Data.Chunk.Usage != nil {
			usageIn = event.Data.Chunk.Usage.InputTokens
			usageOut = event.Data.Chunk.Usage.OutputTokens
			usageCache = event.Data.Chunk.Usage.CacheReadTokens
			hasUsage = true
		} else if event.Data.Usage != nil {
			usageIn = event.Data.Usage.InputTokens
			usageOut = event.Data.Usage.OutputTokens
			usageCache = event.Data.Usage.CacheReadTokens
			hasUsage = true
		}

		if hasUsage {
			key := fmt.Sprintf("%d:%d", event.Data.Turn, event.Data.Step)
			if !seenTurnStep[key] {
				seenTurnStep[key] = true
				turnIndex++

				turn := Turn{
					Index:     turnIndex,
					Timestamp: ts,
					Role:      "assistant",
					Model:     session.Model,
					Usage: TokenUsage{
						InputTokens:     usageIn,
						OutputTokens:    usageOut,
						CacheReadTokens: usageCache,
					},
					Tools: make([]string, 0),
				}

				session.TotalUsage.InputTokens += usageIn
				session.TotalUsage.OutputTokens += usageOut
				session.TotalUsage.CacheReadTokens += usageCache

				session.Turns = append(session.Turns, turn)
			}
		}

		if event.Type == "tool/call" && event.Data.Name != "" {
			if len(session.Turns) > 0 {
				session.Turns[len(session.Turns)-1].Tools = append(session.Turns[len(session.Turns)-1].Tools, event.Data.Name)
			}
		}

		return nil
	})

	if err != nil {
		return nil, endOffset, err
	}

	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "dsh:" + session.SessionID
	session.StartTime = firstTime
	session.EndTime = lastTime
	if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
		session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
	}

	return session, endOffset, nil
}
