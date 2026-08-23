package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

func TestIngestStoreOperations(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "ingest_test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	sess1 := models.Session{
		ID:              "sess_1",
		SessionID:       "s1",
		AgentName:       "claude_code",
		ProjectName:     "proj-a",
		FilePath:        "/logs/s1.jsonl",
		MachineID:       "machine-alpha",
		StartTime:       now,
		EndTime:         now.Add(5 * time.Minute),
		DurationSeconds: 300,
		ModelRaw:        "claude-3-7-sonnet",
		ModelResolved:   "claude-3-7-sonnet",
		InputTokens:     1000,
		OutputTokens:    500,
		NetCostUSD:      0.05,
		Turns: []models.MessageTurn{
			{
				ID:          "turn_1",
				SessionID:   "sess_1",
				TurnIndex:   0,
				Timestamp:   now,
				Role:        "assistant",
				ModelName:   "claude-3-7-sonnet",
				InputTokens: 1000,
				OutputTokens: 500,
				CostUSD:     0.05,
			},
		},
	}

	// 1. Initial save
	if err := db.SaveSessionWithTurnsAndSubagents(ctx, &sess1); err != nil {
		t.Fatalf("SaveSessionWithTurnsAndSubagents failed: %v", err)
	}

	// 2. Query GetSession and check MachineID
	saved, err := db.GetSessionDetail(ctx, "sess_1")
	if err != nil {
		t.Fatalf("GetSessionDetail failed: %v", err)
	}
	if saved.MachineID != "machine-alpha" {
		t.Errorf("expected MachineID 'machine-alpha', got %q", saved.MachineID)
	}
	if len(saved.Turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(saved.Turns))
	}

	// 3. Test GetExistingSessionIDs
	testSessions := []models.Session{
		{ID: "sess_1"},
		{ID: "sess_non_existent"},
	}
	existing, err := db.GetExistingSessionIDs(ctx, testSessions)
	if err != nil {
		t.Fatalf("GetExistingSessionIDs failed: %v", err)
	}
	if !existing["sess_1"] {
		t.Errorf("expected sess_1 to exist")
	}
	if existing["sess_non_existent"] {
		t.Errorf("expected sess_non_existent to not exist")
	}

	// 4. Update with turn replacement (idempotent update with new turn)
	sess1Updated := sess1
	sess1Updated.InputTokens = 2000
	sess1Updated.Turns = []models.MessageTurn{
		sess1.Turns[0],
		{
			ID:          "turn_2",
			SessionID:   "sess_1",
			TurnIndex:   1,
			Timestamp:   now.Add(2 * time.Minute),
			Role:        "assistant",
			ModelName:   "claude-3-7-sonnet",
			InputTokens: 1000,
			OutputTokens: 200,
			CostUSD:     0.02,
		},
	}
	if err := db.SaveSessionWithTurnsAndSubagents(ctx, &sess1Updated); err != nil {
		t.Fatalf("SaveSessionWithTurnsAndSubagents update failed: %v", err)
	}

	savedUpdated, err := db.GetSessionDetail(ctx, "sess_1")
	if err != nil {
		t.Fatalf("GetSessionDetail failed: %v", err)
	}
	if savedUpdated.InputTokens != 2000 {
		t.Errorf("expected updated input tokens 2000, got %d", savedUpdated.InputTokens)
	}
	if len(savedUpdated.Turns) != 2 {
		t.Fatalf("expected 2 turns after update, got %d", len(savedUpdated.Turns))
	}
}
