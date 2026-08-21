package parsers

import (
	"database/sql"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type HermesParser struct{}

func NewHermesParser() *HermesParser {
	return &HermesParser{}
}

func (p *HermesParser) AgentName() string {
	return "hermes"
}

func (p *HermesParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, "hermes") &&
		(strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".db") || strings.HasSuffix(lower, ".log"))
}

func (p *HermesParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:   "hermes",
		Model:       "hermes-3-llama-3.1-405b",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	var firstTime, lastTime time.Time
	turnIndex := 0

	endOffset, err := ReadLines(r, startOffset, func(line []byte, offset int64) error {
		var item struct {
			Event            string  `json:"event"`
			SessionID        string  `json:"session_id"`
			Model            string  `json:"model"`
			Timestamp        string  `json:"timestamp"`
			InputTokens      int64   `json:"input_tokens"`
			OutputTokens     int64   `json:"output_tokens"`
			CacheReadTokens  int64   `json:"cache_read_tokens"`
			CacheWriteTokens int64   `json:"cache_write_tokens"`
			ReasoningTokens  int64   `json:"reasoning_tokens"`
			CostUSD          float64 `json:"cost_usd"`
			EndReason        string  `json:"end_reason"`
			ParentSessionID  string  `json:"parent_session_id"`
			Role             string  `json:"role"`
			Tools            []string `json:"tools"`
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

		if item.SessionID != "" {
			session.SessionID = item.SessionID
		}
		if item.Model != "" {
			session.Model = item.Model
		}
		if item.ParentSessionID != "" {
			session.IsSubagent = true
			session.ParentSessionID = item.ParentSessionID
		}
		if item.EndReason != "" {
			session.Status = item.EndReason
		}

		turnIndex++
		role := item.Role
		if role == "" {
			role = "assistant"
		}

		turn := Turn{
			Index:     turnIndex,
			Timestamp: ts,
			Role:      role,
			Model:     session.Model,
			Usage: TokenUsage{
				InputTokens:         item.InputTokens,
				OutputTokens:        item.OutputTokens,
				CacheReadTokens:     item.CacheReadTokens,
				CacheCreationTokens: item.CacheWriteTokens,
			},
			CostUSD: item.CostUSD,
			Tools:   item.Tools,
		}

		session.TotalUsage.InputTokens += item.InputTokens
		session.TotalUsage.OutputTokens += item.OutputTokens
		session.TotalUsage.CacheReadTokens += item.CacheReadTokens
		session.TotalUsage.CacheCreationTokens += item.CacheWriteTokens

		session.Turns = append(session.Turns, turn)
		return nil
	})

	if err != nil {
		return nil, endOffset, err
	}

	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "hermes:" + session.SessionID
	session.StartTime = firstTime
	session.EndTime = lastTime
	if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
		session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
	}

	return session, endOffset, nil
}

// ParseDatabase parses Hermes SQLite state.db directly.
func (p *HermesParser) ParseDatabase(dbPath string) ([]ParsedSession, error) {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, model, parent_session_id, started_at, ended_at,
		       input_tokens, output_tokens, cache_read_tokens, cache_write_tokens,
		       reasoning_tokens, actual_cost_usd, end_reason
		FROM sessions;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []ParsedSession
	for rows.Next() {
		var s ParsedSession
		var model, endReason string
		var parentID sql.NullString
		var startedAt, endedAt sql.NullInt64
		var inT, outT, readT, writeT, reasonT int64
		var cost sql.NullFloat64

		if err := rows.Scan(
			&s.SessionID, &model, &parentID, &startedAt, &endedAt,
			&inT, &outT, &readT, &writeT, &reasonT, &cost, &endReason,
		); err != nil {
			continue
		}

		s.AgentName = "hermes"
		s.ID = "hermes:" + s.SessionID
		s.Model = model
		if s.Model == "" {
			s.Model = "hermes-3-llama-3.1-405b"
		}
		if startedAt.Valid {
			s.StartTime = time.UnixMilli(startedAt.Int64).UTC()
		}
		if endedAt.Valid {
			s.EndTime = time.UnixMilli(endedAt.Int64).UTC()
			if s.EndTime.After(s.StartTime) {
				s.DurationSeconds = s.EndTime.Sub(s.StartTime).Seconds()
			}
		}
		s.TotalUsage = TokenUsage{
			InputTokens:         inT,
			OutputTokens:        outT,
			CacheReadTokens:     readT,
			CacheCreationTokens: writeT,
		}
		s.Status = endReason
		if parentID.Valid && parentID.String != "" {
			s.IsSubagent = true
			s.ParentSessionID = parentID.String
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}
