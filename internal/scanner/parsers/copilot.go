package parsers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CopilotParser struct{}

func NewCopilotParser() *CopilotParser {
	return &CopilotParser{}
}

func (p *CopilotParser) AgentName() string {
	return "copilot"
}

func (p *CopilotParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return (strings.Contains(lower, "copilot") || strings.Contains(lower, "chatsessions")) &&
		(strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".log"))
}

func (p *CopilotParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, startOffset, err
	}
	endOffset := startOffset + int64(len(data))

	session := &ParsedSession{
		AgentName:   "copilot",
		Model:       "gpt-4o",
		Turns:       make([]Turn, 0),
		Status:      "completed",
		TotalUsage:  TokenUsage{},
	}

	// 1. Try VS Code Chat JSON format
	var vsDoc struct {
		Version      int   `json:"version"`
		CreationDate int64 `json:"creationDate"`
		Requests     []struct {
			ModelID          string `json:"modelId"`
			Timestamp        int64  `json:"timestamp"`
			CompletionTokens int64  `json:"completionTokens"`
			Message          struct {
				Text string `json:"text"`
			} `json:"message"`
		} `json:"requests"`
	}

	if err := json.Unmarshal(data, &vsDoc); err == nil && len(vsDoc.Requests) > 0 {
		var firstTime, lastTime time.Time
		for i, req := range vsDoc.Requests {
			ts := time.UnixMilli(req.Timestamp).UTC()
			if ts.IsZero() || req.Timestamp == 0 {
				ts = time.Now().UTC()
			}
			if firstTime.IsZero() || ts.Before(firstTime) {
				firstTime = ts
			}
			if lastTime.IsZero() || ts.After(lastTime) {
				lastTime = ts
			}

			modelName := req.ModelID
			if modelName == "" {
				modelName = session.Model
			} else {
				session.Model = modelName
			}

			inputTokens := int64(len(req.Message.Text) / 4)
			if inputTokens < 1 && len(req.Message.Text) > 0 {
				inputTokens = 1
			}

			turn := Turn{
				Index:     i + 1,
				Timestamp: ts,
				Role:      "assistant",
				Model:     modelName,
				Usage: TokenUsage{
					InputTokens:  inputTokens,
					OutputTokens: req.CompletionTokens,
				},
			}

			session.TotalUsage.InputTokens += inputTokens
			session.TotalUsage.OutputTokens += req.CompletionTokens
			session.Turns = append(session.Turns, turn)
		}

		session.StartTime = firstTime
		session.EndTime = lastTime
		if !firstTime.IsZero() && !lastTime.IsZero() && lastTime.After(firstTime) {
			session.DurationSeconds = lastTime.Sub(firstTime).Seconds()
		}
	} else {
		// 2. Line-by-line Copilot CLI events
		var firstTime, lastTime time.Time
		turnIndex := 0

		_, _ = ReadLines(strings.NewReader(string(data)), 0, func(line []byte, offset int64) error {
			var event struct {
				Type string `json:"type"`
				Data struct {
					Context *struct {
						Cwd       string `json:"cwd"`
						StartTime string `json:"startTime"`
					} `json:"context"`
					Model        string `json:"model"`
					OutputTokens int64  `json:"outputTokens"`
					ModelMetrics map[string]struct {
						Usage struct {
							InputTokens      int64 `json:"inputTokens"`
							OutputTokens     int64 `json:"outputTokens"`
							CacheReadTokens  int64 `json:"cacheReadTokens"`
							CacheWriteTokens int64 `json:"cacheWriteTokens"`
							ReasoningTokens  int64 `json:"reasoningTokens"`
						} `json:"usage"`
					} `json:"modelMetrics"`
				} `json:"data"`
			}
			if err := json.Unmarshal(line, &event); err != nil {
				return nil
			}

			ts := time.Now().UTC()
			if firstTime.IsZero() {
				firstTime = ts
			}
			lastTime = ts

			if event.Type == "session.start" && event.Data.Context != nil && event.Data.Context.StartTime != "" {
				if t, err := time.Parse(time.RFC3339, event.Data.Context.StartTime); err == nil {
					firstTime = t
				}
			}

			if event.Type == "assistant.message" {
				turnIndex++
				turnModel := event.Data.Model
				if turnModel == "" {
					turnModel = session.Model
				} else {
					session.Model = turnModel
				}

				session.Turns = append(session.Turns, Turn{
					Index:     turnIndex,
					Timestamp: ts,
					Role:      "assistant",
					Model:     turnModel,
					Usage: TokenUsage{
						OutputTokens: event.Data.OutputTokens,
					},
				})
			}

			if event.Type == "session.shutdown" && len(event.Data.ModelMetrics) > 0 {
				for modelName, metric := range event.Data.ModelMetrics {
					session.Model = modelName
					grossInput := metric.Usage.InputTokens
					cacheRead := metric.Usage.CacheReadTokens
					cacheWrite := metric.Usage.CacheWriteTokens
					netInput := grossInput - cacheRead - cacheWrite
					if netInput < 0 {
						netInput = 0
					}

					session.TotalUsage.InputTokens = netInput
					session.TotalUsage.OutputTokens = metric.Usage.OutputTokens
					session.TotalUsage.CacheReadTokens = cacheRead
					session.TotalUsage.CacheCreationTokens = cacheWrite
				}
			}

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
	session.ID = "copilot:" + session.SessionID

	return session, endOffset, nil
}
