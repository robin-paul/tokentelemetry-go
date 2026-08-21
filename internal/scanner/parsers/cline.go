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

type ClineParser struct{}

func NewClineParser() *ClineParser {
	return &ClineParser{}
}

func (p *ClineParser) AgentName() string {
	return "cline"
}

func (p *ClineParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return (strings.Contains(lower, ".cline") || strings.Contains(lower, "claude-dev") || strings.Contains(lower, "taskhistory")) &&
		(strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".db"))
}

func (p *ClineParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, startOffset, err
	}
	endOffset := startOffset + int64(len(data))

	session := &ParsedSession{
		AgentName:   "cline",
		Model:       "claude-3-5-sonnet",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	// Try taskHistory.json array format
	var historyItems []struct {
		ID          string  `json:"id"`
		TS          int64   `json:"ts"`
		TotalCost   float64 `json:"totalCost"`
		TokensIn    int64   `json:"tokensIn"`
		TokensOut   int64   `json:"tokensOut"`
		CacheReads  int64   `json:"cacheReads"`
		CacheWrites int64   `json:"cacheWrites"`
		Model       string  `json:"model"`
	}

	if err := json.Unmarshal(data, &historyItems); err == nil && len(historyItems) > 0 {
		var firstTime, lastTime time.Time
		for i, item := range historyItems {
			ts := time.UnixMilli(item.TS).UTC()
			if ts.IsZero() || item.TS == 0 {
				ts = time.Now().UTC()
			}
			if firstTime.IsZero() || ts.Before(firstTime) {
				firstTime = ts
			}
			if lastTime.IsZero() || ts.After(lastTime) {
				lastTime = ts
			}

			modelName := item.Model
			if modelName != "" {
				session.Model = modelName
			}

			turn := Turn{
				Index:     i + 1,
				Timestamp: ts,
				Role:      "assistant",
				Model:     session.Model,
				Usage: TokenUsage{
					InputTokens:         item.TokensIn,
					OutputTokens:        item.TokensOut,
					CacheReadTokens:     item.CacheReads,
					CacheCreationTokens: item.CacheWrites,
				},
				CostUSD: item.TotalCost,
			}

			session.TotalUsage.InputTokens += item.TokensIn
			session.TotalUsage.OutputTokens += item.TokensOut
			session.TotalUsage.CacheReadTokens += item.CacheReads
			session.TotalUsage.CacheCreationTokens += item.CacheWrites

			session.Turns = append(session.Turns, turn)
		}

		session.StartTime = firstTime
		session.EndTime = lastTime
		if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
			session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
		}
	} else {
		// Try single JSON object
		var item struct {
			ID           string `json:"id"`
			SessionID    string `json:"session_id"`
			Model        string `json:"model"`
			MetadataJSON string `json:"metadata_json"`
			Usage        *struct {
				InputTokens      int64 `json:"inputTokens"`
				OutputTokens     int64 `json:"outputTokens"`
				CacheReadTokens  int64 `json:"cacheReadTokens"`
				CacheWriteTokens int64 `json:"cacheWriteTokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(data, &item); err == nil {
			if item.SessionID != "" {
				session.SessionID = item.SessionID
			} else if item.ID != "" {
				session.SessionID = item.ID
			}
			if item.Model != "" {
				session.Model = item.Model
			}

			if item.Usage != nil {
				session.TotalUsage.InputTokens = item.Usage.InputTokens
				session.TotalUsage.OutputTokens = item.Usage.OutputTokens
				session.TotalUsage.CacheReadTokens = item.Usage.CacheReadTokens
				session.TotalUsage.CacheCreationTokens = item.Usage.CacheWriteTokens
			} else if item.MetadataJSON != "" {
				var meta struct {
					Usage struct {
						InputTokens      int64 `json:"inputTokens"`
						OutputTokens     int64 `json:"outputTokens"`
						CacheReadTokens  int64 `json:"cacheReadTokens"`
						CacheWriteTokens int64 `json:"cacheWriteTokens"`
					} `json:"usage"`
				}
				if err := json.Unmarshal([]byte(item.MetadataJSON), &meta); err == nil {
					session.TotalUsage.InputTokens = meta.Usage.InputTokens
					session.TotalUsage.OutputTokens = meta.Usage.OutputTokens
					session.TotalUsage.CacheReadTokens = meta.Usage.CacheReadTokens
					session.TotalUsage.CacheCreationTokens = meta.Usage.CacheWriteTokens
				}
			}
		}
	}

	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "cline:" + session.SessionID

	return session, endOffset, nil
}

// ParseDatabase parses Cline SQLite sessions.db directly.
func (p *ClineParser) ParseDatabase(dbPath string) ([]ParsedSession, error) {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT session_id, model, metadata_json, is_subagent, parent_session_id
		FROM sessions;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []ParsedSession
	for rows.Next() {
		var s ParsedSession
		var model, metadataJSON string
		var isSubagentInt int
		var parentID sql.NullString

		if err := rows.Scan(&s.SessionID, &model, &metadataJSON, &isSubagentInt, &parentID); err != nil {
			continue
		}

		s.AgentName = "cline"
		s.ID = "cline:" + s.SessionID
		s.Model = model
		if s.Model == "" {
			s.Model = "claude-3-5-sonnet"
		}
		s.IsSubagent = isSubagentInt == 1
		if parentID.Valid {
			s.ParentSessionID = parentID.String
		}

		if metadataJSON != "" {
			var meta struct {
				Usage struct {
					InputTokens      int64 `json:"inputTokens"`
					OutputTokens     int64 `json:"outputTokens"`
					CacheReadTokens  int64 `json:"cacheReadTokens"`
					CacheWriteTokens int64 `json:"cacheWriteTokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(metadataJSON), &meta); err == nil {
				s.TotalUsage = TokenUsage{
					InputTokens:         meta.Usage.InputTokens,
					OutputTokens:        meta.Usage.OutputTokens,
					CacheReadTokens:     meta.Usage.CacheReadTokens,
					CacheCreationTokens: meta.Usage.CacheWriteTokens,
				}
			}
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}
