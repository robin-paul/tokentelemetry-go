package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

type CodexParser struct{}

func NewCodexParser() *CodexParser {
	return &CodexParser{}
}

func (p *CodexParser) AgentName() string {
	return "codex"
}

func (p *CodexParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, ".codex/sessions") && strings.HasSuffix(lower, ".jsonl")
}

type codexLine struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Payload   struct {
		ID            string `json:"id"`
		Cwd           string `json:"cwd"`
		ModelProvider string `json:"model_provider"`
		ThreadSource  string `json:"thread_source"`
		ForkedFromID  string `json:"forked_from_id"`
		Content       string `json:"content"`
		Text          string `json:"text"`
		Message       string `json:"message"`
		Prompt        string `json:"prompt"`
		Source        *struct {
			Subagent *struct {
				ThreadSpawn *struct {
					ParentThreadID string `json:"parent_thread_id"`
					AgentRole      string `json:"agent_role"`
				} `json:"thread_spawn"`
			} `json:"subagent"`
		} `json:"source"`
		Type string `json:"type"` // for event_msg / response_item
		Info *struct {
			TotalTokenUsage *struct {
				InputTokens           int64 `json:"input_tokens"`
				CachedInputTokens     int64 `json:"cached_input_tokens"`
				OutputTokens          int64 `json:"output_tokens"`
				ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
				TotalTokens           int64 `json:"total_tokens"`
			} `json:"total_token_usage"`
		} `json:"info"`
		Name      string `json:"name"`      // function name
		Arguments string `json:"arguments"` // function args
	} `json:"payload"`
}

func (p *CodexParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:   "codex",
		Model:       "o3-mini",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	var firstTime, lastTime time.Time
	turnIndex := 0

	endOffset, err := ReadLines(r, startOffset, func(line []byte, offset int64) error {
		var item codexLine
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

		if item.Type == "session_meta" {
			if item.Payload.ID != "" {
				session.SessionID = item.Payload.ID
			}
			if item.Payload.ThreadSource == "subagent" || item.Payload.ForkedFromID != "" {
				session.IsSubagent = true
				if item.Payload.ForkedFromID != "" {
					session.ParentSessionID = item.Payload.ForkedFromID
				}
				if item.Payload.Source != nil && item.Payload.Source.Subagent != nil && item.Payload.Source.Subagent.ThreadSpawn != nil {
					session.ParentSessionID = item.Payload.Source.Subagent.ThreadSpawn.ParentThreadID
					session.SubagentType = item.Payload.Source.Subagent.ThreadSpawn.AgentRole
				}
			}
		}

		if item.Type == "event_msg" && item.Payload.Type == "token_count" && item.Payload.Info != nil && item.Payload.Info.TotalTokenUsage != nil {
			u := item.Payload.Info.TotalTokenUsage
			turnIndex++

			// Net prompt calculation: gross input minus cached tokens
			grossInput := u.InputTokens
			cachedInput := u.CachedInputTokens
			netInput := grossInput - cachedInput
			if netInput < 0 {
				netInput = 0
			}

			outputTokens := u.OutputTokens
			effort := ""
			if u.ReasoningOutputTokens > 0 {
				effort = "medium"
			}
			if u.ReasoningOutputTokens > 0 && u.TotalTokens > (grossInput+outputTokens) {
				outputTokens += u.ReasoningOutputTokens
			}

			turn := Turn{
				Index:           turnIndex,
				Timestamp:       ts,
				Role:            "assistant",
				Model:           session.Model,
				ReasoningEffort: effort,
				Usage: TokenUsage{
					InputTokens:     netInput,
					OutputTokens:    outputTokens,
					CacheReadTokens: cachedInput,
				},
				Tools:          make([]string, 0),
				ToolCalls:      make([]models.ToolCall, 0),
				RawPayloadJSON: string(line),
			}

			// Cumulative update: store latest state
			session.TotalUsage.InputTokens = netInput
			session.TotalUsage.OutputTokens = outputTokens
			session.TotalUsage.CacheReadTokens = cachedInput

			session.Turns = append(session.Turns, turn)
		}

		if item.Type == "response_item" && item.Payload.Name != "" {
			var argsMap map[string]interface{}
			if item.Payload.Arguments != "" {
				_ = json.Unmarshal([]byte(item.Payload.Arguments), &argsMap)
			}
			toolCall := models.ToolCall{
				Name:     item.Payload.Name,
				Args:     argsMap,
				ArgsJSON: item.Payload.Arguments,
			}
			if len(session.Turns) > 0 {
				lastTurn := &session.Turns[len(session.Turns)-1]
				lastTurn.Tools = append(lastTurn.Tools, item.Payload.Name)
				lastTurn.ToolCalls = append(lastTurn.ToolCalls, toolCall)
			} else {
				turnIndex++
				session.Turns = append(session.Turns, Turn{
					Index:          turnIndex,
					Timestamp:      ts,
					Role:           "assistant",
					Model:          session.Model,
					Tools:          []string{item.Payload.Name},
					ToolCalls:      []models.ToolCall{toolCall},
					RawPayloadJSON: string(line),
				})
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
	session.ID = "codex:" + session.SessionID
	session.StartTime = firstTime
	session.EndTime = lastTime
	if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
		session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
	}

	return session, endOffset, nil
}
