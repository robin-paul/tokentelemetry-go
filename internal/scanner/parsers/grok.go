package parsers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// GrokTurn captures per-turn prompt, cache, completion, and reasoning token counts.
type GrokTurn struct {
	PromptTokens     int64
	CachedTokens     int64
	CompletionTokens int64
	ReasoningTokens  int64
}

// GrokBilledUsage holds aggregated usage and turn records parsed from unified.jsonl.
type GrokBilledUsage struct {
	Input     int64
	Output    int64
	Cached    int64
	Reasoning int64
	Turns     []GrokTurn
}

type grokCacheKey struct {
	mtimeNano int64
	size      int64
	path      string
}

var (
	grokCacheMu sync.RWMutex
	grokCache   struct {
		key  grokCacheKey
		data map[string]GrokBilledUsage
	}
)

// ResetGrokLogCache clears the in-memory stat cache (primarily for unit tests).
func ResetGrokLogCache() {
	grokCacheMu.Lock()
	defer grokCacheMu.Unlock()
	grokCache.key = grokCacheKey{}
	grokCache.data = nil
}

// LoadGrokUnifiedLog parses ~/.grok/logs/unified.jsonl with mtime/size stat caching.
func LoadGrokUnifiedLog(logPath string) (map[string]GrokBilledUsage, error) {
	st, err := os.Stat(logPath)
	if err != nil {
		return nil, err
	}

	key := grokCacheKey{
		mtimeNano: st.ModTime().UnixNano(),
		size:      st.Size(),
		path:      logPath,
	}

	grokCacheMu.RLock()
	if grokCache.key == key && grokCache.data != nil {
		res := grokCache.data
		grokCacheMu.RUnlock()
		return res, nil
	}
	grokCacheMu.RUnlock()

	f, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bySID := make(map[string]GrokBilledUsage)
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if !bytes.Contains(line, []byte("shell.turn.inference_done")) {
			continue
		}

		var rec struct {
			Msg string `json:"msg"`
			SID string `json:"sid"`
			Ctx struct {
				PromptTokens       int64 `json:"prompt_tokens"`
				CachedPromptTokens int64 `json:"cached_prompt_tokens"`
				CompletionTokens   int64 `json:"completion_tokens"`
				ReasoningTokens    int64 `json:"reasoning_tokens"`
			} `json:"ctx"`
		}

		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}
		if rec.Msg != "shell.turn.inference_done" || rec.SID == "" {
			continue
		}

		prompt := rec.Ctx.PromptTokens
		cached := rec.Ctx.CachedPromptTokens
		completion := rec.Ctx.CompletionTokens
		reasoning := rec.Ctx.ReasoningTokens

		if prompt < 0 || cached < 0 || completion < 0 {
			continue
		}
		if cached > prompt {
			cached = prompt
		}
		uncached := prompt - cached
		if reasoning < 0 {
			reasoning = 0
		}

		row := bySID[rec.SID]
		row.Input += uncached
		row.Output += completion
		row.Cached += cached
		row.Reasoning += reasoning
		row.Turns = append(row.Turns, GrokTurn{
			PromptTokens:     prompt,
			CachedTokens:     cached,
			CompletionTokens: completion,
			ReasoningTokens:  reasoning,
		})
		bySID[rec.SID] = row
	}

	if err := scanner.Err(); err != nil && len(bySID) == 0 {
		return nil, err
	}

	grokCacheMu.Lock()
	grokCache.key = key
	grokCache.data = bySID
	grokCacheMu.Unlock()

	return bySID, nil
}

type GrokParser struct {
	UnifiedLogPath string
}

func NewGrokParser() *GrokParser {
	return &GrokParser{}
}

func NewGrokParserWithLog(logPath string) *GrokParser {
	return &GrokParser{UnifiedLogPath: logPath}
}

func (p *GrokParser) AgentName() string {
	return "grok"
}

func (p *GrokParser) Detect(filePath string) bool {
	lower := strings.ToLower(filePath)
	return strings.Contains(lower, ".grok/sessions") && (strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".jsonl"))
}

func (p *GrokParser) ResolveUnifiedLogPath(filePath string) string {
	if p.UnifiedLogPath != "" {
		return p.UnifiedLogPath
	}
	if env := os.Getenv("GROK_UNIFIED_LOG"); env != "" {
		return env
	}
	if filePath != "" {
		dir := filePath
		for {
			parent := filepath.Dir(dir)
			if parent == dir || parent == "." || parent == "/" {
				break
			}
			if filepath.Base(parent) == ".grok" {
				candidate := filepath.Join(parent, "logs", "unified.jsonl")
				if _, err := os.Stat(candidate); err == nil {
					return candidate
				}
				break
			}
			dir = parent
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		candidate := filepath.Join(home, ".grok", "logs", "unified.jsonl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func (p *GrokParser) Parse(r io.Reader, startOffset int64) (*ParsedSession, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, startOffset, err
	}
	endOffset := startOffset + int64(len(data))

	var filePath string
	if f, ok := r.(*os.File); ok {
		filePath = f.Name()
	}

	session := &ParsedSession{
		AgentName:  "grok",
		Model:      "grok-build",
		Turns:      make([]Turn, 0),
		Status:     "completed",
		TotalUsage: TokenUsage{},
		FilePath:   filePath,
	}

	var sid string
	if filePath != "" {
		parentDir := filepath.Dir(filePath)
		base := filepath.Base(parentDir)
		if base != "." && base != "/" && base != "sessions" && !strings.HasPrefix(base, "%") {
			sid = base
		}
	}

	// 1. Parse JSON document if structured summary.json or signals.json
	var summaryDoc struct {
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
		GeneratedTitle string `json:"generated_title"`
		CurrentModelID string `json:"current_model_id"`
		NumMessages    int    `json:"num_messages"`
		AgentName      string `json:"agent_name"`
		Info           *struct {
			Cwd string `json:"cwd"`
		} `json:"info"`
	}

	var signalsDoc struct {
		ContextTokensUsed int64    `json:"contextTokensUsed"`
		ToolsUsed         []string `json:"toolsUsed"`
		ModelsUsed        []string `json:"modelsUsed"`
	}

	isSummary := false
	isSignals := false

	if err := json.Unmarshal(data, &summaryDoc); err == nil && summaryDoc.CreatedAt != "" {
		isSummary = true
		if summaryDoc.CurrentModelID != "" {
			session.Model = summaryDoc.CurrentModelID
		}
		if summaryDoc.Info != nil && summaryDoc.Info.Cwd != "" {
			session.ProjectName = ExtractProjectName(summaryDoc.Info.Cwd)
		}
		if t, err := time.Parse(time.RFC3339, summaryDoc.CreatedAt); err == nil {
			session.StartTime = t
		}
		if t, err := time.Parse(time.RFC3339, summaryDoc.UpdatedAt); err == nil {
			session.EndTime = t
		}
		if !session.StartTime.IsZero() && !session.EndTime.IsZero() && session.EndTime.After(session.StartTime) {
			session.DurationSeconds = session.EndTime.Sub(session.StartTime).Seconds()
		}
	} else if err := json.Unmarshal(data, &signalsDoc); err == nil && (signalsDoc.ContextTokensUsed > 0 || len(signalsDoc.ToolsUsed) > 0 || len(signalsDoc.ModelsUsed) > 0) {
		isSignals = true
		if len(signalsDoc.ModelsUsed) > 0 {
			session.Model = signalsDoc.ModelsUsed[0]
		}
	}

	// If reading summary.json, inspect sibling signals.json if available
	if isSummary && filePath != "" {
		signalsPath := filepath.Join(filepath.Dir(filePath), "signals.json")
		if sigBytes, err := os.ReadFile(signalsPath); err == nil {
			var sigDoc struct {
				ContextTokensUsed int64    `json:"contextTokensUsed"`
				ToolsUsed         []string `json:"toolsUsed"`
				ModelsUsed        []string `json:"modelsUsed"`
			}
			if err := json.Unmarshal(sigBytes, &sigDoc); err == nil {
				if len(sigDoc.ModelsUsed) > 0 && session.Model == "grok-build" {
					session.Model = sigDoc.ModelsUsed[0]
				}
				if signalsDoc.ContextTokensUsed == 0 {
					signalsDoc.ContextTokensUsed = sigDoc.ContextTokensUsed
				}
				if len(signalsDoc.ToolsUsed) == 0 {
					signalsDoc.ToolsUsed = sigDoc.ToolsUsed
				}
			}
		}
	}

	// If reading signals.json, inspect sibling summary.json if available
	if isSignals && filePath != "" {
		summaryPath := filepath.Join(filepath.Dir(filePath), "summary.json")
		if sumBytes, err := os.ReadFile(summaryPath); err == nil {
			var sumDoc struct {
				CreatedAt      string `json:"created_at"`
				UpdatedAt      string `json:"updated_at"`
				GeneratedTitle string `json:"generated_title"`
				CurrentModelID string `json:"current_model_id"`
				Info           *struct {
					Cwd string `json:"cwd"`
				} `json:"info"`
			}
			if err := json.Unmarshal(sumBytes, &sumDoc); err == nil {
				if sumDoc.CurrentModelID != "" {
					session.Model = sumDoc.CurrentModelID
				}
				if sumDoc.Info != nil && sumDoc.Info.Cwd != "" {
					session.ProjectName = ExtractProjectName(sumDoc.Info.Cwd)
				}
				if t, err := time.Parse(time.RFC3339, sumDoc.CreatedAt); err == nil {
					session.StartTime = t
				}
				if t, err := time.Parse(time.RFC3339, sumDoc.UpdatedAt); err == nil {
					session.EndTime = t
				}
				if !session.StartTime.IsZero() && !session.EndTime.IsZero() && session.EndTime.After(session.StartTime) {
					session.DurationSeconds = session.EndTime.Sub(session.StartTime).Seconds()
				}
			}
		}
	}

	// 2. Check unified.jsonl for billed per-turn usage
	logPath := p.ResolveUnifiedLogPath(filePath)
	var billed *GrokBilledUsage

	if logPath != "" && sid != "" {
		if usageMap, err := LoadGrokUnifiedLog(logPath); err == nil {
			if u, found := usageMap[sid]; found && len(u.Turns) > 0 {
				billed = &u
			}
		}
	}

	if billed != nil {
		session.TotalUsage.InputTokens = billed.Input
		session.TotalUsage.OutputTokens = billed.Output
		session.TotalUsage.CacheReadTokens = billed.Cached
		session.TotalUsage.CacheCreationTokens = 0

		session.Turns = make([]Turn, len(billed.Turns))
		for i, t := range billed.Turns {
			turnTime := session.StartTime
			if turnTime.IsZero() {
				turnTime = time.Now().UTC()
			}
			session.Turns[i] = Turn{
				Index:     i + 1,
				Timestamp: turnTime,
				Role:      "assistant",
				Model:     session.Model,
				Usage: TokenUsage{
					InputTokens:     t.PromptTokens - t.CachedTokens,
					OutputTokens:    t.CompletionTokens,
					CacheReadTokens: t.CachedTokens,
				},
				Tools: signalsDoc.ToolsUsed,
			}
		}
	} else if isSignals && signalsDoc.ContextTokensUsed > 0 {
		// Fallback to measured context footprint
		session.TotalUsage.InputTokens = signalsDoc.ContextTokensUsed
		session.Turns = append(session.Turns, Turn{
			Index:     1,
			Timestamp: time.Now().UTC(),
			Role:      "assistant",
			Model:     session.Model,
			Usage: TokenUsage{
				InputTokens: signalsDoc.ContextTokensUsed,
			},
			Tools: signalsDoc.ToolsUsed,
		})
	} else if !isSummary && !isSignals {
		// Line-by-line updates.jsonl fallback
		_, _ = ReadLines(strings.NewReader(string(data)), 0, func(line []byte, offset int64) error {
			var update struct {
				Params struct {
					Meta struct {
						TotalTokens int64 `json:"totalTokens"`
					} `json:"_meta"`
				} `json:"params"`
			}
			if err := json.Unmarshal(line, &update); err == nil && update.Params.Meta.TotalTokens > session.TotalUsage.InputTokens {
				session.TotalUsage.InputTokens = update.Params.Meta.TotalTokens
			}
			return nil
		})
	}

	if sid != "" {
		session.SessionID = sid
	}
	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	session.ID = "grok:" + session.SessionID

	return session, endOffset, nil
}
