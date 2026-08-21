package parsers

import (
	"strings"
	"testing"
)

func TestDefaultRegistry(t *testing.T) {
	reg := NewDefaultRegistry()
	if len(reg.All()) < 18 {
		t.Fatalf("expected at least 18 parsers, got %d", len(reg.All()))
	}

	expectedAgents := []string{
		"claude_code", "antigravity", "gemini_cli", "codex", "cursor",
		"copilot", "opencode", "hermes", "grok", "pi", "dsh", "muse",
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

	jsonl := `{"type":"assistant","sessionId":"sess-1","timestamp":"2026-06-10T12:00:00Z","message":{"model":"claude-3-7-sonnet","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":1000,"cache_creation_input_tokens":200},"content":[{"type":"tool_use","name":"Skill"}]}}`
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
	if len(sess.Turns) != 1 || len(sess.Turns[0].Tools) != 1 || sess.Turns[0].Tools[0] != "Skill" {
		t.Errorf("unexpected turns: %+v", sess.Turns)
	}
}

func TestAntigravityParser(t *testing.T) {
	p := NewAntigravityParser()
	if !p.Detect("/Users/dev/.gemini/antigravity-cli/brain/session123/transcript.jsonl") {
		t.Errorf("detection failed for antigravity path")
	}

	jsonl := `{"step_index":1,"source":"USER_EXPLICIT","type":"USER_INPUT","content":"Fix the bug","created_at":"2026-06-10T12:00:00Z"}
{"step_index":2,"source":"MODEL","type":"MODEL_RESPONSE","content":"Running fix","created_at":"2026-06-10T12:00:05Z","tool_calls":[{"name":"exec_command"}],"metrics":{"input_tokens":50,"output_tokens":25,"cache_read_tokens":0,"cache_creation_tokens":0}}`
	sess, _, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "antigravity" {
		t.Errorf("expected agent antigravity, got %s", sess.AgentName)
	}
	if sess.TotalUsage.InputTokens != 52 || sess.TotalUsage.OutputTokens != 25 {
		t.Errorf("unexpected usage: %+v", sess.TotalUsage)
	}
	if len(sess.Turns) != 2 {
		t.Errorf("expected 2 turns, got %d", len(sess.Turns))
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

func TestHermesParser(t *testing.T) {
	p := NewHermesParser()
	if !p.Detect("/Users/dev/.hermes/telemetry/sess.jsonl") {
		t.Errorf("detection failed for hermes path")
	}

	jsonl := `{"event":"session_end","session_id":"h-1","model":"hermes-3","input_tokens":500,"output_tokens":100,"cache_read_tokens":200,"cache_write_tokens":50,"end_reason":"completed"}`
	sess, _, err := p.Parse(strings.NewReader(jsonl), 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if sess.AgentName != "hermes" || sess.TotalUsage.InputTokens != 500 || sess.TotalUsage.OutputTokens != 100 || sess.Status != "completed" {
		t.Errorf("unexpected hermes session: %+v", sess)
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
