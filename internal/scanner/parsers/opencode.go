package parsers

import (
	"database/sql"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
	_ "modernc.org/sqlite"
)

type OpenCodeParser struct{}

func NewOpenCodeParser() *OpenCodeParser {
	return &OpenCodeParser{}
}

func (p *OpenCodeParser) AgentName() string {
	return "opencode"
}

func (p *OpenCodeParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, "opencode") && (strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".db"))
}

func (p *OpenCodeParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:   "opencode",
		Model:       "claude-3-7-sonnet",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	var firstTime, lastTime time.Time
	turnIndex := 0

	endOffset, err := ReadLines(r, startOffset, func(line []byte, offset int64) error {
		var item struct {
			Type      string `json:"type"`
			Role      string `json:"role"`
			Model     string `json:"model"`
			ModelID   string `json:"modelID"`
			Timestamp int64  `json:"timestamp"`
			Content   string `json:"content"`
			Text      string `json:"text"`
			Prompt    string `json:"prompt"`
			Tool      string `json:"tool"`
			Name      string `json:"name"`
			Tokens    *struct {
				Input  int64 `json:"input"`
				Output int64 `json:"output"`
				Cache  *struct {
					Read  int64 `json:"read"`
					Write int64 `json:"write"`
				} `json:"cache"`
			} `json:"tokens"`
			Data *struct {
				Type    string          `json:"type"`
				Name    string          `json:"name"`
				Tool    string          `json:"tool"`
				Content string          `json:"content"`
				Text    string          `json:"text"`
				Args    json.RawMessage `json:"args"`
				Tokens  *struct {
					Input  int64 `json:"input"`
					Output int64 `json:"output"`
					Cache  *struct {
						Read  int64 `json:"read"`
						Write int64 `json:"write"`
					} `json:"cache"`
				} `json:"tokens"`
			} `json:"data"`
		}

		if err := json.Unmarshal(line, &item); err != nil {
			return nil
		}

		ts := time.Now().UTC()
		if item.Timestamp > 0 {
			ts = time.UnixMilli(item.Timestamp).UTC()
		}
		if firstTime.IsZero() || ts.Before(firstTime) {
			firstTime = ts
		}
		if lastTime.IsZero() || ts.After(lastTime) {
			lastTime = ts
		}

		modelName := item.Model
		if modelName == "" {
			modelName = item.ModelID
		}
		if modelName != "" {
			session.Model = modelName
		}

		tokens := item.Tokens
		if tokens == nil && item.Data != nil && item.Data.Tokens != nil {
			tokens = item.Data.Tokens
		}

		var toolName string
		var toolArgs json.RawMessage
		if item.Data != nil && item.Data.Name != "" {
			toolName = item.Data.Name
			toolArgs = item.Data.Args
		} else if item.Data != nil && item.Data.Tool != "" {
			toolName = item.Data.Tool
			toolArgs = item.Data.Args
		} else if item.Name != "" {
			toolName = item.Name
		} else if item.Tool != "" {
			toolName = item.Tool
		}

		content := item.Content
		if content == "" {
			content = item.Text
		}
		if content == "" {
			content = item.Prompt
		}
		if content == "" && item.Data != nil {
			if item.Data.Content != "" {
				content = item.Data.Content
			} else if item.Data.Text != "" {
				content = item.Data.Text
			}
		}

		role := item.Role
		if role == "" {
			if item.Type == "user" {
				role = "user"
			} else {
				role = "assistant"
			}
		}

		if tokens != nil || content != "" || toolName != "" {
			turnIndex++
			var cacheRead, cacheWrite int64
			var inTok, outTok int64
			if tokens != nil {
				inTok = tokens.Input
				outTok = tokens.Output
				if tokens.Cache != nil {
					cacheRead = tokens.Cache.Read
					cacheWrite = tokens.Cache.Write
				}
			}

			tools := make([]string, 0)
			toolCalls := make([]models.ToolCall, 0)
			if toolName != "" {
				tools = append(tools, toolName)
				var argsMap map[string]interface{}
				if len(toolArgs) > 0 {
					_ = json.Unmarshal(toolArgs, &argsMap)
				}
				toolCalls = append(toolCalls, models.ToolCall{
					Name:     toolName,
					Args:     argsMap,
					ArgsJSON: string(toolArgs),
				})
			}

			turn := Turn{
				Index:          turnIndex,
				Timestamp:      ts,
				Role:           role,
				Model:          session.Model,
				Content:        content,
				Tools:          tools,
				ToolCalls:      toolCalls,
				RawPayloadJSON: string(line),
				Usage: TokenUsage{
					InputTokens:         inTok,
					OutputTokens:        outTok,
					CacheReadTokens:     cacheRead,
					CacheCreationTokens: cacheWrite,
				},
			}

			session.TotalUsage.InputTokens += inTok
			session.TotalUsage.OutputTokens += outTok
			session.TotalUsage.CacheReadTokens += cacheRead
			session.TotalUsage.CacheCreationTokens += cacheWrite

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
	session.ID = "opencode:" + session.SessionID
	session.StartTime = firstTime
	session.EndTime = lastTime
	if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
		session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
	}

	return session, endOffset, nil
}

// ParseDatabase parses OpenCode's SQLite database directly.
func (p *OpenCodeParser) ParseDatabase(dbPath string) ([]ParsedSession, error) {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, directory, title, time_created, time_updated, model, parent_id
		FROM session;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []ParsedSession
	for rows.Next() {
		var s ParsedSession
		var dir, title, model string
		var created, updated int64
		var parentID sql.NullString

		if err := rows.Scan(&s.SessionID, &dir, &title, &created, &updated, &model, &parentID); err != nil {
			continue
		}

		s.AgentName = "opencode"
		s.ID = "opencode:" + s.SessionID
		s.Model = model
		if s.Model == "" {
			s.Model = "claude-3-7-sonnet"
		}
		s.StartTime = time.UnixMilli(created).UTC()
		s.EndTime = time.UnixMilli(updated).UTC()
		if s.EndTime.After(s.StartTime) {
			s.DurationSeconds = s.EndTime.Sub(s.StartTime).Seconds()
		}
		if parentID.Valid && parentID.String != "" {
			s.IsSubagent = true
			s.ParentSessionID = parentID.String
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}
