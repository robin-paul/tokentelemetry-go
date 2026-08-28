package parsers

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
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
		AgentName:  "dsh",
		Model:      "zai-glm-4.7",
		Turns:      make([]Turn, 0),
		Status:     "completed",
		TotalUsage: TokenUsage{},
	}

	seenTurnStep := make(map[string]bool)
	stepStart := make(map[string]int64)
	stepFirstChunk := make(map[string]int64)
	stepFinish := make(map[string]int64)
	toolCallAt := make(map[string]int64)

	var toolMsTotal int64
	var turnsCount, stepsCount int
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
				Turn      int    `json:"turn"`
				Step      int    `json:"step"`
				Provider  string `json:"provider"`
				Model     string `json:"model"`
				CallID    string `json:"callId"`
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
				Chunk     *struct {
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
				Message *struct {
					Source *struct {
						Kind   string `json:"kind"`
						CallID string `json:"callId"`
					} `json:"source"`
				} `json:"message"`
			} `json:"data"`
		}

		if err := json.Unmarshal(line, &event); err != nil {
			return nil
		}

		rawTime := event.Time
		if rawTime <= 0 {
			rawTime = event.CreatedAt
		}

		var ts time.Time
		if rawTime > 0 {
			ts = time.UnixMilli(rawTime).UTC()
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

		if event.Type == "turn/start" {
			turnsCount++
		}

		if event.Type == "step/start" {
			stepsCount++
			if rawTime > 0 {
				key := fmt.Sprintf("%d:%d", event.Data.Turn, event.Data.Step)
				stepStart[key] = rawTime
			}
		}

		if event.Type == "tool/call" {
			if event.Data.CallID != "" && rawTime > 0 {
				toolCallAt[event.Data.CallID] = rawTime
			}
		}

		if event.Type == "tool/result" {
			callID := ""
			if event.Data.Message != nil && event.Data.Message.Source != nil && event.Data.Message.Source.CallID != "" {
				callID = event.Data.Message.Source.CallID
			} else if event.Data.CallID != "" {
				callID = event.Data.CallID
			}
			if callID != "" {
				if started, ok := toolCallAt[callID]; ok {
					if rawTime >= started {
						toolMsTotal += (rawTime - started)
					}
					delete(toolCallAt, callID)
				}
			}
		}

		// Handle chunk / message usage with deduplication by (turn, step)
		var usageIn, usageOut, usageCache int64
		hasUsage := false

		if event.Type == "assistant/chunk" {
			key := fmt.Sprintf("%d:%d", event.Data.Turn, event.Data.Step)
			if rawTime > 0 {
				if _, ok := stepFirstChunk[key]; !ok {
					stepFirstChunk[key] = rawTime
				}
				if event.Data.Chunk != nil && (event.Data.Chunk.Type == "finish" || event.Data.Chunk.Type == "usage") {
					if rawTime > stepFinish[key] {
						stepFinish[key] = rawTime
					}
				}
			}

			if event.Data.Chunk != nil && event.Data.Chunk.Usage != nil {
				usageIn = event.Data.Chunk.Usage.InputTokens
				usageOut = event.Data.Chunk.Usage.OutputTokens
				usageCache = event.Data.Chunk.Usage.CacheReadTokens
				hasUsage = true
			}
		} else if event.Type == "assistant/message" {
			key := fmt.Sprintf("%d:%d", event.Data.Turn, event.Data.Step)
			if rawTime > 0 && rawTime > stepFinish[key] {
				stepFinish[key] = rawTime
			}
			if event.Data.Usage != nil {
				usageIn = event.Data.Usage.InputTokens
				usageOut = event.Data.Usage.OutputTokens
				usageCache = event.Data.Usage.CacheReadTokens
				hasUsage = true
			}
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

	// --- Latency and throughput breakdown derivation
	var ttfts []float64
	for k, firstT := range stepFirstChunk {
		if startT, ok := stepStart[k]; ok && firstT >= startT {
			ttfts = append(ttfts, float64(firstT-startT))
		}
	}

	var llmMsSum int64
	for k, finishT := range stepFinish {
		if startT, ok := stepStart[k]; ok && finishT >= startT {
			llmMsSum += (finishT - startT)
		}
	}

	var genMsSum int64
	for k, finishT := range stepFinish {
		if firstT, ok := stepFirstChunk[k]; ok && finishT >= firstT {
			genMsSum += (finishT - firstT)
		}
	}

	var llmMs *float64
	if llmMsSum > 0 {
		v := float64(llmMsSum)
		llmMs = &v
	}

	var toolMs *float64
	if toolMsTotal > 0 {
		v := float64(toolMsTotal)
		toolMs = &v
	}

	var ttftAvg *float64
	if len(ttfts) > 0 {
		var sum float64
		for _, v := range ttfts {
			sum += v
		}
		avg := math.Round(sum / float64(len(ttfts)))
		ttftAvg = &avg
	}

	var outputTokPerSec *float64
	if genMsSum > 0 && session.TotalUsage.OutputTokens > 0 {
		tps := math.Round((float64(session.TotalUsage.OutputTokens)/(float64(genMsSum)/1000.0))*10) / 10
		outputTokPerSec = &tps
	}

	var cacheHitPct *float64
	billedInput := session.TotalUsage.InputTokens + session.TotalUsage.CacheReadTokens
	if billedInput > 0 {
		pct := math.Round((float64(session.TotalUsage.CacheReadTokens)/float64(billedInput)*100.0)*10) / 10
		cacheHitPct = &pct
	}

	session.DSH = &models.DSHContext{
		Metrics: &models.DSHMetrics{
			Turns:           turnsCount,
			Steps:           stepsCount,
			LLMMs:           llmMs,
			ToolMs:          toolMs,
			TTFTMsAvg:       ttftAvg,
			OutputTokPerSec: outputTokPerSec,
			CacheHitPct:     cacheHitPct,
		},
	}

	return session, endOffset, nil
}
