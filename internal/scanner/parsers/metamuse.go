package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type MetaMuseParser struct{}

func NewMetaMuseParser() *MetaMuseParser {
	return &MetaMuseParser{}
}

func (p *MetaMuseParser) AgentName() string {
	return "muse"
}

func (p *MetaMuseParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, ".muse/sessions") && strings.HasSuffix(lower, ".jsonl")
}

func (p *MetaMuseParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:   "muse",
		Model:       "llama-3.3-70b-versatile",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	var firstTime, lastTime time.Time
	turnIndex := 0

	endOffset, err := ReadLines(r, startOffset, func(line []byte, offset int64) error {
		var item struct {
			RecordedAt int64 `json:"recorded_at"`
			Payload    struct {
				Event struct {
					Model string `json:"model"`
					Usage *struct {
						InputTokens      int64 `json:"input_tokens"`
						OutputTokens     int64 `json:"output_tokens"`
						CacheReadTokens  int64 `json:"cache_read_tokens"`
						CacheWriteTokens int64 `json:"cache_write_tokens"`
						ReasoningTokens  int64 `json:"reasoning_tokens"`
					} `json:"usage"`
					ChildSessionLogPath string `json:"child_session_log_path"`
				} `json:"event"`
			} `json:"payload"`
		}

		if err := json.Unmarshal(line, &item); err != nil {
			return nil
		}

		var ts time.Time
		if item.RecordedAt > 0 {
			ts = time.UnixMicro(item.RecordedAt).UTC()
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

		ev := item.Payload.Event
		if ev.Model != "" {
			session.Model = ev.Model
		}

		if ev.Usage != nil {
			turnIndex++
			u := ev.Usage

			turn := Turn{
				Index:     turnIndex,
				Timestamp: ts,
				Role:      "assistant",
				Model:     session.Model,
				Usage: TokenUsage{
					InputTokens:         u.InputTokens,
					OutputTokens:        u.OutputTokens,
					CacheReadTokens:     u.CacheReadTokens,
					CacheCreationTokens: u.CacheWriteTokens,
				},
			}

			session.TotalUsage.InputTokens += u.InputTokens
			session.TotalUsage.OutputTokens += u.OutputTokens
			session.TotalUsage.CacheReadTokens += u.CacheReadTokens
			session.TotalUsage.CacheCreationTokens += u.CacheWriteTokens

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
	session.ID = "muse:" + session.SessionID
	session.StartTime = firstTime
	session.EndTime = lastTime
	if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
		session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
	}

	return session, endOffset, nil
}
