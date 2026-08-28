package parsers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultRegistry(t *testing.T) {
	reg := NewDefaultRegistry()
	if len(reg.All()) < 18 {
		t.Fatalf("expected at least 18 parsers, got %d", len(reg.All()))
	}

	expectedAgents := []string{
		"claude_code", "antigravity", "gemini_cli", "codex", "cursor",
		"copilot", "opencode", "grok", "pi", "dsh", "muse",
		"prime", "qwen", "cline", "smallcode", "vibe", "windsurf", "ollama",
	}

	for _, agent := range expectedAgents {
		if reg.Get(agent) == nil {
			t.Errorf("expected parser for agent %q to be registered", agent)
		}
	}
}

func TestClaudeParser(t *testing.T) {
	p := NewClaudeParser()
	if !p.Detect("/Users/dev/.claude/projects/proj1/session123.jsonl") {
		t.Errorf("detection failed for claude path")
	}

	jsonl := `{"type":"user","message":{"content":[{"type":"text","text":"Please analyze this repository."}]}}
{"type":"assistant","sessionId":"sess-1","timestamp":"2026-06-10T12:00:00Z","message":{"model":"claude-3-7-sonnet","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":1000,"cache_creation_input_tokens":200},"content":[{"type":"thinking","thinking":"I should explore the codebase."},{"type":"text","text":"Here is the architecture overview."},{"type":"tool_use","id":"tool_1","name":"Skill","input":{"skill":"codebase-design"}}]}}`
	sess, offset, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if offset <= 0 {
		t.Errorf("expected positive offset, got %d", offset)
	}
	if sess.AgentName != "claude_code" {
		t.Errorf("expected agent claude_code, got %s", sess.AgentName)
	}
	if sess.TotalUsage.InputTokens != 100 || sess.TotalUsage.OutputTokens != 50 || sess.TotalUsage.CacheReadTokens != 1000 {
		t.Errorf("unexpected usage: %+v", sess.TotalUsage)
	}
	if len(sess.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(sess.Turns))
	}
	if sess.Turns[0].Role != "user" || sess.Turns[0].Content != "Please analyze this repository." {
		t.Errorf("unexpected user turn: %+v", sess.Turns[0])
	}
	if sess.Turns[1].Role != "assistant" || sess.Turns[1].Content != "Here is the architecture overview." {
		t.Errorf("unexpected assistant content: %q", sess.Turns[1].Content)
	}
	if sess.Turns[1].Thinking != "I should explore the codebase." || sess.Turns[1].ReasoningEffort != "high" {
		t.Errorf("unexpected thinking or effort: %q / %q", sess.Turns[1].Thinking, sess.Turns[1].ReasoningEffort)
	}
	if len(sess.Turns[1].ToolCalls) != 1 || sess.Turns[1].ToolCalls[0].Name != "Skill" {
		t.Errorf("unexpected tool calls: %+v", sess.Turns[1].ToolCalls)
	}
}

func TestAntigravityParser(t *testing.T) {
	p := NewAntigravityParser()
	if !p.Detect("/Users/dev/.gemini/antigravity-cli/brain/session123/transcript.jsonl") {
		t.Errorf("detection failed for antigravity path")
	}

	// 1. Default fallback to gemini-3.7-flash
	jsonl := `{"step_index":1,"source":"USER_EXPLICIT","type":"USER_INPUT","content":"Fix the bug","created_at":"2026-06-10T12:00:00Z"}
{"step_index":2,"source":"MODEL","type":"MODEL_RESPONSE","content":"Running fix","created_at":"2026-06-10T12:00:05Z","tool_calls":[{"name":"exec_command"}],"metrics":{"input_tokens":50,"output_tokens":25,"cache_read_tokens":0,"cache_creation_tokens":0}}`
	sess, _, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "antigravity" {
		t.Errorf("expected agent antigravity, got %s", sess.AgentName)
	}
	if sess.Model != "gemini-3.7-flash" {
		t.Errorf("expected default model gemini-3.7-flash, got %s", sess.Model)
	}
	if sess.TotalUsage.InputTokens != 52 || sess.TotalUsage.OutputTokens != 25 {
		t.Errorf("unexpected usage: %+v", sess.TotalUsage)
	}
	if len(sess.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(sess.Turns))
	}
	if sess.Turns[0].Content != "Fix the bug" || sess.Turns[1].Content != "Running fix" {
		t.Errorf("unexpected antigravity turn content: %q / %q", sess.Turns[0].Content, sess.Turns[1].Content)
	}
	if len(sess.Turns[1].ToolCalls) != 1 || sess.Turns[1].ToolCalls[0].Name != "exec_command" {
		t.Errorf("unexpected antigravity tool calls: %+v", sess.Turns[1].ToolCalls)
	}

	// 2. Dynamic model detection from USER_SETTINGS_CHANGE
	jsonlWithSettings := `{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","created_at":"2026-08-23T21:06:58Z","content":"<USER_REQUEST>\nFix the bug\n</USER_REQUEST>\n<USER_SETTINGS_CHANGE>\nThe user changed setting ` + "`Model Selection`" + ` from None to Gemini 3.7 Flash (High). No need to comment on this change if the user doesn't ask about it.\n</USER_SETTINGS_CHANGE>"}
{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","created_at":"2026-08-23T21:07:00Z","tool_calls":[{"name":"view_file","args":{}}],"metrics":{"input_tokens":100,"output_tokens":40}}`
	sess2, _, err := p.Parse(strings.NewReader(jsonlWithSettings), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess2.Model != "gemini-3.7-flash" {
		t.Errorf("expected model gemini-3.7-flash, got %s", sess2.Model)
	}
	if sess2.Turns[0].Model != "gemini-3.7-flash" || sess2.Turns[1].Model != "gemini-3.7-flash" {
		t.Errorf("expected turn models to be gemini-3.7-flash, got turn0=%s turn1=%s", sess2.Turns[0].Model, sess2.Turns[1].Model)
	}

	// 3. Dynamic model detection for Gemini 2.5 Pro
	jsonlPro := `{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","created_at":"2026-08-23T21:06:58Z","content":"<USER_SETTINGS_CHANGE>\nThe user changed setting ` + "`Model Selection`" + ` from None to Gemini 2.5 Pro (High).\n</USER_SETTINGS_CHANGE>"}`
	sess3, _, err := p.Parse(strings.NewReader(jsonlPro), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess3.Model != "gemini-2.5-pro" {
		t.Errorf("expected model gemini-2.5-pro, got %s", sess3.Model)
	}
}

func TestCodexParser(t *testing.T) {
	p := NewCodexParser()
	if !p.Detect("/Users/dev/.codex/sessions/2026/06/10/rollout-12345.jsonl") {
		t.Errorf("detection failed for codex path")
	}

	jsonl := `{"type":"session_meta","payload":{"id":"sess-codex-1","model_provider":"openai"}}
{"type":"event_msg","timestamp":"2026-06-10T12:00:00Z","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1500,"cached_input_tokens":1000,"output_tokens":200,"total_tokens":1700}}}}`
	sess, _, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "codex" {
		t.Errorf("expected agent codex, got %s", sess.AgentName)
	}
	// Net prompt tokens: 1500 - 1000 = 500
	if sess.TotalUsage.InputTokens != 500 || sess.TotalUsage.CacheReadTokens != 1000 || sess.TotalUsage.OutputTokens != 200 {
		t.Errorf("unexpected usage: %+v", sess.TotalUsage)
	}
}

func TestDSHParser(t *testing.T) {
	p := NewDSHParser()
	if !p.Detect("/Users/dev/.dsh/sessions/proj/sess1/session.jsonl") {
		t.Errorf("detection failed for dsh path")
	}

	jsonl := `{"type":"session","id":"dsh-sess-1","createdAt":1786806413737,"origin":"subagent","parentSession":"parent-1"}
{"type":"assistant/chunk","data":{"turn":1,"step":1,"chunk":{"type":"usage","usage":{"inputTokens":1000,"outputTokens":200}}}}
{"type":"assistant/message","data":{"turn":1,"step":1,"usage":{"inputTokens":1000,"outputTokens":200}}}`
	sess, _, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "dsh" {
		t.Errorf("expected agent dsh, got %s", sess.AgentName)
	}
	if !sess.IsSubagent || sess.ParentSessionID != "parent-1" {
		t.Errorf("expected subagent with parent-1, got %v / %s", sess.IsSubagent, sess.ParentSessionID)
	}
	// Deduplication: should only be counted once (1000 in, 200 out)
	if sess.TotalUsage.InputTokens != 1000 || sess.TotalUsage.OutputTokens != 200 {
		t.Errorf("deduplication failed, usage: %+v", sess.TotalUsage)
	}
}

func TestCopilotParser(t *testing.T) {
	p := NewCopilotParser()
	if !p.Detect("/Users/dev/.copilot/session-state/sess1/events.jsonl") {
		t.Errorf("detection failed for copilot path")
	}

	jsonl := `{"type":"session.start","data":{"context":{"cwd":"/Users/dev/proj","startTime":"2026-06-10T12:00:00Z"}}}
{"type":"session.shutdown","data":{"modelMetrics":{"claude-haiku-4.5":{"usage":{"inputTokens":63219,"outputTokens":1365,"cacheReadTokens":34516,"cacheWriteTokens":28676}}}}}`
	sess, _, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "copilot" {
		t.Errorf("expected agent copilot, got %s", sess.AgentName)
	}
	// Net input: 63219 - 34516 - 28676 = 27
	if sess.TotalUsage.InputTokens != 27 || sess.TotalUsage.OutputTokens != 1365 || sess.TotalUsage.CacheReadTokens != 34516 || sess.TotalUsage.CacheCreationTokens != 28676 {
		t.Errorf("unexpected net copilot usage: %+v", sess.TotalUsage)
	}
}

func TestGeminiParser(t *testing.T) {
	p := NewGeminiParser()
	if !p.Detect("/Users/dev/.gemini/transcripts/sess1.json") {
		t.Errorf("detection failed for gemini path")
	}

	jsonDoc := `{"sessionId":"gem-1","model":"gemini-2.0-flash","turns":[{"role":"user","parts":[{"text":"hello"}]},{"role":"model","model":"gemini-2.0-flash","usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":34,"cachedContentTokenCount":5},"parts":[{"text":"world"}]}]}`
	sess, _, err := p.Parse(strings.NewReader(jsonDoc), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "gemini_cli" || sess.TotalUsage.InputTokens != 12 || sess.TotalUsage.OutputTokens != 34 || sess.TotalUsage.CacheReadTokens != 5 {
		t.Errorf("unexpected gemini session: %+v", sess)
	}
}

func TestCursorParser(t *testing.T) {
	p := NewCursorParser()
	if !p.Detect("/Users/dev/.cursor/projects/p1/agent-transcripts/s1/s1.jsonl") {
		t.Errorf("detection failed for cursor path")
	}

	jsonl := `{"role":"assistant","message":{"model":"claude-3-5-sonnet","usage":{"input_tokens":1200,"output_tokens":300,"cache_read_input_tokens":4000,"cache_creation_input_tokens":500},"content":[{"name":"Subagent"}]}}`
	sess, _, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "cursor" || sess.TotalUsage.InputTokens != 1200 || sess.TotalUsage.OutputTokens != 300 || sess.TotalUsage.CacheReadTokens != 4000 || sess.TotalUsage.CacheCreationTokens != 500 {
		t.Errorf("unexpected cursor session: %+v", sess)
	}
}

func TestOpenCodeParser(t *testing.T) {
	p := NewOpenCodeParser()
	if !p.Detect("/Users/dev/.local/share/opencode/sessions/sess.jsonl") {
		t.Errorf("detection failed for opencode path")
	}

	jsonl := `{"type":"step-finish","model":"claude-sonnet-4-6","tokens":{"input":1500,"output":250,"cache":{"read":5000,"write":600}}}`
	sess, _, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "opencode" || sess.TotalUsage.InputTokens != 1500 || sess.TotalUsage.OutputTokens != 250 || sess.TotalUsage.CacheReadTokens != 5000 || sess.TotalUsage.CacheCreationTokens != 600 {
		t.Errorf("unexpected opencode session: %+v", sess)
	}
}

func TestGrokParser(t *testing.T) {
	p := NewGrokParser()
	if !p.Detect("/Users/dev/.grok/sessions/proj/s1/signals.json") {
		t.Errorf("detection failed for grok path")
	}

	jsonDoc := `{"contextTokensUsed":45200,"toolsUsed":["read_file","spawn_subagent"],"modelsUsed":["grok-build"]}`
	sess, _, err := p.Parse(strings.NewReader(jsonDoc), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "grok" || sess.TotalUsage.InputTokens != 45200 || sess.Model != "grok-build" {
		t.Errorf("unexpected grok session: %+v", sess)
	}
}

func TestGrokUnifiedLogAggregation(t *testing.T) {
	ResetGrokLogCache()
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "unified.jsonl")

	logContent := `{"msg":"noise","sid":"s1"}
{"msg":"shell.turn.inference_done","sid":"s1","ctx":{"prompt_tokens":100,"cached_prompt_tokens":20,"completion_tokens":10,"reasoning_tokens":7}}
{"msg":"shell.turn.inference_done","sid":"s1","ctx":{"prompt_tokens":250000,"cached_prompt_tokens":200000,"completion_tokens":50,"reasoning_tokens":40}}
{"msg":"shell.turn.inference_done","sid":"other","ctx":{"prompt_tokens":10,"cached_prompt_tokens":0,"completion_tokens":1,"reasoning_tokens":0}}
`
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write log: %v", err)
	}

	bySID, err := LoadGrokUnifiedLog(logPath)
	if err != nil {
		t.Fatalf("LoadGrokUnifiedLog failed: %v", err)
	}

	row, found := bySID["s1"]
	if !found {
		t.Fatalf("expected s1 in parsed log")
	}

	// Turn 1: uncached = 100 - 20 = 80, cached = 20, out = 10, reasoning = 7
	// Turn 2: uncached = 250000 - 200000 = 50000, cached = 200000, out = 50, reasoning = 40
	// Total: input = 50080, cached = 200020, output = 60, reasoning = 47
	if row.Input != 50080 {
		t.Errorf("expected input tokens 50080, got %d", row.Input)
	}
	if row.Cached != 200020 {
		t.Errorf("expected cached tokens 200020, got %d", row.Cached)
	}
	if row.Output != 60 {
		t.Errorf("expected output tokens 60, got %d", row.Output)
	}
	if row.Reasoning != 47 {
		t.Errorf("expected reasoning tokens 47, got %d", row.Reasoning)
	}
	if len(row.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(row.Turns))
	}
	if row.Turns[0].PromptTokens != 100 || row.Turns[0].CachedTokens != 20 || row.Turns[0].CompletionTokens != 10 {
		t.Errorf("unexpected turn 0: %+v", row.Turns[0])
	}
	if row.Turns[1].PromptTokens != 250000 || row.Turns[1].CachedTokens != 200000 || row.Turns[1].CompletionTokens != 50 {
		t.Errorf("unexpected turn 1: %+v", row.Turns[1])
	}

	if _, ok := bySID["other"]; !ok {
		t.Errorf("expected 'other' sid to be present")
	}
}

func TestGrokStatCaching(t *testing.T) {
	ResetGrokLogCache()
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "unified.jsonl")

	log1 := `{"msg":"shell.turn.inference_done","sid":"s1","ctx":{"prompt_tokens":100,"cached_prompt_tokens":10,"completion_tokens":5}}` + "\n"
	if err := os.WriteFile(logPath, []byte(log1), 0644); err != nil {
		t.Fatalf("failed to write log: %v", err)
	}

	res1, err := LoadGrokUnifiedLog(logPath)
	if err != nil {
		t.Fatalf("LoadGrokUnifiedLog failed: %v", err)
	}
	if res1["s1"].Input != 90 {
		t.Errorf("expected input 90, got %d", res1["s1"].Input)
	}

	// Verify second read returns cached map
	res2, err := LoadGrokUnifiedLog(logPath)
	if err != nil {
		t.Fatalf("LoadGrokUnifiedLog second call failed: %v", err)
	}
	if res2["s1"].Input != 90 {
		t.Errorf("expected cached input 90, got %d", res2["s1"].Input)
	}

	// Modify file
	time.Sleep(10 * time.Millisecond)
	log2 := log1 + `{"msg":"shell.turn.inference_done","sid":"s1","ctx":{"prompt_tokens":200,"cached_prompt_tokens":20,"completion_tokens":10}}` + "\n"
	if err := os.WriteFile(logPath, []byte(log2), 0644); err != nil {
		t.Fatalf("failed to update log: %v", err)
	}

	res3, err := LoadGrokUnifiedLog(logPath)
	if err != nil {
		t.Fatalf("LoadGrokUnifiedLog third call failed: %v", err)
	}
	// 90 + 180 = 270
	if res3["s1"].Input != 270 {
		t.Errorf("expected updated input 270, got %d", res3["s1"].Input)
	}
}

func TestGrokParserWithUnifiedLog(t *testing.T) {
	ResetGrokLogCache()
	tmpDir := t.TempDir()
	sid := "01a0test-0000-0000-0000-000000000001"

	grokRoot := filepath.Join(tmpDir, ".grok")
	sessDir := filepath.Join(grokRoot, "sessions", "%2Ftmp%2Fx", sid)
	logsDir := filepath.Join(grokRoot, "logs")
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatalf("failed to create sess dir: %v", err)
	}
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("failed to create logs dir: %v", err)
	}

	summaryPath := filepath.Join(sessDir, "summary.json")
	summaryDoc := `{"created_at":"2026-08-21T00:00:00Z","updated_at":"2026-08-21T00:01:00Z","generated_title":"sess 01a0test","current_model_id":"grok-4.6","info":{"cwd":"/tmp/x"}}`
	if err := os.WriteFile(summaryPath, []byte(summaryDoc), 0644); err != nil {
		t.Fatalf("failed to write summary: %v", err)
	}

	signalsPath := filepath.Join(sessDir, "signals.json")
	signalsDoc := `{"contextTokensUsed":9999,"toolsUsed":["read_file"],"modelsUsed":["grok-4.6"]}`
	if err := os.WriteFile(signalsPath, []byte(signalsDoc), 0644); err != nil {
		t.Fatalf("failed to write signals: %v", err)
	}

	logPath := filepath.Join(logsDir, "unified.jsonl")
	logDoc := `{"msg":"shell.turn.inference_done","sid":"` + sid + `","ctx":{"prompt_tokens":100,"cached_prompt_tokens":20,"completion_tokens":10}}
{"msg":"shell.turn.inference_done","sid":"` + sid + `","ctx":{"prompt_tokens":250000,"cached_prompt_tokens":200000,"completion_tokens":50}}
`
	if err := os.WriteFile(logPath, []byte(logDoc), 0644); err != nil {
		t.Fatalf("failed to write unified log: %v", err)
	}

	f, err := os.Open(summaryPath)
	if err != nil {
		t.Fatalf("failed to open summary: %v", err)
	}
	defer f.Close()

	p := NewGrokParserWithLog(logPath)
	sess, _, err := p.Parse(f, 0)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if sess.TotalUsage.InputTokens != 50080 {
		t.Errorf("expected billed input 50080, got %d", sess.TotalUsage.InputTokens)
	}
	if sess.TotalUsage.CacheReadTokens != 200020 {
		t.Errorf("expected cached 200020, got %d", sess.TotalUsage.CacheReadTokens)
	}
	if sess.TotalUsage.OutputTokens != 60 {
		t.Errorf("expected output 60, got %d", sess.TotalUsage.OutputTokens)
	}
	if sess.SessionID != sid {
		t.Errorf("expected session id %s, got %s", sid, sess.SessionID)
	}
	if len(sess.Turns) != 2 {
		t.Errorf("expected 2 turns, got %d", len(sess.Turns))
	}
}

func TestGrokParserFallbackWithoutLog(t *testing.T) {
	ResetGrokLogCache()
	tmpDir := t.TempDir()
	sid := "01a0test-0000-0000-0000-000000000002"

	sessDir := filepath.Join(tmpDir, ".grok", "sessions", "%2Ftmp%2Fx", sid)
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatalf("failed to create sess dir: %v", err)
	}

	signalsPath := filepath.Join(sessDir, "signals.json")
	signalsDoc := `{"contextTokensUsed":1500,"toolsUsed":["read_file"],"modelsUsed":["grok-4.6"]}`
	if err := os.WriteFile(signalsPath, []byte(signalsDoc), 0644); err != nil {
		t.Fatalf("failed to write signals: %v", err)
	}

	f, err := os.Open(signalsPath)
	if err != nil {
		t.Fatalf("failed to open signals: %v", err)
	}
	defer f.Close()

	p := NewGrokParserWithLog(filepath.Join(tmpDir, "missing.jsonl"))
	sess, _, err := p.Parse(f, 0)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if sess.TotalUsage.InputTokens != 1500 {
		t.Errorf("expected fallback input 1500, got %d", sess.TotalUsage.InputTokens)
	}
	if sess.TotalUsage.OutputTokens != 0 || sess.TotalUsage.CacheReadTokens != 0 {
		t.Errorf("expected output/cached 0 on fallback, got %+v", sess.TotalUsage)
	}
}

func TestPiParser(t *testing.T) {
	p := NewPiParser()
	if !p.Detect("/Users/dev/.pi/agent/sessions/proj/s1.jsonl") {
		t.Errorf("detection failed for pi path")
	}

	jsonl := `{"type":"session","id":"pi-1","timestamp":"2026-07-05T07:40:17Z"}
{"type":"message","timestamp":"2026-07-05T07:41:00Z","message":{"role":"assistant","model":"zai-glm-4.7","usage":{"input":2965,"output":185,"cacheRead":100,"cacheWrite":50}}}`
	sess, _, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "pi" || sess.TotalUsage.InputTokens != 2965 || sess.TotalUsage.OutputTokens != 185 || sess.TotalUsage.CacheReadTokens != 100 || sess.TotalUsage.CacheCreationTokens != 50 {
		t.Errorf("unexpected pi session: %+v", sess)
	}
}

func TestMetaMuseParser(t *testing.T) {
	p := NewMetaMuseParser()
	if !p.Detect("/Users/dev/.muse/sessions/proj/s1/session.jsonl") {
		t.Errorf("detection failed for muse path")
	}

	jsonl := `{"recorded_at":1786806413737000,"payload":{"event":{"model":"llama-3.3-70b-versatile","usage":{"input_tokens":500,"output_tokens":100,"cache_read_tokens":200,"cache_write_tokens":50}}}}`
	sess, _, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "muse" || sess.TotalUsage.InputTokens != 500 || sess.TotalUsage.OutputTokens != 100 {
		t.Errorf("unexpected muse session: %+v", sess)
	}
}

func TestPrimeParser(t *testing.T) {
	p := NewPrimeParser()
	if !p.Detect("/Users/dev/.prime/sessions/s1.jsonl") {
		t.Errorf("detection failed for prime path")
	}

	jsonl := `{"id":"p-1","parentId":"parent-0","model":"claude-3-5-sonnet","usage":{"input_tokens":400,"output_tokens":80,"cache_read_tokens":150,"cache_creation_tokens":30}}`
	sess, _, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "prime" || sess.TotalUsage.InputTokens != 400 || sess.TotalUsage.OutputTokens != 80 || !sess.IsSubagent || sess.ParentSessionID != "parent-0" {
		t.Errorf("unexpected prime session: %+v", sess)
	}
}

func TestQwenParser(t *testing.T) {
	p := NewQwenParser()
	if !p.Detect("/Users/dev/.qwen/projects/p1/chats/s1.jsonl") {
		t.Errorf("detection failed for qwen path")
	}

	jsonl := `{"role":"assistant","model":"qwen-2.5-coder","usage":{"input_tokens":750,"output_tokens":150,"cache_read_input_tokens":300,"cache_creation_input_tokens":40}}`
	sess, _, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "qwen" || sess.TotalUsage.InputTokens != 750 || sess.TotalUsage.OutputTokens != 150 {
		t.Errorf("unexpected qwen session: %+v", sess)
	}
}

func TestClineParser(t *testing.T) {
	p := NewClineParser()
	if !p.Detect("/Users/dev/.cline/data/taskHistory.json") {
		t.Errorf("detection failed for cline path")
	}

	jsonDoc := `[{"id":"cline-1","ts":1781420401000,"totalCost":0.05,"tokensIn":300,"tokensOut":60,"cacheReads":100,"cacheWrites":20,"model":"claude-3-5-sonnet"}]`
	sess, _, err := p.Parse(strings.NewReader(jsonDoc), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "cline" || sess.TotalUsage.InputTokens != 300 || sess.TotalUsage.OutputTokens != 60 {
		t.Errorf("unexpected cline session: %+v", sess)
	}
}

func TestSmallCodeParser(t *testing.T) {
	p := NewSmallCodeParser()
	if !p.Detect("/Users/dev/proj/.smallcode/traces/trace1.json") {
		t.Errorf("detection failed for smallcode path")
	}

	jsonDoc := `{"id":"sc-1","model":"nemotron-3-nano:4b","prompt":"test","tokens":{"prompt":8331,"completion":185}}`
	sess, _, err := p.Parse(strings.NewReader(jsonDoc), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "smallcode" || sess.TotalUsage.InputTokens != 8331 || sess.TotalUsage.OutputTokens != 185 {
		t.Errorf("unexpected smallcode session: %+v", sess)
	}
}

func TestVibeParser(t *testing.T) {
	p := NewVibeParser()
	if !p.Detect("/Users/dev/.vibe/logs/session/s1.json") {
		t.Errorf("detection failed for vibe path")
	}

	jsonDoc := `{"id":"v-1","metadata":{"stats":{"session_prompt_tokens":900,"session_completion_tokens":200,"context_tokens":150}}}`
	sess, _, err := p.Parse(strings.NewReader(jsonDoc), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "vibe" || sess.TotalUsage.InputTokens != 900 || sess.TotalUsage.OutputTokens != 200 || sess.TotalUsage.CacheReadTokens != 150 {
		t.Errorf("unexpected vibe session: %+v", sess)
	}
}

func TestWindsurfParser(t *testing.T) {
	p := NewWindsurfParser()
	if !p.Detect("/Users/dev/windsurf/logs/session.json") {
		t.Errorf("detection failed for windsurf path")
	}

	jsonDoc := `{"sessionId":"ws-1","model":"claude-3-5-sonnet","turns":[{"role":"assistant","input_tokens":600,"output_tokens":120}]}`
	sess, _, err := p.Parse(strings.NewReader(jsonDoc), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "windsurf" || sess.TotalUsage.InputTokens != 600 || sess.TotalUsage.OutputTokens != 120 {
		t.Errorf("unexpected windsurf session: %+v", sess)
	}
}

func TestOllamaParser(t *testing.T) {
	p := NewOllamaParser()
	if !p.Detect("/Users/dev/local-models/ollama/trace.json") {
		t.Errorf("detection failed for ollama path")
	}

	jsonDoc := `{"id":"ol-1","model":"llama3:8b","usage":{"prompt_tokens":400,"completion_tokens":80,"total_tokens":480}}`
	sess, _, err := p.Parse(strings.NewReader(jsonDoc), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "ollama" || sess.TotalUsage.InputTokens != 400 || sess.TotalUsage.OutputTokens != 80 {
		t.Errorf("unexpected ollama session: %+v", sess)
	}
}

func TestRichTurnExtractionAcrossAgents(t *testing.T) {
	// 1. Cursor rich turn
	cursorP := NewCursorParser()
	cursorJSON := `{"role":"user","content":"Implement auth"}
{"role":"assistant","message":{"model":"claude-3-5-sonnet","content":[{"type":"text","text":"I will create auth middleware"},{"name":"ast_grep_search","input":{"pattern":"func Auth"}}]}}`
	cSess, _, err := cursorP.Parse(strings.NewReader(cursorJSON), 0)
	if err != nil {
		t.Fatalf("cursor parse failed: %v", err)
	}
	if len(cSess.Turns) != 2 || cSess.Turns[0].Content != "Implement auth" || cSess.Turns[1].Content != "I will create auth middleware" {
		t.Errorf("unexpected cursor turns: %+v", cSess.Turns)
	}
	if len(cSess.Turns[1].ToolCalls) != 1 || cSess.Turns[1].ToolCalls[0].Name != "ast_grep_search" {
		t.Errorf("unexpected cursor tool calls: %+v", cSess.Turns[1].ToolCalls)
	}

	// 2. OpenCode rich turn
	openCodeP := NewOpenCodeParser()
	openCodeJSON := `{"type":"user","content":"Fix database schema"}
{"type":"step-finish","model":"claude-sonnet-4-6","data":{"content":"Updated migration file","name":"replace_file_content","args":{"file":"0005.sql"}},"tokens":{"input":1200,"output":300}}`
	ocSess, _, err := openCodeP.Parse(strings.NewReader(openCodeJSON), 0)
	if err != nil {
		t.Fatalf("opencode parse failed: %v", err)
	}
	if len(ocSess.Turns) != 2 || ocSess.Turns[0].Content != "Fix database schema" || ocSess.Turns[1].Content != "Updated migration file" {
		t.Errorf("unexpected opencode turns: %+v", ocSess.Turns)
	}
	if len(ocSess.Turns[1].ToolCalls) != 1 || ocSess.Turns[1].ToolCalls[0].Name != "replace_file_content" {
		t.Errorf("unexpected opencode tool calls: %+v", ocSess.Turns[1].ToolCalls)
	}

	// 3. Codex rich turn
	codexP := NewCodexParser()
	codexJSON := `{"type":"session_meta","payload":{"id":"sess-codex-rich"}}
{"type":"event_msg","timestamp":"2026-06-10T12:00:00Z","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":500,"output_tokens":150,"reasoning_output_tokens":80,"total_tokens":650}}}}
{"type":"response_item","payload":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}}`
	codexSess, _, err := codexP.Parse(strings.NewReader(codexJSON), 0)
	if err != nil {
		t.Fatalf("codex parse failed: %v", err)
	}
	if len(codexSess.Turns) != 1 || codexSess.Turns[0].ReasoningEffort != "medium" {
		t.Errorf("unexpected codex turns: %+v", codexSess.Turns)
	}
	if len(codexSess.Turns[0].ToolCalls) != 1 || codexSess.Turns[0].ToolCalls[0].Name != "read_file" {
		t.Errorf("unexpected codex tool calls: %+v", codexSess.Turns[0].ToolCalls)
	}

	// 4. Copilot rich turn
	copilotP := NewCopilotParser()
	copilotJSON := `{"version":1,"creationDate":1786806413737,"requests":[{"modelId":"gpt-4o","timestamp":1786806413737,"completionTokens":120,"message":{"text":"Here is your solution in Go."}}]}`
	copilotSess, _, err := copilotP.Parse(strings.NewReader(copilotJSON), 0)
	if err != nil {
		t.Fatalf("copilot parse failed: %v", err)
	}
	if len(copilotSess.Turns) != 1 || copilotSess.Turns[0].Content != "Here is your solution in Go." {
		t.Errorf("unexpected copilot turns: %+v", copilotSess.Turns)
	}
}
