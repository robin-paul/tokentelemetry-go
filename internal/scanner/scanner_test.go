package scanner

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
	"github.com/robin-paul/tokentelemetry-go/internal/pricing"
	"github.com/robin-paul/tokentelemetry-go/internal/store"
)

func setupTestDB(t *testing.T) (*store.DB, func()) {
	tmpDir, err := os.MkdirTemp("", "tokentelemetry_scanner_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestScannerEndToEnd(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pe, err := pricing.NewEngine()
	if err != nil {
		t.Fatalf("failed to initialize pricing engine: %v", err)
	}

	engine := NewEngine(db, pe, Config{
		WorkerPoolSize: 2,
		BatchTimeout:   20 * time.Millisecond,
		BatchSize:      10,
	})

	ctx := context.Background()
	engine.Start(ctx)
	defer engine.Stop()

	// Create a temporary mock Claude session file
	tmpDir, err := os.MkdirTemp("", "claude_scan_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	claudeProjDir := filepath.Join(tmpDir, ".claude", "projects", "myproject")
	if err := os.MkdirAll(claudeProjDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	sessionFile := filepath.Join(claudeProjDir, "sess_123.jsonl")
	content := `{"type":"assistant","sessionId":"sess-123","timestamp":"2026-06-10T12:00:00Z","message":{"model":"claude-3-7-sonnet","usage":{"input_tokens":1000,"output_tokens":200,"cache_read_input_tokens":500,"cache_creation_input_tokens":100},"content":[{"type":"tool_use","name":"read_file"}]}}`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write session file: %v", err)
	}

	// 1. Scan file directly
	sess, err := engine.ScanFile(ctx, sessionFile)
	if err != nil {
		t.Fatalf("ScanFile failed: %v", err)
	}
	if sess == nil {
		t.Fatalf("expected non-nil session")
	}

	if sess.AgentName != "claude_code" || sess.SessionID != "sess-123" {
		t.Errorf("unexpected session data: %+v", sess)
	}
	if sess.GrossCostUSD <= 0 || sess.NetCostUSD <= 0 {
		t.Errorf("expected positive costs, got gross=%f, net=%f", sess.GrossCostUSD, sess.NetCostUSD)
	}
	if len(sess.Turns) != 1 || len(sess.Turns[0].ToolsInvoked) != 1 {
		t.Errorf("unexpected turns: %+v", sess.Turns)
	}

	// 2. Test Checkpoint logic - second scan should skip
	sess2, err := engine.ScanFile(ctx, sessionFile)
	if err != nil {
		t.Fatalf("ScanFile second call failed: %v", err)
	}
	if sess2 != nil {
		t.Errorf("expected second scan to be skipped by checkpoint, got: %+v", sess2)
	}

	// 3. Test Batch pipeline via EnqueueFile by appending to the file
	time.Sleep(10 * time.Millisecond)
	newLine := "\n" + `{"type":"assistant","sessionId":"sess-123","timestamp":"2026-06-10T12:01:00Z","message":{"model":"claude-3-7-sonnet","usage":{"input_tokens":500,"output_tokens":100,"cache_read_input_tokens":200,"cache_creation_input_tokens":50},"content":[{"type":"tool_use","name":"write_file"}]}}`
	f, err := os.OpenFile(sessionFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open session file: %v", err)
	}
	_, _ = f.WriteString(newLine)
	_ = f.Close()

	engine.EnqueueFile(sessionFile)

	// Verify database persistence with retry poll for slow race CI
	var saved *models.Session
	for i := 0; i < 40; i++ {
		time.Sleep(50 * time.Millisecond)
		s, err := db.GetSessionDetail(ctx, sess.ID)
		if err == nil && s != nil && len(s.Turns) == 2 {
			saved = s
			break
		}
	}

	if saved == nil {
		s, err := db.GetSessionDetail(ctx, sess.ID)
		if err != nil {
			t.Fatalf("failed to get saved session: %v", err)
		}
		saved = s
	}

	if saved.InputTokens != 1500 || saved.OutputTokens != 300 {
		t.Errorf("saved session token mismatch: %+v", saved)
	}
	if len(saved.Turns) != 2 {
		t.Errorf("expected 2 saved turns, got %d", len(saved.Turns))
	}
}

func TestScannerGrokBilledUsageAndTieredCost(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pe, err := pricing.NewEngine()
	if err != nil {
		t.Fatalf("failed to initialize pricing engine: %v", err)
	}

	engine := NewEngine(db, pe, Config{
		WorkerPoolSize: 2,
		BatchTimeout:   20 * time.Millisecond,
		BatchSize:      10,
	})

	ctx := context.Background()

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

	sess, err := engine.ScanFile(ctx, summaryPath)
	if err != nil {
		t.Fatalf("ScanFile failed: %v", err)
	}
	if sess == nil {
		t.Fatalf("expected non-nil session")
	}

	// Turn 1: 80 uncached, 20 cached, 10 completion (<200k)
	// Turn 2: 50,000 uncached, 200,000 cached, 50 completion (>=200k, 2x cliff)
	// Total: input=50080, cached=200020, output=60
	if sess.InputTokens != 50080 {
		t.Errorf("expected input tokens 50080, got %d", sess.InputTokens)
	}
	if sess.CacheReadTokens != 200020 {
		t.Errorf("expected cached tokens 200020, got %d", sess.CacheReadTokens)
	}
	if sess.OutputTokens != 60 {
		t.Errorf("expected output tokens 60, got %d", sess.OutputTokens)
	}
	if sess.InputTokens == 9999 {
		t.Errorf("must not use context footprint 9999 when log is present")
	}

	// Cost verification:
	// Turn 1: 80/1M*$2 + 10/1M*$6 + 20/1M*$0.50 = 0.000160 + 0.000060 + 0.000010 = 0.000230
	// Turn 2: (50000/1M*$2 + 50/1M*$6 + 200000/1M*$0.50) * 2 = (0.10 + 0.0003 + 0.10) * 2 = 0.2003 * 2 = 0.400600
	// Total: 0.000230 + 0.400600 = 0.400830
	expectedCost := 0.400830
	if math.Abs(sess.NetCostUSD-expectedCost) > 1e-4 {
		t.Errorf("expected net cost %.6f, got %.6f", expectedCost, sess.NetCostUSD)
	}
}

