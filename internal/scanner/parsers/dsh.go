package parsers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
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

func parseToolArguments(raw interface{}) (map[string]interface{}, string) {
	if raw == nil {
		return map[string]interface{}{}, "{}"
	}
	switch v := raw.(type) {
	case string:
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(v), &m); err == nil && m != nil {
			return m, v
		}
		return map[string]interface{}{"raw": v}, v
	case map[string]interface{}:
		b, _ := json.Marshal(v)
		return v, string(b)
	default:
		b, _ := json.Marshal(v)
		var m map[string]interface{}
		_ = json.Unmarshal(b, &m)
		if m == nil {
			m = map[string]interface{}{}
		}
		return m, string(b)
	}
}

func (p *DSHParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	session := &ParsedSession{
		AgentName:  "dsh",
		Model:      "zai-glm-4.7",
		Turns:      make([]Turn, 0),
		Status:     "completed",
		TotalUsage: TokenUsage{},
	}

	seenTurnStep := make(map[string]int) // key -> turn index in session.Turns
	usageRecorded := make(map[string]bool)
	stepStart := make(map[string]int64)
	stepFirstChunk := make(map[string]int64)
	stepFinish := make(map[string]int64)
	toolCallAt := make(map[string]int64)
	sandbox := make(map[string]interface{})
	var presetChain []string

	skillsCatalogMap := make(map[string]string)
	toolsAvailableMap := make(map[string]bool)
	var providersUsed []string
	var modelsUsed []string

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
			AgentPreset   string `json:"agentPreset"`
			Data          struct {
				Turn        int         `json:"turn"`
				Step        int         `json:"step"`
				Provider    string      `json:"provider"`
				Model       string      `json:"model"`
				CallID      string      `json:"callId"`
				Name        string      `json:"name"`
				Arguments   interface{} `json:"arguments"`
				Mode        string      `json:"mode"`
				Policy      string      `json:"policy"`
				Preset      string      `json:"preset"`
				AgentPreset string      `json:"agentPreset"`
				Source      interface{} `json:"source"`
				Header *struct {
					Tools []struct {
						Name string `json:"name"`
					} `json:"tools"`
				} `json:"header"`
				Content []struct {
					Type       string      `json:"type"`
					Text       string      `json:"text"`
					Reasoning  string      `json:"reasoning"`
					Thinking   string      `json:"thinking"`
					ToolCallID string      `json:"toolCallId"`
					IsError    bool        `json:"isError"`
					Content    interface{} `json:"content"`
				} `json:"content"`
				Chunk *struct {
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
					Role   string `json:"role"`
					Source *struct {
						Kind     string `json:"kind"`
						Provider string `json:"provider"`
						Model    string `json:"model"`
						CallID   string `json:"callId"`
					} `json:"source"`
					Content []struct {
						Type       string      `json:"type"`
						Text       string      `json:"text"`
						Reasoning  string      `json:"reasoning"`
						Thinking   string      `json:"thinking"`
						ToolCallID string      `json:"toolCallId"`
						IsError    bool        `json:"isError"`
						Content    interface{} `json:"content"`
					} `json:"content"`
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
				session.SubagentType = "dsh-subagent"
			}
			if event.AgentPreset != "" {
				presetChain = append(presetChain, event.AgentPreset)
			}
		}

		if event.Type == "request/context" {
			if event.Data.Model != "" {
				session.Model = event.Data.Model
				hasModel := false
				for _, m := range modelsUsed {
					if m == event.Data.Model {
						hasModel = true
						break
					}
				}
				if !hasModel {
					modelsUsed = append(modelsUsed, event.Data.Model)
				}
			}
			if event.Data.Provider != "" {
				session.Provider = event.Data.Provider
				hasProv := false
				for _, p := range providersUsed {
					if p == event.Data.Provider {
						hasProv = true
						break
					}
				}
				if !hasProv {
					providersUsed = append(providersUsed, event.Data.Provider)
				}
			}
		}

		if event.Type == "request/header" && event.Data.Header != nil {
			for _, tool := range event.Data.Header.Tools {
				if tool.Name != "" {
					toolsAvailableMap[tool.Name] = true
				}
			}
		}

		if event.Type == "agent-preset/selected" {
			preset := event.Data.AgentPreset
			if preset != "" && (len(presetChain) == 0 || presetChain[len(presetChain)-1] != preset) {
				presetChain = append(presetChain, preset)
			}
		}

		if event.Type == "sandbox/mode" {
			if event.Data.Mode != "" {
				sandbox["mode"] = event.Data.Mode
				srcStr := ""
				if s, ok := event.Data.Source.(string); ok {
					srcStr = s
				}
				if srcStr != "" {
					sandbox["mode_source"] = srcStr
				} else {
					sandbox["mode_source"] = "session"
				}
			}
		}

		if event.Type == "approval/policy" {
			if event.Data.Policy != "" {
				sandbox["approval"] = event.Data.Policy
				srcStr := ""
				if s, ok := event.Data.Source.(string); ok {
					srcStr = s
				}
				if srcStr != "" {
					sandbox["approval_source"] = srcStr
				} else {
					sandbox["approval_source"] = "session"
				}
			}
		}

		if event.Type == "permission/preset" {
			if event.Data.Preset != "" {
				sandbox["permission_preset"] = event.Data.Preset
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

		if event.Type == "user/message" {
			sourceKind := ""
			if event.Data.Source != nil {
				if s, ok := event.Data.Source.(string); ok {
					sourceKind = s
				} else if m, ok := event.Data.Source.(map[string]interface{}); ok {
					if k, ok := m["kind"].(string); ok {
						sourceKind = k
					}
					if sourceKind == "skill-catalog" {
						if entries, ok := m["entries"].([]interface{}); ok {
							for _, ent := range entries {
								if em, ok := ent.(map[string]interface{}); ok {
									name, _ := em["name"].(string)
									desc, _ := em["description"].(string)
									if name != "" {
										skillsCatalogMap[name] = desc
									}
								}
							}
						}
					}
				}
			}

			// Only real user turns (source.kind == "user" or empty source) should be preserved in trace
			if sourceKind == "user" || (sourceKind == "" && event.Data.Source == nil) {
				var parts []string
				for _, c := range event.Data.Content {
					if (c.Type == "text" || c.Type == "") && c.Text != "" {
						parts = append(parts, c.Text)
					}
				}
				userText := strings.Join(parts, "\n\n")
				if strings.TrimSpace(userText) != "" && !strings.HasPrefix(strings.TrimSpace(userText), "<system-reminder>") {
					turnIndex++
					turn := Turn{
						Index:       turnIndex,
						Timestamp:   ts,
						Role:        "user",
						Content:     userText,
						Tools:       make([]string, 0),
						ToolCalls:   make([]models.ToolCall, 0),
						ToolResults: make([]models.ToolResult, 0),
					}
					session.Turns = append(session.Turns, turn)
				}
			}
		}

		if event.Type == "tool/call" {
			callID := event.Data.CallID
			toolName := event.Data.Name
			if callID != "" && rawTime > 0 {
				toolCallAt[callID] = rawTime
			}
			argsMap, argsJSON := parseToolArguments(event.Data.Arguments)
			toolCall := models.ToolCall{
				ID:       callID,
				Name:     toolName,
				Args:     argsMap,
				ArgsJSON: argsJSON,
			}

			// Attach to the latest assistant turn if exists, else create an assistant turn
			if len(session.Turns) > 0 && session.Turns[len(session.Turns)-1].Role == "assistant" {
				lastIdx := len(session.Turns) - 1
				if toolName != "" {
					session.Turns[lastIdx].Tools = append(session.Turns[lastIdx].Tools, toolName)
				}
				session.Turns[lastIdx].ToolCalls = append(session.Turns[lastIdx].ToolCalls, toolCall)
			} else {
				turnIndex++
				turn := Turn{
					Index:       turnIndex,
					Timestamp:   ts,
					Role:        "assistant",
					Model:       session.Model,
					Tools:       []string{toolName},
					ToolCalls:   []models.ToolCall{toolCall},
					ToolResults: make([]models.ToolResult, 0),
				}
				key := fmt.Sprintf("%d:%d", event.Data.Turn, event.Data.Step)
				seenTurnStep[key] = len(session.Turns)
				session.Turns = append(session.Turns, turn)
			}
		}

		if event.Type == "tool/result" {
			callID := event.Data.CallID
			if callID == "" && event.Data.Message != nil && event.Data.Message.Source != nil {
				callID = event.Data.Message.Source.CallID
			}

			var textParts []string
			contentList := event.Data.Content
			if event.Data.Message != nil && len(event.Data.Message.Content) > 0 {
				contentList = event.Data.Message.Content
			}
			for _, block := range contentList {
				if callID == "" && block.ToolCallID != "" {
					callID = block.ToolCallID
				}
				if block.Text != "" {
					textParts = append(textParts, block.Text)
				} else if block.Content != nil {
					switch val := block.Content.(type) {
					case string:
						textParts = append(textParts, val)
					case []interface{}:
						for _, item := range val {
							if im, ok := item.(map[string]interface{}); ok {
								if t, ok := im["text"].(string); ok {
									textParts = append(textParts, t)
								}
							}
						}
					}
				}
			}

			if callID != "" {
				if started, ok := toolCallAt[callID]; ok {
					if rawTime >= started {
						toolMsTotal += (rawTime - started)
					}
					delete(toolCallAt, callID)
				}
			}

			toolResult := models.ToolResult{
				ID:      callID,
				Content: strings.Join(textParts, "\n"),
			}

			if len(session.Turns) > 0 {
				lastIdx := len(session.Turns) - 1
				session.Turns[lastIdx].ToolResults = append(session.Turns[lastIdx].ToolResults, toolResult)
			}
		}

		// Handle chunk / message usage and assistant turns
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

		if event.Type == "assistant/message" {
			var textParts []string
			var reasoningParts []string
			contentList := event.Data.Content
			if event.Data.Message != nil && len(event.Data.Message.Content) > 0 {
				contentList = event.Data.Message.Content
			}
			for _, c := range contentList {
				if c.Type == "reasoning" || c.Type == "thinking" {
					th := c.Reasoning
					if th == "" {
						th = c.Thinking
					}
					if th == "" {
						th = c.Text
					}
					if th != "" {
						reasoningParts = append(reasoningParts, th)
					}
				} else if (c.Type == "text" || c.Type == "") && c.Text != "" {
					textParts = append(textParts, c.Text)
				}
			}

			textContent := strings.Join(textParts, "\n\n")
			reasoningContent := strings.Join(reasoningParts, "\n\n")

			key := fmt.Sprintf("%d:%d", event.Data.Turn, event.Data.Step)
			turnIdx, exists := seenTurnStep[key]
			if exists && turnIdx >= 0 && turnIdx < len(session.Turns) && session.Turns[turnIdx].Role == "assistant" {
				if textContent != "" {
					if session.Turns[turnIdx].Content != "" {
						session.Turns[turnIdx].Content += "\n\n" + textContent
					} else {
						session.Turns[turnIdx].Content = textContent
					}
				}
				if reasoningContent != "" {
					session.Turns[turnIdx].Thinking = reasoningContent
					session.Turns[turnIdx].ReasoningEffort = "high"
				}
				if hasUsage {
					session.Turns[turnIdx].Usage.InputTokens = usageIn
					session.Turns[turnIdx].Usage.OutputTokens = usageOut
					session.Turns[turnIdx].Usage.CacheReadTokens = usageCache
				}
			} else {
				turnIndex++
				turn := Turn{
					Index:       turnIndex,
					Timestamp:   ts,
					Role:        "assistant",
					Model:       session.Model,
					Content:     textContent,
					Thinking:    reasoningContent,
					Tools:       make([]string, 0),
					ToolCalls:   make([]models.ToolCall, 0),
					ToolResults: make([]models.ToolResult, 0),
				}
				if reasoningContent != "" {
					turn.ReasoningEffort = "high"
				}
				if hasUsage {
					turn.Usage = TokenUsage{
						InputTokens:     usageIn,
						OutputTokens:    usageOut,
						CacheReadTokens: usageCache,
					}
				}
				seenTurnStep[key] = len(session.Turns)
				session.Turns = append(session.Turns, turn)
			}
		}

		if hasUsage {
			key := fmt.Sprintf("%d:%d", event.Data.Turn, event.Data.Step)
			if !usageRecorded[key] {
				usageRecorded[key] = true
				session.TotalUsage.InputTokens += usageIn
				session.TotalUsage.OutputTokens += usageOut
				session.TotalUsage.CacheReadTokens += usageCache

				// Ensure an assistant turn exists if one hasn't been created yet
				if _, exists := seenTurnStep[key]; !exists {
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
						Tools:       make([]string, 0),
						ToolCalls:   make([]models.ToolCall, 0),
						ToolResults: make([]models.ToolResult, 0),
					}
					seenTurnStep[key] = len(session.Turns)
					session.Turns = append(session.Turns, turn)
				}
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

	var effectivePreset string
	if len(presetChain) > 0 {
		effectivePreset = presetChain[len(presetChain)-1]
	}

	// Prepare skills catalog and tools available
	var skillsCatalog []models.DSHSkillEntry
	var skillNames []string
	for k := range skillsCatalogMap {
		skillNames = append(skillNames, k)
	}
	sort.Strings(skillNames)
	for _, name := range skillNames {
		skillsCatalog = append(skillsCatalog, models.DSHSkillEntry{
			Name:        name,
			Description: skillsCatalogMap[name],
		})
	}
	if skillsCatalog == nil {
		skillsCatalog = []models.DSHSkillEntry{}
	}

	var toolsAvailable []string
	for t := range toolsAvailableMap {
		toolsAvailable = append(toolsAvailable, t)
	}
	sort.Strings(toolsAvailable)
	if toolsAvailable == nil {
		toolsAvailable = []string{}
	}

	if modelsUsed == nil {
		modelsUsed = []string{}
	}
	if providersUsed == nil {
		providersUsed = []string{}
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
		AgentPreset:    effectivePreset,
		PresetChain:    presetChain,
		Sandbox:        sandbox,
		ModelsUsed:     modelsUsed,
		ProvidersUsed:  providersUsed,
		SkillsCatalog:  skillsCatalog,
		ToolsAvailable: toolsAvailable,
	}

	// Correlate plugin lifecycle sidecar events if present
	lifecyclePath := DefaultDSHLifecycleFilePath()
	if lifecyclePath != "" {
		if _, statErr := os.Stat(lifecyclePath); statErr == nil {
			var since, until *int64
			if !firstTime.IsZero() {
				s := firstTime.UnixMilli()
				since = &s
			}
			if !lastTime.IsZero() {
				u := lastTime.UnixMilli()
				until = &u
			}
			ev := ReadDSHLifecycleEvents(lifecyclePath, since, until, 500)
			summary := SummarizeDSHLifecycleEvents(ev)
			summary.Installed = true
			if since != nil || until != nil {
				summary.Correlation = "time-window"
			}
			session.DSH.Lifecycle = &summary
		}
	}

	return session, endOffset, nil
}

// DSHFiberStates maps Cordis FiberState const enum ordinals to readable state names.
var DSHFiberStates = map[int]string{
	0: "pending",
	1: "loading",
	2: "active",
	3: "failed",
	4: "disposed",
	5: "unloading",
}

// NormalizeDSHFiberState maps either integer ordinals (0-5) or state name strings to lowercase canonical state names.
func NormalizeDSHFiberState(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return strings.ToLower(strings.TrimSpace(val))
	case float64:
		return DSHFiberStates[int(val)]
	case int:
		return DSHFiberStates[val]
	case int64:
		return DSHFiberStates[int(val)]
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return DSHFiberStates[int(i)]
		}
	}
	return ""
}

// ResolveDSHLifecycleFilePath finds dsh_lifecycle.jsonl by walking up from the session file directory or checking env/home dirs.
func ResolveDSHLifecycleFilePath(sessionPath string) string {
	if sessionPath != "" {
		dir := filepath.Dir(sessionPath)
		for i := 0; i < 6; i++ {
			p := filepath.Join(dir, ".tokentelemetry", "dsh_lifecycle.jsonl")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			p2 := filepath.Join(dir, "dsh_lifecycle.jsonl")
			if _, err := os.Stat(p2); err == nil {
				return p2
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return DefaultDSHLifecycleFilePath()
}

// DefaultDSHLifecycleFilePath resolves the path to dsh_lifecycle.jsonl respecting environment variables and user home.
func DefaultDSHLifecycleFilePath() string {
	if dir := os.Getenv("TOKENTELEMETRY_DATA_DIR"); dir != "" {
		p := filepath.Join(dir, "dsh_lifecycle.jsonl")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		p2 := filepath.Join(dir, ".tokentelemetry", "dsh_lifecycle.jsonl")
		if _, err := os.Stat(p2); err == nil {
			return p2
		}
		return p
	}
	if home := os.Getenv("TOKENTELEMETRY_HOME"); home != "" {
		p := filepath.Join(home, ".tokentelemetry", "dsh_lifecycle.jsonl")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		p2 := filepath.Join(home, "dsh_lifecycle.jsonl")
		if _, err := os.Stat(p2); err == nil {
			return p2
		}
		return p
	}
	if scanDir := os.Getenv("E2E_SCAN_DIR"); scanDir != "" {
		p := filepath.Join(scanDir, ".tokentelemetry", "dsh_lifecycle.jsonl")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		p2 := filepath.Join(scanDir, "dsh_lifecycle.jsonl")
		if _, err := os.Stat(p2); err == nil {
			return p2
		}
		return p
	}
	if tmpDir := os.Getenv("E2E_TEMP_DIR"); tmpDir != "" {
		p := filepath.Join(tmpDir, "logs", ".tokentelemetry", "dsh_lifecycle.jsonl")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".tokentelemetry", "dsh_lifecycle.jsonl")
	}
	return ""
}

// ReadDSHLifecycleEvents parses plugin lifecycle transitions from the sidecar log file.
// If filePath is empty, the default path is used. Missing files return an empty list without error.
// Torn/partial lines are ignored. Results are sorted ascending by timestamp and capped at limit (most recent).
func ReadDSHLifecycleEvents(filePath string, sinceMs, untilMs *int64, limit int) []models.DSHLifecycleEvent {
	if filePath == "" {
		filePath = DefaultDSHLifecycleFilePath()
	}
	if filePath == "" {
		return []models.DSHLifecycleEvent{}
	}

	f, err := os.Open(filePath)
	if err != nil {
		return []models.DSHLifecycleEvent{}
	}
	defer f.Close()

	var events []models.DSHLifecycleEvent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var row map[string]interface{}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			// Torn / partial line
			continue
		}

		rawTS, ok := row["ts"]
		if !ok || rawTS == nil {
			continue
		}

		var ts int64
		switch t := rawTS.(type) {
		case float64:
			ts = int64(t)
		case int64:
			ts = t
		case int:
			ts = int64(t)
		case json.Number:
			if parsed, err := t.Int64(); err == nil {
				ts = parsed
			} else {
				continue
			}
		default:
			continue
		}

		if sinceMs != nil && ts < *sinceMs {
			continue
		}
		if untilMs != nil && ts > *untilMs {
			continue
		}

		plugin := "unknown"
		if p, ok := row["plugin"].(string); ok && p != "" {
			plugin = p
		} else if n, ok := row["name"].(string); ok && n != "" {
			plugin = n
		}

		var entryID *string
		if eid, ok := row["entry_id"].(string); ok && eid != "" {
			entryID = &eid
		}

		var uid *int64
		if u, ok := row["uid"].(float64); ok {
			i := int64(u)
			uid = &i
		} else if u, ok := row["uid"].(int64); ok {
			uid = &u
		}

		fromState := NormalizeDSHFiberState(row["from"])
		toState := NormalizeDSHFiberState(row["to"])

		var errStr string
		if e, ok := row["error"].(string); ok {
			errStr = e
		}

		events = append(events, models.DSHLifecycleEvent{
			TS:      ts,
			Plugin:  plugin,
			EntryID: entryID,
			UID:     uid,
			From:    fromState,
			To:      toState,
			Error:   errStr,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].TS < events[j].TS
	})

	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}

	if events == nil {
		events = []models.DSHLifecycleEvent{}
	}

	return events
}

// SummarizeDSHLifecycleEvents rolls transitions up into summary metrics.
func SummarizeDSHLifecycleEvents(events []models.DSHLifecycleEvent) models.DSHLifecycleSummary {
	pluginsMap := make(map[string]*models.DSHPluginSummary)
	failed := 0
	reloads := 0
	unloads := 0

	for _, e := range events {
		p, exists := pluginsMap[e.Plugin]
		if !exists {
			p = &models.DSHPluginSummary{
				Plugin: e.Plugin,
			}
			pluginsMap[e.Plugin] = p
		}
		p.Transitions++
		p.FinalState = e.To

		if e.To == "failed" {
			p.Failed++
			failed++
		} else if e.To == "loading" && (e.From == "active" || e.From == "failed" || e.From == "disposed") {
			reloads++
		} else if e.To == "unloading" {
			unloads++
		}
	}

	pluginsList := make([]models.DSHPluginSummary, 0, len(pluginsMap))
	for _, p := range pluginsMap {
		pluginsList = append(pluginsList, *p)
	}

	// Sort plugins: failed count descending, then plugin name ascending
	sort.Slice(pluginsList, func(i, j int) bool {
		if pluginsList[i].Failed != pluginsList[j].Failed {
			return pluginsList[i].Failed > pluginsList[j].Failed
		}
		return pluginsList[i].Plugin < pluginsList[j].Plugin
	})

	var firstTS, lastTS *int64
	if len(events) > 0 {
		f := events[0].TS
		l := events[len(events)-1].TS
		firstTS = &f
		lastTS = &l
	}

	return models.DSHLifecycleSummary{
		Installed:   true,
		Correlation: "none",
		Transitions: len(events),
		Failed:      failed,
		Reloads:     reloads,
		Unloads:     unloads,
		FirstTS:     firstTS,
		LastTS:      lastTS,
		Plugins:     pluginsList,
		Events:      events,
	}
}
