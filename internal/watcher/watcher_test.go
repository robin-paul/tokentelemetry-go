package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/pricing"
	"github.com/robin-paul/tokentelemetry-go/internal/scanner"
	"github.com/robin-paul/tokentelemetry-go/internal/store"
)

func setupTestDB(t *testing.T) (*store.DB, func()) {
	tmpDir, err := os.MkdirTemp("", "tokentelemetry_watcher_test_*")
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

func TestWatcherAndReconcilerIntegration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	pe, err := pricing.NewEngine()
	if err != nil {
		t.Fatalf("failed to init pricing engine: %v", err)
	}

	eng := scanner.NewEngine(db, pe, scanner.Config{
		WorkerPoolSize: 2,
		BatchTimeout:   20 * time.Millisecond,
		BatchSize:      5,
	})

	ctx := context.Background()
	eng.Start(ctx)
	defer eng.Stop()

	// 1. Create temporary root folder for watching
	tmpDir, err := os.MkdirTemp("", "watcher_root_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	w, err := NewWatcher(eng, Config{
		DebounceDuration: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	if err := w.AddRoot(tmpDir); err != nil {
		t.Fatalf("failed to add root: %v", err)
	}

	w.Start(ctx)
	defer func() { _ = w.Stop() }()

	// 2. Create subfolder and write a transcript file
	claudeDir := filepath.Join(tmpDir, ".claude", "projects", "proj1")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create claude dir: %v", err)
	}

	// Give watcher time to register new directory
	time.Sleep(50 * time.Millisecond)

	sessionFile := filepath.Join(claudeDir, "sess_watcher.jsonl")
	content := `{"type":"assistant","sessionId":"sess-watch-1","timestamp":"2026-06-10T12:00:00Z","message":{"model":"claude-3-7-sonnet","usage":{"input_tokens":500,"output_tokens":100,"cache_read_input_tokens":200,"cache_creation_input_tokens":50},"content":[{"type":"tool_use","name":"bash"}]}}`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write session file: %v", err)
	}

	// Wait for debounce and batch commit
	time.Sleep(200 * time.Millisecond)

	// 3. Verify Reconciler
	rec := NewReconciler(eng, ReconcilerConfig{
		Interval: 100 * time.Millisecond,
		Roots:    []string{tmpDir},
	})
	rec.Start(ctx)
	defer rec.Stop()

	if err := rec.Sweep(ctx); err != nil {
		t.Fatalf("reconciler sweep failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify persistence in SQLite
	saved, err := db.GetSessionDetail(ctx, "claude:sess-watch-1")
	if err != nil {
		t.Fatalf("failed to find session from watcher/reconciler: %v", err)
	}
	if saved.InputTokens != 500 || saved.OutputTokens != 100 {
		t.Errorf("token count mismatch: %+v", saved)
	}
	if len(saved.Turns) != 1 {
		t.Errorf("expected 1 turn, got %d", len(saved.Turns))
	}
}
