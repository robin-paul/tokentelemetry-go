package store_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
	"github.com/robin-paul/tokentelemetry-go/internal/store"
)

func setupTestDB(t *testing.T) (*store.DB, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "tokentelemetry_store_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := store.Open(dbPath)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		t.Fatalf("failed to open test db: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Migrate(ctx); err != nil {
		_ = db.Close()
		_ = os.RemoveAll(tempDir)
		t.Fatalf("failed to run migrations: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
		_ = os.RemoveAll(tempDir)
	}

	return db, cleanup
}

func TestMigrations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	// Running migrate again should be idempotent
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("second migration call failed: %v", err)
	}
}

func TestSessionsCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	s := &models.Session{
		ID:                  "test-sess-1",
		SessionID:           "claude-sess-100",
		AgentName:           "claude_code",
		ProjectName:         "token-analyzer",
		FilePath:            "/path/to/claude/log.jsonl",
		StartTime:           time.Now().Add(-1 * time.Hour),
		EndTime:             time.Now(),
		DurationSeconds:     3600,
		ModelRaw:            "claude-3-7-sonnet-20250219",
		ModelResolved:       "claude-3-7-sonnet",
		InputTokens:         5000,
		OutputTokens:        1500,
		CacheReadTokens:     2000,
		CacheCreationTokens: 500,
		GrossCostUSD:        0.05,
		NetCostUSD:          0.035,
		ElectricityCostUSD:  0.0001,
		HardwareProfile:     "apple_m3_max",
		Status:              "completed",
		GitBranch:           "feature/go-port",
		IsSubagent:          false,
	}

	// 1. Insert session
	if err := db.UpsertSession(ctx, s); err != nil {
		t.Fatalf("failed to insert session: %v", err)
	}

	// 2. Fetch session
	fetched, err := db.GetSession(ctx, s.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if fetched.AgentName != s.AgentName || fetched.InputTokens != s.InputTokens {
		t.Errorf("mismatched session data: got %+v, want %+v", fetched, s)
	}

	// 3. Save with turns and subagents
	s.Turns = []models.MessageTurn{
		{
			ID:                  uuid.NewString(),
			SessionID:           s.ID,
			TurnIndex:           0,
			Timestamp:           time.Now().Add(-30 * time.Minute),
			Role:                "user",
			ModelName:           "claude-3-7-sonnet",
			InputTokens:         1000,
			OutputTokens:        200,
			CostUSD:             0.005,
			ToolsInvoked:        []string{"search_web"},
		},
		{
			ID:                  uuid.NewString(),
			SessionID:           s.ID,
			TurnIndex:           1,
			Timestamp:           time.Now(),
			Role:                "assistant",
			ModelName:           "claude-3-7-sonnet",
			InputTokens:         4000,
			OutputTokens:        1300,
			CostUSD:             0.03,
			ToolsInvoked:        []string{"run_command", "view_file"},
		},
	}

	childSessID := "child-sess-1"
	childS := &models.Session{
		ID:              childSessID,
		SessionID:       "child-claude-1",
		AgentName:       "claude_code",
		ProjectName:     "token-analyzer",
		FilePath:        "/path/to/child/log.jsonl",
		StartTime:       time.Now(),
		EndTime:         time.Now(),
		ModelRaw:        "claude-3-5-haiku",
		ModelResolved:   "claude-3-5-haiku",
		IsSubagent:      true,
		ParentSessionID: s.ID,
		SubagentType:    "research",
	}
	if err := db.UpsertSession(ctx, childS); err != nil {
		t.Fatalf("failed to insert child session: %v", err)
	}

	s.SubagentRuns = []models.SubagentRun{
		{
			ID:              uuid.NewString(),
			ParentSessionID: s.ID,
			ChildSessionID:  childSessID,
			AgentType:       "research",
			Tokens:          1200,
			CostUSD:         0.002,
			CreatedAt:       time.Now(),
		},
	}

	if err := db.SaveSessionWithTurnsAndSubagents(ctx, s); err != nil {
		t.Fatalf("failed to save session with turns and subagents: %v", err)
	}

	// 4. Fetch detail
	detail, err := db.GetSessionDetail(ctx, s.ID)
	if err != nil {
		t.Fatalf("failed to get session detail: %v", err)
	}
	if len(detail.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(detail.Turns))
	}
	if len(detail.Turns[0].ToolsInvoked) != 1 || detail.Turns[0].ToolsInvoked[0] != "search_web" {
		t.Errorf("unexpected tools invoked: %v", detail.Turns[0].ToolsInvoked)
	}
	if len(detail.SubagentRuns) != 1 {
		t.Fatalf("expected 1 subagent run, got %d", len(detail.SubagentRuns))
	}

	// 5. List with filters
	list, total, err := db.ListSessions(ctx, models.FilterParams{
		Agent:   "claude_code",
		Project: "token-analyzer",
	})
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Errorf("expected 2 sessions, got total %d, len %d", total, len(list))
	}

	// 6. Delete session and cascade check
	if err := db.DeleteSession(ctx, s.ID); err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}
	_, err = db.GetSession(ctx, s.ID)
	if err != store.ErrNotFound {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}
	turns, _ := db.GetMessageTurns(ctx, s.ID)
	if len(turns) != 0 {
		t.Errorf("expected turns to cascade delete, got %d", len(turns))
	}
}

func TestSummariesAndLeaderboard(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Insert sessions
	s1 := &models.Session{
		ID:            "sess-1",
		SessionID:     "s1",
		AgentName:     "claude_code",
		ProjectName:   "proj-a",
		FilePath:      "/f1",
		StartTime:     time.Now().Add(-2 * time.Hour),
		EndTime:       time.Now().Add(-1 * time.Hour),
		ModelRaw:      "claude-3-7-sonnet",
		ModelResolved: "claude-3-7-sonnet",
		InputTokens:   10000,
		OutputTokens:  2000,
		GrossCostUSD:  0.06,
		NetCostUSD:    0.05,
	}
	s2 := &models.Session{
		ID:            "sess-2",
		SessionID:     "s2",
		AgentName:     "gemini_cli",
		ProjectName:   "proj-b",
		FilePath:      "/f2",
		StartTime:     time.Now().Add(-1 * time.Hour),
		EndTime:       time.Now(),
		ModelRaw:      "gemini-2.5-pro",
		ModelResolved: "gemini-2.5-pro",
		InputTokens:   20000,
		OutputTokens:  5000,
		GrossCostUSD:  0.10,
		NetCostUSD:    0.08,
	}
	_ = db.UpsertSession(ctx, s1)
	_ = db.UpsertSession(ctx, s2)

	// 2. Stats Overview
	stats, err := db.GetStatsOverview(ctx, "", "", "", "")
	if err != nil {
		t.Fatalf("failed to get stats overview: %v", err)
	}
	if stats.TotalSessions != 2 || stats.TotalTokens != 37000 {
		t.Errorf("unexpected stats overview: %+v", stats)
	}
	if stats.ActiveAgents != 2 || stats.ActiveProjects != 2 {
		t.Errorf("unexpected active counts: agents=%d, projects=%d", stats.ActiveAgents, stats.ActiveProjects)
	}

	// 3. Leaderboard
	modelLeader, agentLeader, err := db.GetLeaderboard(ctx, 5, "", "")
	if err != nil {
		t.Fatalf("failed to get leaderboard: %v", err)
	}
	if len(modelLeader) != 2 || modelLeader[0].Name != "gemini-2.5-pro" {
		t.Errorf("unexpected model leaderboard: %+v", modelLeader)
	}
	if len(agentLeader) != 2 || agentLeader[0].Name != "gemini_cli" {
		t.Errorf("unexpected agent leaderboard: %+v", agentLeader)
	}

	// 4. Projects
	projects, err := db.GetProjects(ctx)
	if err != nil {
		t.Fatalf("failed to get projects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	// 5. Daily Summary upsert & query
	ds := &models.DailySummary{
		Date:                     "2026-08-21",
		AgentName:                "claude_code",
		ProjectName:              "proj-a",
		ModelName:                "claude-3-7-sonnet",
		TotalSessions:            1,
		TotalInputTokens:         10000,
		TotalOutputTokens:        2000,
		TotalCacheReadTokens:     500,
		TotalCacheCreationTokens: 100,
		TotalCostUSD:             0.05,
		TotalDurationSeconds:     1800,
	}
	if err := db.UpsertDailySummary(ctx, ds); err != nil {
		t.Fatalf("failed to upsert daily summary: %v", err)
	}
	summaries, err := db.QueryDailySummaries(ctx, "2026-08-01", "2026-08-31", "", "", "")
	if err != nil {
		t.Fatalf("failed to query daily summaries: %v", err)
	}
	if len(summaries) != 1 || summaries[0].TotalSessions != 1 {
		t.Errorf("unexpected daily summaries result: %+v", summaries)
	}
}

func TestScannerCheckpointsAndPricingOverrides(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Checkpoint
	cp := &store.ScannerCheckpoint{
		FilePath:     "/var/logs/agent.jsonl",
		LastModified: time.Now().UTC().Truncate(time.Second),
		FileSize:     10240,
		ByteOffset:   5120,
		LineNumber:   100,
		FileHash:     "sha256:abcd1234ef",
	}
	if err := db.UpsertCheckpoint(ctx, cp); err != nil {
		t.Fatalf("failed to upsert checkpoint: %v", err)
	}
	fetchedCP, err := db.GetCheckpoint(ctx, cp.FilePath)
	if err != nil {
		t.Fatalf("failed to get checkpoint: %v", err)
	}
	if fetchedCP.ByteOffset != cp.ByteOffset || fetchedCP.FileHash != cp.FileHash {
		t.Errorf("mismatched checkpoint: got %+v, want %+v", fetchedCP, cp)
	}

	// 2. Pricing Override
	po := &models.PricingOverride{
		ModelPattern:       "custom-gpt-5-preview",
		InputCostPerM:      10.0,
		OutputCostPerM:     30.0,
		CacheReadCostPerM:  2.0,
		CacheWriteCostPerM: 12.5,
		Source:             "user_test",
	}
	if err := db.UpsertPricingOverride(ctx, po); err != nil {
		t.Fatalf("failed to upsert pricing override: %v", err)
	}
	fetchedPO, err := db.GetPricingOverride(ctx, po.ModelPattern)
	if err != nil {
		t.Fatalf("failed to get pricing override: %v", err)
	}
	if fetchedPO.InputCostPerM != po.InputCostPerM {
		t.Errorf("mismatched pricing override: got %+v, want %+v", fetchedPO, po)
	}
}

func TestWALConcurrency(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	const numWriters = 8
	const numReaders = 16
	const iterations = 25

	var wg sync.WaitGroup
	errChan := make(chan error, (numWriters+numReaders)*iterations)

	// Concurrently write sessions
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				sess := &models.Session{
					ID:            fmt.Sprintf("wal-sess-%d-%d", workerID, i),
					SessionID:     fmt.Sprintf("s-%d-%d", workerID, i),
					AgentName:     "concurrent_agent",
					ProjectName:   "concurrency_test",
					FilePath:      fmt.Sprintf("/log-%d-%d.jsonl", workerID, i),
					StartTime:     time.Now(),
					EndTime:       time.Now(),
					ModelRaw:      "claude-3-7-sonnet",
					ModelResolved: "claude-3-7-sonnet",
					InputTokens:   int64(i * 100),
					OutputTokens:  int64(i * 50),
					Status:        "completed",
				}
				if err := db.UpsertSession(ctx, sess); err != nil {
					errChan <- fmt.Errorf("writer %d iter %d error: %w", workerID, i, err)
					return
				}
				time.Sleep(2 * time.Millisecond)
			}
		}(w)
	}

	// Concurrently read sessions and overview stats
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_, _, err := db.ListSessions(ctx, models.FilterParams{
					Agent: "concurrent_agent",
					Limit: 10,
				})
				if err != nil {
					errChan <- fmt.Errorf("reader %d iter %d error: %w", readerID, i, err)
					return
				}
				_, err = db.GetStatsOverview(ctx, "", "", "", "")
				if err != nil {
					errChan <- fmt.Errorf("reader %d stats error: %w", readerID, err)
					return
				}
				time.Sleep(1 * time.Millisecond)
			}
		}(r)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		t.Fatalf("WAL concurrency test had %d errors: %v", len(errs), errs[0])
	}
}
