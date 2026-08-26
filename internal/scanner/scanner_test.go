package scanner

import (
	"context"
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
